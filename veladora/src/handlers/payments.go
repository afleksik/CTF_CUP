package handlers

import (
	"net/http"
	"veladora/database"
	"veladora/models"

	"github.com/gin-gonic/gin"
)

func PayBill(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDInt := userID.(int)

	var req models.PayBillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Comment = ""
	}

	ctx := c.Request.Context()

	var bill models.Bill
	err := database.DB.QueryRow(ctx,
		"SELECT id, user_id, amount, COALESCE(comment, ''), COALESCE(status, 'active') FROM bills WHERE user_id = $1 AND COALESCE(status, 'active') = 'active'",
		userIDInt).Scan(&bill.ID, &bill.UserID, &bill.Amount, &bill.Comment, &bill.Status)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No active bill found"})
		return
	}

	var balance int
	var username string
	err = database.DB.QueryRow(ctx,
		"SELECT balance, username FROM users WHERE id = $1", userIDInt).Scan(&balance, &username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
		return
	}

	if balance < bill.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Insufficient balance"})
		return
	}

	var paymentID string
	err = database.DB.QueryRow(ctx,
		"SELECT generate_payment_id($1, $2)", username, balance).Scan(&paymentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate payment ID"})
		return
	}

	tx, err := database.DB.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		"UPDATE users SET balance = balance - $1 WHERE id = $2", bill.Amount, userIDInt)
	if err != nil {
		tx.Rollback(ctx)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deduct balance"})
		return
	}

	_, err = tx.Exec(ctx,
		"UPDATE bills SET status = 'paid', payment_id = $1, comment = $2 WHERE id = $3",
		paymentID, req.Comment, bill.ID)
	if err != nil {
		tx.Rollback(ctx)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update bill"})
		return
	}

	_, err = tx.Exec(ctx,
		"UPDATE users SET payment_links = array_append(COALESCE(payment_links, ARRAY[]::TEXT[]), $1) WHERE id = $2",
		paymentID, userIDInt)
	if err != nil {
		tx.Rollback(ctx)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add payment link"})
		return
	}

	if err = tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	var newBalance int
	database.DB.QueryRow(ctx,
		"SELECT balance FROM users WHERE id = $1", userIDInt).Scan(&newBalance)

	c.JSON(http.StatusOK, gin.H{
		"message":      "Bill paid successfully",
		"bill_id":      bill.ID,
		"payment_id":   paymentID,
		"amount":       bill.Amount,
		"balance":      newBalance,
		"comment":      req.Comment,
		"status":       "paid",
		"payment_link": paymentID,
	})
}

func GetBillByID(c *gin.Context) {
	paymentID := c.Param("payment_id")
	if paymentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment ID required"})
		return
	}

	ctx := c.Request.Context()

	var bill models.Bill
	err := database.DB.QueryRow(ctx,
		"SELECT id, user_id, amount, COALESCE(comment, ''), COALESCE(status, 'active'), COALESCE(payment_id, ''), created_at FROM bills WHERE payment_id = $1",
		paymentID).Scan(&bill.ID, &bill.UserID, &bill.Amount, &bill.Comment, &bill.Status, &bill.PaymentID, &bill.CreatedAt)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bill not found"})
		return
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

	var username string
	database.DB.QueryRow(ctx,
		"SELECT username FROM users WHERE id = $1", bill.UserID).Scan(&username)

	c.JSON(http.StatusOK, gin.H{
		"bill":     bill,
		"username": username,
	})
}

func GetBills(c *gin.Context) {
	userID, _ := c.Get("user_id")
	userIDInt := userID.(int)

	ctx := c.Request.Context()

	rows, err := database.DB.Query(ctx,
		"SELECT id, user_id, amount, COALESCE(comment, ''), COALESCE(status, 'active'), COALESCE(payment_id, ''), created_at FROM bills WHERE user_id = $1 ORDER BY created_at DESC",
		userIDInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get bills"})
		return
	}
	defer rows.Close()

	var bills []models.Bill
	for rows.Next() {
		var bill models.Bill
		err := rows.Scan(&bill.ID, &bill.UserID, &bill.Amount, &bill.Comment, &bill.Status, &bill.PaymentID, &bill.CreatedAt)
		if err != nil {
			continue
		}
		if bill.Status == "" {
			bill.Status = "active"
		}
		bills = append(bills, bill)
	}

	c.JSON(http.StatusOK, gin.H{"bills": bills})
}
