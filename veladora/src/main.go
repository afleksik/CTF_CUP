package main

import (
	"log"
	"veladora/config"
	"veladora/database"
	"veladora/handlers"
	"veladora/middleware"
	"veladora/redis"
	"veladora/utils"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	utils.SetJWTSecret(cfg.Server.JWTSecret)

	if err := database.InitDatabase(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	if err := redis.InitRedis(cfg); err != nil {
		log.Fatalf("Failed to initialize Redis: %v", err)
	}
	defer redis.Close()

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		api.POST("/register", handlers.Register)
		api.POST("/login", handlers.Login)

	}

	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/profile", handlers.GetProfile)

		protected.POST("/order", handlers.OrderDrink)
		protected.GET("/orders", handlers.GetOrders)

		protected.GET("/bill/active", handlers.GetActiveBill)
		protected.POST("/bill/pay", handlers.PayBill)
		protected.GET("/bills", handlers.GetBills)
		protected.GET("/bill/:payment_id", handlers.GetBillByID)

		protected.POST("/talk", handlers.Talk)
		protected.GET("/conversations", handlers.GetConversations)
		protected.POST("/remember", handlers.Remember)

	}

	port := ":" + cfg.Server.Port
	log.Printf("Server starting on port %s", port)
	if err := r.Run(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
