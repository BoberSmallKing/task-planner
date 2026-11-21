package main

import (
	"database/sql"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

type Task struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartAt     time.Time `json:"startAt"`
	Done        bool      `json:"done"`
	Color       string    `json:"color"`
	CreatedAt   time.Time `json:"createdAt"`
}

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "planner.db"
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		start_at TEXT NOT NULL,
		done INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL
	)`)
	if err != nil {
		panic(err)
	}
	columns, err := db.Query(`PRAGMA table_info(tasks)`)
	if err != nil {
		panic(err)
	}
	hasColor := false
	for columns.Next() {
		var cid, notnull, pk int
		var name, typ string
		var def any
		if err := columns.Scan(&cid, &name, &typ, &notnull, &def, &pk); err != nil {
			columns.Close()
			panic(err)
		}
		if name == "color" {
			hasColor = true
		}
	}
	columns.Close()
	if !hasColor {
		if _, err := db.Exec(`ALTER TABLE tasks ADD COLUMN color TEXT NOT NULL DEFAULT '#ffffff'`); err != nil {
			panic(err)
		}
	}

	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})
	r.GET("/api/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.GET("/api/tasks", func(c *gin.Context) {
		rows, err := db.Query(`SELECT id,title,description,start_at,done,created_at,color FROM tasks ORDER BY start_at`)
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось загрузить задачи"})
			return
		}
		defer rows.Close()
		tasks := []Task{}
		for rows.Next() {
			var t Task
			var start, created string
			var done int
			if err := rows.Scan(&t.ID, &t.Title, &t.Description, &start, &done, &created, &t.Color); err != nil {
				c.JSON(500, gin.H{"error": "Ошибка чтения задачи"})
				return
			}
			t.StartAt, _ = time.Parse(time.RFC3339, start)
			t.CreatedAt, _ = time.Parse(time.RFC3339, created)
			t.Done = done == 1
			tasks = append(tasks, t)
		}
		c.JSON(200, tasks)
	})
	r.POST("/api/tasks", func(c *gin.Context) {
		var input struct {
			Title       string    `json:"title"`
			Description string    `json:"description"`
			StartAt     time.Time `json:"startAt"`
			Color       string    `json:"color"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || input.Title == "" || input.StartAt.IsZero() {
			c.JSON(400, gin.H{"error": "Укажите название и дату задачи"})
			return
		}
		now := time.Now().UTC()
		if input.Color == "" {
			input.Color = "#ffffff"
		}
		res, err := db.Exec(`INSERT INTO tasks(title,description,start_at,created_at,color) VALUES(?,?,?,?,?)`, input.Title, input.Description, input.StartAt.Format(time.RFC3339), now.Format(time.RFC3339), input.Color)
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось сохранить задачу"})
			return
		}
		id, _ := res.LastInsertId()
		c.JSON(201, Task{ID: id, Title: input.Title, Description: input.Description, StartAt: input.StartAt, CreatedAt: now, Color: input.Color})
	})
	r.PATCH("/api/tasks/:id", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		var input struct {
			Done        *bool      `json:"done"`
			Title       *string    `json:"title"`
			Description *string    `json:"description"`
			StartAt     *time.Time `json:"startAt"`
			Color       *string    `json:"color"`
		}
		if err != nil || c.ShouldBindJSON(&input) != nil || (input.Done == nil && input.Title == nil && input.Description == nil && input.StartAt == nil && input.Color == nil) {
			c.JSON(400, gin.H{"error": "Некорректный запрос"})
			return
		}
		var current Task
		var start, created string
		var done int
		err = db.QueryRow(`SELECT id,title,description,start_at,done,created_at,color FROM tasks WHERE id=?`, id).Scan(&current.ID, &current.Title, &current.Description, &start, &done, &created, &current.Color)
		if err == sql.ErrNoRows {
			c.JSON(404, gin.H{"error": "Задача не найдена"})
			return
		}
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось загрузить задачу"})
			return
		}
		current.StartAt, _ = time.Parse(time.RFC3339, start)
		current.Done = done == 1
		if input.Done != nil {
			current.Done = *input.Done
		}
		if input.Title != nil {
			current.Title = *input.Title
		}
		if input.Description != nil {
			current.Description = *input.Description
		}
		if input.StartAt != nil {
			current.StartAt = *input.StartAt
		}
		if input.Color != nil {
			current.Color = *input.Color
		}
		res, err := db.Exec(`UPDATE tasks SET title=?,description=?,start_at=?,done=?,color=? WHERE id=?`, current.Title, current.Description, current.StartAt.Format(time.RFC3339), current.Done, current.Color, id)
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось обновить задачу"})
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			c.JSON(404, gin.H{"error": "Задача не найдена"})
			return
		}
		c.Status(204)
	})
	r.DELETE("/api/tasks/:id", func(c *gin.Context) {
		res, err := db.Exec(`DELETE FROM tasks WHERE id=?`, c.Param("id"))
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось удалить задачу"})
			return
		}
		n, _ := res.RowsAffected()
		if n == 0 {
			c.JSON(404, gin.H{"error": "Задача не найдена"})
			return
		}
		c.Status(204)
	})
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if err := r.Run(addr); err != nil {
		panic(err)
	}
}
