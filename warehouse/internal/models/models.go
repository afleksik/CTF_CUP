package models

import (
	"time"
)

type Realm struct {
	ID               string    `json:"id" db:"id"`
	Name             string    `json:"name" db:"name"`
	Description      string    `json:"description" db:"description"`
	OwnerUserID      string    `json:"owner_user_id" db:"owner_user_id"`
	GatewayVSID      *string   `json:"gateway_vs_id,omitempty" db:"gateway_vs_id"`
	GatewayVSSlug    *string   `json:"gateway_vs_slug,omitempty" db:"gateway_vs_slug"`
	GatewayProtected bool      `json:"gateway_protected" db:"gateway_protected"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type RealmWithRole struct {
	Realm
	Role string `json:"role"`
}

type RealmUser struct {
	RealmID  string    `json:"realm_id" db:"realm_id"`
	UserID   string    `json:"user_id" db:"user_id"`
	Role     string    `json:"role" db:"role"`
	AddedAt  time.Time `json:"added_at" db:"added_at"`
	Username string    `json:"username,omitempty"`
}

type Asset struct {
	ID          string    `json:"id" db:"id"`
	RealmID     string    `json:"realm_id" db:"realm_id"`
	Name        string    `json:"name" db:"name"`
	AssetType   string    `json:"asset_type" db:"asset_type"`
	Description string    `json:"description" db:"description"`
	OwnerUserID string    `json:"owner_user_id" db:"owner_user_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type CreateRealmRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type UpdateRealmRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type AddRealmUserRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type CreateAssetRequest struct {
	Name        string `json:"name"`
	AssetType   string `json:"asset_type"`
	Description string `json:"description"`
	OwnerUserID string `json:"owner_user_id"`
}

type UpdateAssetRequest struct {
	Name        string `json:"name"`
	AssetType   string `json:"asset_type"`
	Description string `json:"description"`
	OwnerUserID string `json:"owner_user_id"`
}

type CreateGatewayProtectionRequest struct {
	Slug                string `json:"slug"`
	RequireAuth         bool   `json:"require_auth"`
	TIMode              string `json:"ti_mode"`
	RateLimitEnabled    bool   `json:"rate_limit_enabled"`
	RateLimitRequests   int    `json:"rate_limit_requests"`
	RateLimitWindowSec  int    `json:"rate_limit_window_sec"`
	LogRetentionMinutes int    `json:"log_retention_minutes"`
}

type GatewayProtectionResponse struct {
	VSID        string `json:"vs_id"`
	VSSlug      string `json:"vs_slug"`
	PublicURL   string `json:"public_url"`
	BackendURL  string `json:"backend_url"`
	IsProtected bool   `json:"is_protected"`
}

type PaginatedResponse struct {
	Data   interface{} `json:"data"`
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

type UserSuggestion struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}
