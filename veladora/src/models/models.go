package models

import (
	"time"
)

type User struct {
	ID           int       `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Balance      int       `json:"balance" db:"balance"`
	PaymentLinks []string  `json:"payment_links,omitempty" db:"payment_links"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type Order struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	BillID    *int      `json:"bill_id" db:"bill_id"`
	DrinkName string    `json:"drink_name" db:"drink_name"`
	Amount    int       `json:"amount" db:"amount"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type Bill struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Amount    int       `json:"amount" db:"amount"`
	Comment   string    `json:"comment" db:"comment"`
	Status    string    `json:"status" db:"status"`
	PaymentID string    `json:"payment_id,omitempty" db:"payment_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	Orders    []Order   `json:"orders,omitempty"`
}

type Conversation struct {
	ID        int       `json:"id" db:"id"`
	UserID    int       `json:"user_id" db:"user_id"`
	Content   string    `json:"content" db:"content"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type OrderRequest struct {
	DrinkName string `json:"drink_name" binding:"required"`
}

type PayBillRequest struct {
	Comment string `json:"comment"`
}

type TalkRequest struct {
	Message  string `json:"message" binding:"required"`
	Username string `json:"username"`
}

type RememberRequest struct {
	Username     string `json:"username"`
	ContextToken string `json:"context_token" binding:"required"`
}
