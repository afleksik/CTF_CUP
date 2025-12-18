package models

import "time"

type Feed struct {
	ID          string    `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	IsPublic    bool      `json:"is_public" db:"is_public"`
	APIKey      string    `json:"api_key,omitempty" db:"api_key"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type IOC struct {
	ID          string    `json:"id" db:"id"`
	FeedID      string    `json:"feed_id" db:"feed_id"`
	Type        string    `json:"type" db:"type"`
	Value       string    `json:"value" db:"value"`
	Severity    string    `json:"severity" db:"severity"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type CreateFeedRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
}

type AddIOCRequest struct {
	Type        string `json:"type"`
	Value       string `json:"value"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
}

type FeedsResponse struct {
	Items  []*Feed `json:"items"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
}
