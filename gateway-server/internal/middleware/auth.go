package middleware

import (
	"context"
	"net/http"
	"strings"

	"gateway-server/internal/auth"
)

type contextKey string

const (
	UserIDKey contextKey = "user_id"
)

type AuthMiddleware struct {
	verifier *auth.JWTVerifier
}

func NewAuthMiddleware(verifier *auth.JWTVerifier) *AuthMiddleware {
	return &AuthMiddleware{
		verifier: verifier,
	}
}

func (m *AuthMiddleware) RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing authorization header", http.StatusUnauthorized)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
			return
		}

		token := parts[1]
		claims, err := m.verifier.VerifyToken(token)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		next(w, r.WithContext(ctx))
	}
}

func (m *AuthMiddleware) OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				token := parts[1]
				claims, err := m.verifier.VerifyToken(token)
				if err == nil {
					ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
					r = r.WithContext(ctx)
				}
			}
		}
		next(w, r)
	}
}

func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}
