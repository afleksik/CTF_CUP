package handlers

import (
	"net/http"
	"strings"
	"time"
	"veladora/database"
	"veladora/models"

	"github.com/gin-gonic/gin"
)

func Talk(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDInt := userID.(int)

	var req models.TalkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ctx := c.Request.Context()
	response := generateBartenderResponse(req.Message)

	conversationOwnerID := userIDInt

	if req.Username != "" {
		var resolvedUserID int
		lookupErr := database.DB.QueryRow(ctx,
			"SELECT id FROM users WHERE username = $1", req.Username).Scan(&resolvedUserID)
		if lookupErr == nil {
			conversationOwnerID = resolvedUserID
		}
	}

	_, err := database.DB.Exec(ctx,
		"INSERT INTO conversations (user_id, content) VALUES ($1, $2)",
		conversationOwnerID, req.Message+"\nBartender: "+response)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save conversation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": response,
	})
}

func GetConversations(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userIDInt := userID.(int)

	ctx := c.Request.Context()

	rows, err := database.DB.Query(ctx,
		"SELECT id, content, created_at FROM conversations WHERE user_id = $1 ORDER BY created_at ASC",
		userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get conversations"})
		return
	}
	defer rows.Close()

	conversations := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int
		var content string
		var createdAt time.Time
		err := rows.Scan(&id, &content, &createdAt)
		if err != nil {
			continue
		}
		conversations = append(conversations, map[string]interface{}{
			"id":         id,
			"content":    content,
			"created_at": createdAt.Format(time.RFC3339),
		})
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get conversations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversations": conversations,
	})
}

func Remember(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.RememberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if len(req.ContextToken) != 32 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token must be exactly 32 bytes"})
		return
	}

	ctx := c.Request.Context()

	var userID int
	var err error

	if req.Username != "" {
		err = database.DB.QueryRow(ctx,
			"SELECT id FROM users WHERE username = $1", req.Username).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
	} else {
		userIDVal, _ := c.Get("user_id")
		userID = userIDVal.(int)
	}

	var found bool
	err = database.DB.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM conversations WHERE user_id = $1 AND POSITION($2 IN content) > 0)",
		userID, req.ContextToken).Scan(&found)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check conversations"})
		return
	}

	if !found {
		c.JSON(http.StatusOK, gin.H{
			"conversations": []map[string]interface{}{},
		})
		return
	}

	rows, err := database.DB.Query(ctx,
		"SELECT id, content, created_at FROM conversations WHERE user_id = $1 ORDER BY created_at ASC",
		userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get conversations"})
		return
	}
	defer rows.Close()

	conversations := make([]map[string]interface{}, 0)
	for rows.Next() {
		var id int
		var content string
		var createdAt time.Time
		err := rows.Scan(&id, &content, &createdAt)
		if err != nil {
			continue
		}
		conversations = append(conversations, map[string]interface{}{
			"id":         id,
			"content":    content,
			"created_at": createdAt.Format(time.RFC3339),
		})
	}

	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get conversations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"conversations": conversations,
	})
}

func generateBartenderResponse(message string) string {
	message = strings.ToLower(message)

	if strings.Contains(message, "flag") {
		return "I don't know anything about flags."
	}

	if strings.Contains(message, "hello") || strings.Contains(message, "hi") {
		return "Hello! Welcome to the bar. What would you like to drink?"
	}

	if strings.Contains(message, "drink") {
		return "We have beer, wine, cocktail, whiskey, and champagne. What would you like?"
	}

	return "Interesting... tell me more."
}
