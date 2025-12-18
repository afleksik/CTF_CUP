package models

import (
	"encoding/json"
	"time"
)

type VirtualService struct {
	ID                  string    `json:"id" db:"id"`
	OwnerUserID         string    `json:"owner_user_id" db:"owner_user_id"`
	Name                string    `json:"name" db:"name"`
	Slug                string    `json:"slug" db:"slug"`
	BackendURL          string    `json:"backend_url" db:"backend_url"`
	IsActive            bool      `json:"is_active" db:"is_active"`
	RequireAuth         bool      `json:"require_auth" db:"require_auth"`
	TIMode              string    `json:"ti_mode" db:"ti_mode"` // "disabled", "monitor", "block"
	RateLimitEnabled    bool      `json:"rate_limit_enabled" db:"rate_limit_enabled"`
	RateLimitRequests   int       `json:"rate_limit_requests" db:"rate_limit_requests"`
	RateLimitWindowSec  int       `json:"rate_limit_window_sec" db:"rate_limit_window_sec"`
	LogRetentionMinutes int       `json:"log_retention_minutes" db:"log_retention_minutes"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

type VirtualServiceUser struct {
	VSID      string    `json:"vs_id" db:"vs_id"`
	UserID    string    `json:"user_id" db:"user_id"`
	GrantedBy string    `json:"granted_by" db:"granted_by"`
	GrantedAt time.Time `json:"granted_at" db:"granted_at"`
}

type VirtualServiceTIFeed struct {
	VSID     string    `json:"vs_id" db:"vs_id"`
	FeedID   string    `json:"feed_id" db:"feed_id"`
	APIKey   *string   `json:"api_key,omitempty" db:"api_key"` // For private feeds
	IsActive bool      `json:"is_active" db:"is_active"`
	AddedAt  time.Time `json:"added_at" db:"added_at"`
}

type TrafficLog struct {
	ID              string          `json:"id" db:"id"`
	VSID            string          `json:"vs_id" db:"vs_id"`
	UserID          *string         `json:"user_id" db:"user_id"`
	ClientIP        string          `json:"client_ip" db:"client_ip"`
	Method          string          `json:"method" db:"method"`
	Path            string          `json:"path" db:"path"`
	RequestHeaders  json.RawMessage `json:"request_headers" db:"request_headers"`
	RequestBody     string          `json:"request_body" db:"request_body"`
	StatusCode      int             `json:"status_code" db:"status_code"`
	ResponseHeaders json.RawMessage `json:"response_headers" db:"response_headers"`
	ResponseBody    string          `json:"response_body" db:"response_body"`
	IOCMatches      json.RawMessage `json:"ioc_matches" db:"ioc_matches"`
	Blocked         bool            `json:"blocked" db:"blocked"`
	ResponseTimeMs  int             `json:"response_time_ms" db:"response_time_ms"`
	Timestamp       time.Time       `json:"timestamp" db:"timestamp"`
}

type IOCMatch struct {
	IOCType  string `json:"ioc_type"`
	IOCValue string `json:"ioc_value"`
	Location string `json:"location"`
	FeedID   string `json:"feed_id"`
	FeedName string `json:"feed_name"`
}

type CreateVSRequest struct {
	Name                string `json:"name"`
	Slug                string `json:"slug"`
	BackendURL          string `json:"backend_url"`
	RequireAuth         bool   `json:"require_auth"`
	TIMode              string `json:"ti_mode"`
	RateLimitEnabled    bool   `json:"rate_limit_enabled"`
	RateLimitRequests   int    `json:"rate_limit_requests"`
	RateLimitWindowSec  int    `json:"rate_limit_window_sec"`
	LogRetentionMinutes int    `json:"log_retention_minutes"`
}

type UpdateVSRequest struct {
	Name                *string `json:"name,omitempty"`
	BackendURL          *string `json:"backend_url,omitempty"`
	IsActive            *bool   `json:"is_active,omitempty"`
	RequireAuth         *bool   `json:"require_auth,omitempty"`
	TIMode              *string `json:"ti_mode,omitempty"`
	RateLimitEnabled    *bool   `json:"rate_limit_enabled,omitempty"`
	RateLimitRequests   *int    `json:"rate_limit_requests,omitempty"`
	RateLimitWindowSec  *int    `json:"rate_limit_window_sec,omitempty"`
	LogRetentionMinutes *int    `json:"log_retention_minutes,omitempty"`
}

type AddUserRequest struct {
	UserID string `json:"user_id"`
}

type AttachTIFeedRequest struct {
	FeedID string  `json:"feed_id"`
	APIKey *string `json:"api_key,omitempty"`
}

type VSWithFeedsResponse struct {
	VirtualService
	TIFeeds []TIFeedInfo `json:"ti_feeds"`
}

type TIFeedInfo struct {
	FeedID   string    `json:"feed_id"`
	FeedName string    `json:"feed_name"`
	IsActive bool      `json:"is_active"`
	AddedAt  time.Time `json:"added_at"`
}

type LogsResponse struct {
	Data   []TrafficLog `json:"data"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}
