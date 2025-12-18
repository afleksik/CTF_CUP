package handlers

import (
	"net/http"
	"veladora/database"
	"veladora/models"
	"veladora/utils"

	"github.com/gin-gonic/gin"
)

func Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var existingID int
	err := database.DB.QueryRow(c.Request.Context(),
		"SELECT id FROM users WHERE username = $1", req.Username).Scan(&existingID)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		return
	}

	passwordHash, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	var userID int
	err = database.DB.QueryRow(c.Request.Context(),
		"INSERT INTO users (username, password_hash, balance) VALUES ($1, $2, $3) RETURNING id",
		req.Username, passwordHash, 2500).Scan(&userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	token, err := utils.GenerateToken(userID, req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User registered successfully",
		"token":   token,
		"user": gin.H{
			"id":       userID,
			"username": req.Username,
			"balance":  2500,
		},
	})
}

func Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	var user models.User
	err := database.DB.QueryRow(c.Request.Context(),
		"SELECT id, username, password_hash, balance FROM users WHERE username = $1",
		req.Username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Balance)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := utils.GenerateToken(user.ID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"balance":  user.Balance,
		},
	})
}

func GetProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	var balance int
	var paymentLinks []string
	err := database.DB.QueryRow(c.Request.Context(),
		"SELECT balance, COALESCE(payment_links, ARRAY[]::TEXT[]) FROM users WHERE id = $1", userID).Scan(&balance, &paymentLinks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user balance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           userID,
		"username":     username,
		"balance":      balance,
		"payment_links": paymentLinks,
	})
}
