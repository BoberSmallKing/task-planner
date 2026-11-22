package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type Task struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"userId"`
	CalendarID  int64     `json:"calendarId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartAt     time.Time `json:"startAt"`
	Done        bool      `json:"done"`
	Color       string    `json:"color"`
	CreatedAt   time.Time `json:"createdAt"`
}

type User struct {
	ID           int64  `json:"id"`
	Login        string `json:"login"`
	PasswordHash string `json:"-"`
}

type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

type tokenClaims struct {
	UserID    int64  `json:"uid"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

func main() {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		panic("could not load backend/.env: " + err.Error())
	}
	secret := os.Getenv("AUTH_SECRET")
	if len(secret) < 32 {
		panic("AUTH_SECRET is missing or too short; set at least 32 characters in backend/.env")
	}
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
		user_id INTEGER NOT NULL DEFAULT 0,
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
	columns, err = db.Query(`PRAGMA table_info(tasks)`)
	if err != nil {
		panic(err)
	}
	hasUserID := false
	for columns.Next() {
		var cid, notnull, pk int
		var name, typ string
		var def any
		if err := columns.Scan(&cid, &name, &typ, &notnull, &def, &pk); err != nil {
			columns.Close()
			panic(err)
		}
		if name == "user_id" {
			hasUserID = true
		}
	}
	columns.Close()
	if !hasUserID {
		if _, err := db.Exec(`ALTER TABLE tasks ADD COLUMN user_id INTEGER NOT NULL DEFAULT 0`); err != nil {
			panic(err)
		}
	}
	for _, column := range []struct{ name, declaration string }{{"calendar_id", "INTEGER NOT NULL DEFAULT 0"}, {"created_by", "INTEGER NOT NULL DEFAULT 0"}} {
		columns, err = db.Query(`PRAGMA table_info(tasks)`)
		if err != nil {
			panic(err)
		}
		found := false
		for columns.Next() {
			var cid, notnull, pk int
			var name, typ string
			var def any
			if err := columns.Scan(&cid, &name, &typ, &notnull, &def, &pk); err != nil {
				columns.Close()
				panic(err)
			}
			if name == column.name {
				found = true
			}
		}
		columns.Close()
		if !found {
			if _, err := db.Exec(`ALTER TABLE tasks ADD COLUMN ` + column.name + ` ` + column.declaration); err != nil {
				panic(err)
			}
		}
	}
	if _, err := db.Exec(`UPDATE tasks SET calendar_id=user_id WHERE calendar_id=0`); err != nil {
		panic(err)
	}
	if _, err := db.Exec(`UPDATE tasks SET created_by=user_id WHERE created_by=0`); err != nil {
		panic(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		login TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at TEXT NOT NULL
	)`)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS sessions (
		token_hash TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		expires_at TEXT NOT NULL
	)`)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS calendar_members (
		owner_id INTEGER NOT NULL REFERENCES users(id),
		member_id INTEGER NOT NULL REFERENCES users(id),
		permission TEXT NOT NULL CHECK(permission IN ('read','edit')),
		status TEXT NOT NULL CHECK(status IN ('pending','accepted')),
		created_at TEXT NOT NULL,
		PRIMARY KEY(owner_id,member_id)
	)`)
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Calendar-ID")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})
	r.GET("/api/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	r.POST("/api/user/lookup", func(c *gin.Context) {
		var input struct {
			Login string `json:"login"`
		}
		if c.ShouldBindJSON(&input) != nil {
			c.JSON(400, gin.H{"error": "Введите адрес электронной почты"})
			return
		}
		login, valid := normalizeEmail(input.Login)
		if !valid {
			c.JSON(400, gin.H{"error": "Введите корректный адрес электронной почты"})
			return
		}
		var exists bool
		err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE login=?)`, login).Scan(&exists)
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось проверить логин"})
			return
		}
		c.JSON(200, gin.H{"exists": exists})
	})
	r.POST("/api/user/register", func(c *gin.Context) {
		var input struct {
			Login                string `json:"login"`
			Password             string `json:"password"`
			PasswordConfirmation string `json:"passwordConfirmation"`
		}
		if c.ShouldBindJSON(&input) != nil {
			c.JSON(400, gin.H{"error": "Некорректные данные регистрации"})
			return
		}
		login, valid := normalizeEmail(input.Login)
		if !valid {
			c.JSON(400, gin.H{"error": "Введите корректный адрес электронной почты"})
			return
		}
		input.Login = login
		if input.Password != input.PasswordConfirmation {
			c.JSON(400, gin.H{"error": "Пароли не совпадают"})
			return
		}
		if !validPassword(input.Password) {
			c.JSON(400, gin.H{"error": "Пароль должен содержать не менее 8 символов и хотя бы одну цифру"})
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось создать пользователя"})
			return
		}
		result, err := db.Exec(`INSERT INTO users(login,password_hash,created_at) VALUES(?,?,?)`, input.Login, string(hash), time.Now().UTC().Format(time.RFC3339))
		if err != nil {
			c.JSON(409, gin.H{"error": "Этот логин уже занят"})
			return
		}
		id, _ := result.LastInsertId()
		pair, err := createTokenPair(db, id, []byte(secret))
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось создать сессию"})
			return
		}
		c.JSON(201, gin.H{"accessToken": pair.AccessToken, "refreshToken": pair.RefreshToken, "user": User{ID: id, Login: input.Login}})
	})
	r.POST("/api/user/login", func(c *gin.Context) {
		var input struct {
			Login    string `json:"login"`
			Password string `json:"password"`
		}
		if c.ShouldBindJSON(&input) != nil {
			c.JSON(400, gin.H{"error": "Укажите логин и пароль"})
			return
		}
		var user User
		login, valid := normalizeEmail(input.Login)
		if !valid {
			c.JSON(400, gin.H{"error": "Введите корректный адрес электронной почты"})
			return
		}
		err := db.QueryRow(`SELECT id,login,password_hash FROM users WHERE login=?`, login).Scan(&user.ID, &user.Login, &user.PasswordHash)
		if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
			c.JSON(401, gin.H{"error": "Неверный логин или пароль"})
			return
		}
		pair, err := createTokenPair(db, user.ID, []byte(secret))
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось создать сессию"})
			return
		}
		c.JSON(200, gin.H{"accessToken": pair.AccessToken, "refreshToken": pair.RefreshToken, "user": user})
	})
	r.POST("/api/user/refresh", func(c *gin.Context) {
		var input struct {
			RefreshToken string `json:"refreshToken"`
		}
		if c.ShouldBindJSON(&input) != nil || input.RefreshToken == "" {
			c.JSON(401, gin.H{"error": "Требуется refresh token"})
			return
		}
		claims, err := verifyToken(input.RefreshToken, "refresh", []byte(secret))
		if err != nil {
			c.JSON(401, gin.H{"error": "Refresh token недействителен или истёк"})
			return
		}
		oldHash := tokenHash(input.RefreshToken)
		tx, err := db.Begin()
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось обновить сессию"})
			return
		}
		var sessionUser int64
		var expires string
		err = tx.QueryRow(`SELECT user_id,expires_at FROM sessions WHERE token_hash=?`, oldHash).Scan(&sessionUser, &expires)
		if err != nil || sessionUser != claims.UserID || expires <= time.Now().UTC().Format(time.RFC3339) {
			tx.Rollback()
			c.JSON(401, gin.H{"error": "Refresh token отозван или истёк"})
			return
		}
		if _, err = tx.Exec(`DELETE FROM sessions WHERE token_hash=?`, oldHash); err != nil {
			tx.Rollback()
			c.JSON(500, gin.H{"error": "Не удалось обновить сессию"})
			return
		}
		pair, err := createTokenPair(tx, claims.UserID, []byte(secret))
		if err != nil {
			tx.Rollback()
			c.JSON(500, gin.H{"error": "Не удалось обновить сессию"})
			return
		}
		if err = tx.Commit(); err != nil {
			c.JSON(500, gin.H{"error": "Не удалось обновить сессию"})
			return
		}
		c.JSON(200, pair)
	})
	r.POST("/api/user/logout", authMiddleware(db, []byte(secret)), func(c *gin.Context) {
		if _, err := db.Exec(`DELETE FROM sessions WHERE user_id=?`, c.GetInt64("userID")); err != nil {
			c.JSON(500, gin.H{"error": "Не удалось завершить сессию"})
			return
		}
		c.Status(http.StatusNoContent)
	})
	tasksAPI := r.Group("/api/tasks")
	tasksAPI.Use(authMiddleware(db, []byte(secret)))
	registerCalendarRoutes(r, db, []byte(secret))
	tasksAPI.GET("", func(c *gin.Context) {
		userID := c.GetInt64("userID")
		calendarID, err := requestedCalendarID(c, userID)
		if err != nil {
			c.JSON(400, gin.H{"error": "Некорректный календарь"})
			return
		}
		if _, ok := calendarPermission(db, userID, calendarID); !ok {
			c.JSON(403, gin.H{"error": "Нет доступа к этому календарю"})
			return
		}
		rows, err := db.Query(`SELECT id,created_by,calendar_id,title,description,start_at,done,created_at,color FROM tasks WHERE calendar_id=? ORDER BY start_at`, calendarID)
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
			if err := rows.Scan(&t.ID, &t.UserID, &t.CalendarID, &t.Title, &t.Description, &start, &done, &created, &t.Color); err != nil {
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
	tasksAPI.POST("", func(c *gin.Context) {
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
		userID := c.GetInt64("userID")
		calendarID, err := requestedCalendarID(c, userID)
		if err != nil {
			c.JSON(400, gin.H{"error": "Некорректный календарь"})
			return
		}
		permission, ok := calendarPermission(db, userID, calendarID)
		if !ok {
			c.JSON(403, gin.H{"error": "Нет доступа к этому календарю"})
			return
		}
		if permission == "read" {
			c.JSON(403, gin.H{"error": "У вас есть только доступ для чтения"})
			return
		}
		now := time.Now().UTC()
		if input.Color == "" {
			input.Color = "#ffffff"
		}
		res, err := db.Exec(`INSERT INTO tasks(user_id,calendar_id,created_by,title,description,start_at,created_at,color) VALUES(?,?,?,?,?,?,?,?)`, userID, calendarID, userID, input.Title, input.Description, input.StartAt.Format(time.RFC3339), now.Format(time.RFC3339), input.Color)
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось сохранить задачу"})
			return
		}
		id, _ := res.LastInsertId()
		c.JSON(201, Task{ID: id, UserID: userID, CalendarID: calendarID, Title: input.Title, Description: input.Description, StartAt: input.StartAt, CreatedAt: now, Color: input.Color})
	})
	tasksAPI.PATCH("/:id", func(c *gin.Context) {
		userID := c.GetInt64("userID")
		calendarID, err := requestedCalendarID(c, userID)
		if err != nil {
			c.JSON(400, gin.H{"error": "Некорректный календарь"})
			return
		}
		permission, ok := calendarPermission(db, userID, calendarID)
		if !ok {
			c.JSON(403, gin.H{"error": "Нет доступа к этому календарю"})
			return
		}
		if permission == "read" {
			c.JSON(403, gin.H{"error": "У вас есть только доступ для чтения"})
			return
		}
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
		err = db.QueryRow(`SELECT id,created_by,calendar_id,title,description,start_at,done,created_at,color FROM tasks WHERE id=? AND calendar_id=?`, id, calendarID).Scan(&current.ID, &current.UserID, &current.CalendarID, &current.Title, &current.Description, &start, &done, &created, &current.Color)
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
		res, err := db.Exec(`UPDATE tasks SET title=?,description=?,start_at=?,done=?,color=? WHERE id=? AND calendar_id=?`, current.Title, current.Description, current.StartAt.Format(time.RFC3339), current.Done, current.Color, id, calendarID)
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
	tasksAPI.DELETE("/:id", func(c *gin.Context) {
		userID := c.GetInt64("userID")
		calendarID, err := requestedCalendarID(c, userID)
		if err != nil {
			c.JSON(400, gin.H{"error": "Некорректный календарь"})
			return
		}
		permission, ok := calendarPermission(db, userID, calendarID)
		if !ok {
			c.JSON(403, gin.H{"error": "Нет доступа к этому календарю"})
			return
		}
		if permission == "read" {
			c.JSON(403, gin.H{"error": "У вас есть только доступ для чтения"})
			return
		}
		res, err := db.Exec(`DELETE FROM tasks WHERE id=? AND calendar_id=?`, c.Param("id"), calendarID)
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

func validPassword(password string) bool {
	if utf8.RuneCountInString(password) < 8 {
		return false
	}
	for _, char := range password {
		if unicode.IsDigit(char) {
			return true
		}
	}
	return false
}

func normalizeEmail(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value {
		return "", false
	}
	return value, true
}

type databaseWriter interface {
	Exec(query string, args ...any) (sql.Result, error)
}

func createTokenPair(db databaseWriter, userID int64, secret []byte) (TokenPair, error) {
	now := time.Now().UTC()
	accessToken, err := signToken(userID, "access", now.Add(15*time.Minute), secret)
	if err != nil {
		return TokenPair{}, err
	}
	refreshExpiry := now.Add(30 * 24 * time.Hour)
	refreshToken, err := signToken(userID, "refresh", refreshExpiry, secret)
	if err != nil {
		return TokenPair{}, err
	}
	_, err = db.Exec(`INSERT INTO sessions(token_hash,user_id,expires_at) VALUES(?,?,?)`, tokenHash(refreshToken), userID, refreshExpiry.Format(time.RFC3339))
	return TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, err
}

func signToken(userID int64, tokenType string, expires time.Time, secret []byte) (string, error) {
	claimID := make([]byte, 16)
	if _, err := rand.Read(claimID); err != nil {
		return "", err
	}
	claims := tokenClaims{
		UserID:    userID,
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			ID:        hex.EncodeToString(claimID),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func verifyToken(token, expectedType string, secret []byte) (tokenClaims, error) {
	var claims tokenClaims
	parsed, err := jwt.ParseWithClaims(token, &claims, func(parsed *jwt.Token) (any, error) {
		if parsed.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing algorithm")
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil {
		return tokenClaims{}, err
	}
	if parsed == nil || !parsed.Valid || claims.UserID <= 0 || claims.TokenType != expectedType {
		return tokenClaims{}, errors.New("invalid token claims")
	}
	return claims, nil
}

func tokenHash(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func authMiddleware(db *sql.DB, secret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Требуется вход в аккаунт"})
			return
		}
		claims, err := verifyToken(parts[1], "access", secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Access token недействителен или истёк"})
			return
		}
		c.Set("userID", claims.UserID)
		c.Next()
	}
}
