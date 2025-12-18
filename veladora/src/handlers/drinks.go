package handlers

import (
	"net/http"
	"veladora/database"
	"veladora/models"
	"veladora/utils"

	"github.com/gin-gonic/gin"
)

func OrderDrink(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDInt := userID.(int)

	var req models.OrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	ctx := c.Request.Context()

	if !utils.IsValidDrink(req.DrinkName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid drink name"})
		return
	}

	price := utils.GetDrinkPrice(req.DrinkName)

	var billID int
	err := database.DB.QueryRow(ctx,
		"SELECT id FROM bills WHERE user_id = $1 AND status = 'active'",
		userIDInt).Scan(&billID)

	if err != nil {
		err = database.DB.QueryRow(ctx,
			"INSERT INTO bills (user_id, amount, status) VALUES ($1, $2, $3) RETURNING id",
			userIDInt, 0, "active").Scan(&billID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create bill"})
			return
		}
	}

	var orderID int
	err = database.DB.QueryRow(ctx,
		"INSERT INTO orders (user_id, bill_id, drink_name, amount, status) VALUES ($1, $2, $3, $4, $5) RETURNING id",
		userIDInt, billID, req.DrinkName, price, "active").Scan(&orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	_, err = database.DB.Exec(ctx,
		"UPDATE bills SET amount = amount + $1 WHERE id = $2", price, billID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bill"})
		return
	}

	var billAmount int
	database.DB.QueryRow(ctx,
		"SELECT amount FROM bills WHERE id = $1", billID).Scan(&billAmount)

	c.JSON(http.StatusOK, gin.H{
		"message":    "Drink added to bill",
		"order_id":   orderID,
		"bill_id":    billID,
		"drink_name": req.DrinkName,
		"amount":     price,
		"bill_total": billAmount,
	})
}

func GetActiveBill(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDInt := userID.(int)

	ctx := c.Request.Context()

	var bill models.Bill
	err := database.DB.QueryRow(ctx,
		`SELECT id, user_id, amount, COALESCE(comment, ''), COALESCE(status, 'active'), COALESCE(payment_id, ''), created_at 
		 FROM bills 
		 WHERE user_id = $1 AND (status != 'paid' OR status IS NULL) 
		 ORDER BY CASE WHEN COALESCE(status, 'active') = 'active' THEN 0 ELSE 1 END, created_at DESC 
		 LIMIT 1`,
		userIDInt).Scan(&bill.ID, &bill.UserID, &bill.Amount, &bill.Comment, &bill.Status, &bill.PaymentID, &bill.CreatedAt)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"bill": nil,
		})
		return
	}

	if bill.Status != "active" {
		_, err = database.DB.Exec(ctx,
			"UPDATE bills SET status = 'active' WHERE id = $1", bill.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bill status"})
			return
		}
		bill.Status = "active"
	}

	rows, err := database.DB.Query(ctx,
		"SELECT id, user_id, bill_id, drink_name, amount, status, created_at FROM orders WHERE bill_id = $1 ORDER BY created_at ASC",
		bill.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get orders"})
		return
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		var billID *int
		err := rows.Scan(&order.ID, &order.UserID, &billID, &order.DrinkName, &order.Amount, &order.Status, &order.CreatedAt)
		if err != nil {
			continue
		}
		order.BillID = billID
		orders = append(orders, order)
	}
	bill.Orders = orders

	c.JSON(http.StatusOK, gin.H{
		"bill": bill,
	})
}

func GetOrders(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDInt := userID.(int)

	ctx := c.Request.Context()

	rows, err := database.DB.Query(ctx,
		"SELECT id, user_id, bill_id, drink_name, amount, status, created_at FROM orders WHERE user_id = $1 ORDER BY created_at DESC",
		userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get orders"})
		return
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var order models.Order
		var billID *int
		err := rows.Scan(&order.ID, &order.UserID, &billID, &order.DrinkName, &order.Amount, &order.Status, &order.CreatedAt)
		if err != nil {
			continue
		}
		order.BillID = billID
		orders = append(orders, order)
	}

	c.JSON(http.StatusOK, gin.H{"orders": orders})
}
