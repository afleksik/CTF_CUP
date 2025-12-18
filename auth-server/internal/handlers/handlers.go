package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"auth-server/internal/auth"
	"auth-server/internal/middleware"
	"auth-server/internal/models"
	"auth-server/internal/storage"
)

type Server struct {
	storage    *storage.Storage
	jwtManager *auth.JWTManager
	logger     *log.Logger
	publicKey  string
}

func NewServer(storage *storage.Storage, jwtManager *auth.JWTManager, publicKey string, logger *log.Logger) *Server {
	return &Server{
		storage:    storage,
		jwtManager: jwtManager,
		logger:     logger,
		publicKey:  publicKey,
	}
}

func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{"error": message})
}

func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		s.respondError(w, http.StatusBadRequest, "Username, email and password are required")
		return
	}

	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.logger.Printf("Error hashing password: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	user, err := s.storage.CreateUser(r.Context(), req.Username, req.Email, passwordHash, req.Bio)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			s.respondError(w, http.StatusConflict, "User already exists")
			return
		}
		if errors.Is(err, storage.ErrInvalidEmail) {
			s.respondError(w, http.StatusBadRequest, "Invalid email format")
			return
		}
		if errors.Is(err, storage.ErrInvalidUsername) {
			s.respondError(w, http.StatusBadRequest, "Invalid username")
			return
		}
		s.logger.Printf("Error creating user: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusCreated, user)
}

func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		s.respondError(w, http.StatusBadRequest, "Username and password are required")
		return
	}

	user, err := s.storage.GetUserByUsername(r.Context(), req.Username)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			s.respondError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}
		s.logger.Printf("Error getting user: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		s.respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	token, err := s.jwtManager.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		s.logger.Printf("Error generating token: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	response := models.LoginResponse{
		Token:     token,
		TokenType: "Bearer",
		ExpiresIn: 900,
		User:      user,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)

	user, err := s.storage.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			s.respondError(w, http.StatusNotFound, "User not found")
			return
		}
		s.logger.Printf("Error getting user: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, user)
}

func (s *Server) HandleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)

	var req models.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	user, err := s.storage.UpdateUser(r.Context(), userID, req.Email, req.Bio)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			s.respondError(w, http.StatusNotFound, "User not found")
			return
		}
		if errors.Is(err, storage.ErrInvalidEmail) {
			s.respondError(w, http.StatusBadRequest, "Invalid email format")
			return
		}
		s.logger.Printf("Error updating user: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, user)
}

func (s *Server) HandleLogout(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(middleware.UserIDKey).(string)
	if userID != "" {
		if err := s.storage.RevokeRefreshTokensByUser(r.Context(), userID); err != nil {
			s.logger.Printf("Error revoking refresh tokens for %s: %v", userID, err)
		}
	}
	s.respondJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

func (s *Server) HandleGetPublicKey(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "public, max-age=300")
	response := models.PublicKeyResponse{
		PublicKey: s.publicKey,
		Algorithm: "RS256",
	}
	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) HandleGetUsers(w http.ResponseWriter, r *http.Request) {
	limit := 10
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	users, total, err := s.storage.GetPublicUsers(r.Context(), limit, offset)
	if err != nil {
		s.logger.Printf("Error getting users: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if users == nil {
		users = []*models.PublicUser{}
	}

	response := models.UsersResponse{
		Users:  users,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) HandleSearchUsers(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if len(query) < 2 {
		s.respondError(w, http.StatusBadRequest, "query must be at least 2 characters")
		return
	}

	limit := 10
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	users, err := s.storage.SearchUsers(r.Context(), query, limit)
	if err != nil {
		s.logger.Printf("Error searching users: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if users == nil {
		users = []*models.User{}
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
	})
}

func (s *Server) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/users/")
	userID := strings.Split(path, "/")[0]

	if userID == "" {
		s.respondError(w, http.StatusBadRequest, "User ID is required")
		return
	}

	if _, err := uuid.Parse(userID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid user ID format")
		return
	}

	user, err := s.storage.GetPublicUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			s.respondError(w, http.StatusNotFound, "User not found")
			return
		}
		s.logger.Printf("Error getting user: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, user)
}
