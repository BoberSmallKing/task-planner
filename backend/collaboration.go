package main

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type CalendarSummary struct {
	OwnerID    int64  `json:"ownerId"`
	OwnerLogin string `json:"ownerLogin"`
	Permission string `json:"permission"`
}

type CalendarParticipant struct {
	UserID     int64  `json:"userId"`
	Login      string `json:"login"`
	Permission string `json:"permission"`
	Status     string `json:"status"`
}

type CalendarInvitation struct {
	OwnerID    int64  `json:"ownerId"`
	OwnerLogin string `json:"ownerLogin"`
	Permission string `json:"permission"`
	CreatedAt  string `json:"createdAt"`
}

func registerCalendarRoutes(r *gin.Engine, db *sql.DB, secret []byte) {
	api := r.Group("/api")
	api.Use(authMiddleware(db, secret))

	api.GET("/calendars", func(c *gin.Context) {
		userID := c.GetInt64("userID")
		var login string
		if err := db.QueryRow(`SELECT login FROM users WHERE id=?`, userID).Scan(&login); err != nil {
			c.JSON(500, gin.H{"error": "Не удалось загрузить календари"})
			return
		}
		calendars := []CalendarSummary{{OwnerID: userID, OwnerLogin: login, Permission: "owner"}}
		rows, err := db.Query(`SELECT owner_id,u.login,permission FROM calendar_members cm JOIN users u ON u.id=cm.owner_id WHERE member_id=? AND status='accepted' ORDER BY u.login`, userID)
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось загрузить календари"})
			return
		}
		defer rows.Close()
		for rows.Next() {
			var calendar CalendarSummary
			if err := rows.Scan(&calendar.OwnerID, &calendar.OwnerLogin, &calendar.Permission); err != nil {
				c.JSON(500, gin.H{"error": "Ошибка чтения календаря"})
				return
			}
			calendars = append(calendars, calendar)
		}
		c.JSON(200, calendars)
	})

	api.GET("/calendar/participants", func(c *gin.Context) {
		userID := c.GetInt64("userID")
		ownerID, err := requestedCalendarID(c, userID)
		if err != nil {
			c.JSON(400, gin.H{"error": "Некорректный календарь"})
			return
		}
		if _, ok := calendarPermission(db, userID, ownerID); !ok {
			c.JSON(403, gin.H{"error": "Нет доступа к этому календарю"})
			return
		}
		var ownerLogin string
		if err := db.QueryRow(`SELECT login FROM users WHERE id=?`, ownerID).Scan(&ownerLogin); err != nil {
			c.JSON(404, gin.H{"error": "Владелец календаря не найден"})
			return
		}
		participants := []CalendarParticipant{{UserID: ownerID, Login: ownerLogin, Permission: "owner", Status: "accepted"}}
		query := `SELECT cm.member_id,u.login,cm.permission,cm.status FROM calendar_members cm JOIN users u ON u.id=cm.member_id WHERE cm.owner_id=? AND (cm.status='accepted' OR (?=? AND cm.status='pending')) ORDER BY cm.status,u.login`
		rows, err := db.Query(query, ownerID, userID, ownerID)
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось загрузить участников"})
			return
		}
		defer rows.Close()
		for rows.Next() {
			var participant CalendarParticipant
			if err := rows.Scan(&participant.UserID, &participant.Login, &participant.Permission, &participant.Status); err != nil {
				c.JSON(500, gin.H{"error": "Ошибка чтения участника"})
				return
			}
			participants = append(participants, participant)
		}
		c.JSON(200, participants)
	})

	api.POST("/calendar/invitations", func(c *gin.Context) {
		userID := c.GetInt64("userID")
		ownerID, err := requestedCalendarID(c, userID)
		if err != nil || ownerID != userID {
			c.JSON(403, gin.H{"error": "Приглашать участников может только владелец календаря"})
			return
		}
		var input struct {
			Email      string `json:"email"`
			Permission string `json:"permission"`
		}
		if c.ShouldBindJSON(&input) != nil {
			c.JSON(400, gin.H{"error": "Укажите email и режим доступа"})
			return
		}
		email, valid := normalizeEmail(input.Email)
		if !valid {
			c.JSON(400, gin.H{"error": "Введите корректный email"})
			return
		}
		if input.Permission != "read" && input.Permission != "edit" {
			c.JSON(400, gin.H{"error": "Выберите режим доступа"})
			return
		}
		var memberID int64
		if err := db.QueryRow(`SELECT id FROM users WHERE login=?`, email).Scan(&memberID); err != nil {
			if err == sql.ErrNoRows {
				c.JSON(404, gin.H{"error": "Аккаунт с таким email не найден"})
				return
			}
			c.JSON(500, gin.H{"error": "Не удалось найти пользователя"})
			return
		}
		if memberID == ownerID {
			c.JSON(400, gin.H{"error": "Нельзя пригласить владельца календаря"})
			return
		}
		var existingStatus string
		err = db.QueryRow(`SELECT status FROM calendar_members WHERE owner_id=? AND member_id=?`, ownerID, memberID).Scan(&existingStatus)
		if err == nil {
			if existingStatus == "accepted" {
				c.JSON(409, gin.H{"error": "Этот пользователь уже участвует в календаре"})
			} else {
				c.JSON(409, gin.H{"error": "Приглашение этому пользователю уже отправлено"})
			}
			return
		}
		if err != sql.ErrNoRows {
			c.JSON(500, gin.H{"error": "Не удалось проверить приглашение"})
			return
		}
		_, err = db.Exec(`INSERT INTO calendar_members(owner_id,member_id,permission,status,created_at) VALUES(?,?,?,'pending',?)`, ownerID, memberID, input.Permission, time.Now().UTC().Format(time.RFC3339))
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось создать приглашение"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"status": "pending"})
	})

	api.GET("/calendar/invitations", func(c *gin.Context) {
		rows, err := db.Query(`SELECT cm.owner_id,u.login,cm.permission,cm.created_at FROM calendar_members cm JOIN users u ON u.id=cm.owner_id WHERE cm.member_id=? AND cm.status='pending' ORDER BY cm.created_at`, c.GetInt64("userID"))
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось загрузить приглашения"})
			return
		}
		defer rows.Close()
		invitations := []CalendarInvitation{}
		for rows.Next() {
			var invitation CalendarInvitation
			if err := rows.Scan(&invitation.OwnerID, &invitation.OwnerLogin, &invitation.Permission, &invitation.CreatedAt); err != nil {
				c.JSON(500, gin.H{"error": "Ошибка чтения приглашения"})
				return
			}
			invitations = append(invitations, invitation)
		}
		c.JSON(200, invitations)
	})

	api.POST("/calendar/invitations/:ownerID/accept", func(c *gin.Context) {
		ownerID, err := strconv.ParseInt(c.Param("ownerID"), 10, 64)
		if err != nil {
			c.JSON(400, gin.H{"error": "Некорректное приглашение"})
			return
		}
		result, err := db.Exec(`UPDATE calendar_members SET status='accepted' WHERE owner_id=? AND member_id=? AND status='pending'`, ownerID, c.GetInt64("userID"))
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось принять приглашение"})
			return
		}
		count, _ := result.RowsAffected()
		if count == 0 {
			c.JSON(404, gin.H{"error": "Приглашение не найдено"})
			return
		}
		c.Status(http.StatusNoContent)
	})

	api.DELETE("/calendar/invitations/:ownerID", func(c *gin.Context) {
		ownerID, err := strconv.ParseInt(c.Param("ownerID"), 10, 64)
		if err != nil {
			c.JSON(400, gin.H{"error": "Некорректное приглашение"})
			return
		}
		result, err := db.Exec(`DELETE FROM calendar_members WHERE owner_id=? AND member_id=? AND status='pending'`, ownerID, c.GetInt64("userID"))
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось отклонить приглашение"})
			return
		}
		count, _ := result.RowsAffected()
		if count == 0 {
			c.JSON(404, gin.H{"error": "Приглашение не найдено"})
			return
		}
		c.Status(http.StatusNoContent)
	})

	api.DELETE("/calendar/participants/:userID", func(c *gin.Context) {
		ownerID := c.GetInt64("userID")
		memberID, err := strconv.ParseInt(c.Param("userID"), 10, 64)
		if err != nil {
			c.JSON(400, gin.H{"error": "Некорректный участник"})
			return
		}
		result, err := db.Exec(`DELETE FROM calendar_members WHERE owner_id=? AND member_id=?`, ownerID, memberID)
		if err != nil {
			c.JSON(500, gin.H{"error": "Не удалось удалить участника"})
			return
		}
		count, _ := result.RowsAffected()
		if count == 0 {
			c.JSON(404, gin.H{"error": "Участник не найден"})
			return
		}
		c.Status(http.StatusNoContent)
	})
}

func requestedCalendarID(c *gin.Context, defaultID int64) (int64, error) {
	value := c.GetHeader("X-Calendar-ID")
	if value == "" {
		return defaultID, nil
	}
	return strconv.ParseInt(value, 10, 64)
}

func calendarPermission(db *sql.DB, userID, ownerID int64) (string, bool) {
	if userID == ownerID {
		return "owner", true
	}
	var permission string
	err := db.QueryRow(`SELECT permission FROM calendar_members WHERE owner_id=? AND member_id=? AND status='accepted'`, ownerID, userID).Scan(&permission)
	return permission, err == nil
}
