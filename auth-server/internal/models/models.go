package models

import "time"

type User struct {
	ID           string    `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Bio          string    `json:"bio" db:"bio"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
type PublicUser struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Bio       string    `json:"bio"`
	CreatedAt time.Time `json:"created_at"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Bio      string `json:"bio"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	TokenType string `json:"token_type"`
	ExpiresIn int    `json:"expires_in"`
	User      *User  `json:"user"`
}

type UpdateProfileRequest struct {
	Email string `json:"email"`
	Bio   string `json:"bio"`
}

type PublicKeyResponse struct {
	PublicKey string `json:"public_key"`
	Algorithm string `json:"algorithm"`
}

type UsersResponse struct {
	Users  []*PublicUser `json:"users"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
}

type AuthCode struct {
	Code                string     `json:"code" db:"code"`
	UserID              string     `json:"user_id" db:"user_id"`
	ClientID            string     `json:"client_id" db:"client_id"`
	RedirectURI         string     `json:"redirect_uri" db:"redirect_uri"`
	State               string     `json:"state" db:"state"`
	CodeChallenge       *string    `json:"code_challenge,omitempty" db:"code_challenge"`
	CodeChallengeMethod *string    `json:"code_challenge_method,omitempty" db:"code_challenge_method"`
	ExpiresAt           time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	Used                bool       `json:"used" db:"used"`
	UsedAt              *time.Time `json:"used_at,omitempty" db:"used_at"`
}

type RefreshToken struct {
	Token     string     `json:"token" db:"token"`
	UserID    string     `json:"user_id" db:"user_id"`
	ClientID  string     `json:"client_id" db:"client_id"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	Revoked   bool       `json:"revoked" db:"revoked"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
}

type OAuthClient struct {
	ClientID     string    `json:"client_id" db:"client_id"`
	ClientSecret string    `json:"-" db:"client_secret"`
	Name         string    `json:"name" db:"name"`
	RedirectURIs []string  `json:"redirect_uris" db:"redirect_uris"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}
