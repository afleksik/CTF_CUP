package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"warehouse/internal/middleware"
	"warehouse/internal/models"
	"warehouse/internal/storage"
)

var errUserNotFound = errors.New("user not found")

type Server struct {
	storage       *storage.Storage
	logger        *log.Logger
	authServerURL string
}

func NewServer(storage *storage.Storage, logger *log.Logger, authServerURL string) *Server {
	return &Server{
		storage:       storage,
		logger:        logger,
		authServerURL: authServerURL,
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

func (s *Server) getUsernameFromAuthServer(userID string) (string, error) {
	users, err := s.searchUsersInAuthServer(userID, 5)
	if err != nil {
		return "", err
	}

	for _, user := range users {
		if user.ID == userID {
			return user.Username, nil
		}
	}

	return "", errUserNotFound
}

func (s *Server) searchUsersInAuthServer(query string, limit int) ([]models.UserSuggestion, error) {
	searchURL, err := url.Parse(fmt.Sprintf("%s/users/search", s.authServerURL))
	if err != nil {
		return nil, err
	}

	q := searchURL.Query()
	q.Set("query", query)
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	searchURL.RawQuery = q.Encode()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Retry logic для eventual consistency при высокой нагрузке
	maxRetries := 3
	var result struct {
		Users []struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		} `json:"users"`
	}

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			// Небольшая задержка перед повторной попыткой
			time.Sleep(time.Duration(attempt*100) * time.Millisecond)
		}

		resp, err := client.Get(searchURL.String())
		if err != nil {
			if attempt == maxRetries-1 {
				return nil, err
			}
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			if attempt == maxRetries-1 {
				return nil, errors.New("failed to search users")
			}
			continue
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			if attempt == maxRetries-1 {
				return nil, err
			}
			continue
		}
		resp.Body.Close()

		// Если нашли результаты или это последняя попытка, возвращаем
		if len(result.Users) > 0 || attempt == maxRetries-1 {
			break
		}
	}

	suggestions := make([]models.UserSuggestion, 0, len(result.Users))
	for _, user := range result.Users {
		suggestions = append(suggestions, models.UserSuggestion{
			ID:       user.ID,
			Username: user.Username,
		})
	}

	return suggestions, nil
}

func (s *Server) resolveUserIdentifier(identifier string) (string, error) {
	if _, err := uuid.Parse(identifier); err == nil {
		return identifier, nil
	}

	users, err := s.searchUsersInAuthServer(identifier, 10)
	if err != nil {
		return "", err
	}

	for _, user := range users {
		if strings.EqualFold(user.Username, identifier) {
			return user.ID, nil
		}
	}

	return "", errUserNotFound
}

func (s *Server) HandleCreateRealm(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req models.CreateRealmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		s.respondError(w, http.StatusBadRequest, "Name is required")
		return
	}

	realm, err := s.storage.CreateRealm(r.Context(), req.Name, req.Description, userID)
	if err != nil {
		s.logger.Printf("Error creating realm: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusCreated, realm)
}

func (s *Server) HandleGetRealms(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	realms, err := s.storage.GetRealmsByUser(r.Context(), userID)
	if err != nil {
		s.logger.Printf("Error getting realms: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if realms == nil {
		realms = []*models.RealmWithRole{}
	}

	s.respondJSON(w, http.StatusOK, realms)
}

func (s *Server) HandleGetRealm(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/realms/")
	realmID := strings.Split(path, "/")[0]

	if realmID == "" {
		s.respondError(w, http.StatusBadRequest, "Realm ID is required")
		return
	}

	if _, err := uuid.Parse(realmID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid realm ID format")
		return
	}

	role, err := s.storage.GetUserRole(r.Context(), realmID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotInRealm) {
			s.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		s.logger.Printf("Error checking user role: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	realm, err := s.storage.GetRealm(r.Context(), realmID)
	if err != nil {
		if errors.Is(err, storage.ErrRealmNotFound) {
			s.respondError(w, http.StatusNotFound, "Realm not found")
			return
		}
		s.logger.Printf("Error getting realm: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	realmWithRole := models.RealmWithRole{
		Realm: *realm,
		Role:  role,
	}

	s.respondJSON(w, http.StatusOK, realmWithRole)
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

	users, err := s.searchUsersInAuthServer(query, limit)
	if err != nil {
		s.logger.Printf("Error searching users: %v", err)
		s.respondError(w, http.StatusBadGateway, "Failed to search users")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
	})
}

func (s *Server) HandleUpdateRealm(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/realms/")
	realmID := strings.Split(path, "/")[0]

	if _, err := uuid.Parse(realmID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid realm ID format")
		return
	}

	role, err := s.storage.GetUserRole(r.Context(), realmID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotInRealm) {
			s.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		s.logger.Printf("Error checking user role: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if !middleware.CanModifyRealm(role) {
		s.respondError(w, http.StatusForbidden, "Only bar managers can modify bars")
		return
	}

	var req models.UpdateRealmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		s.respondError(w, http.StatusBadRequest, "Name is required")
		return
	}

	realm, err := s.storage.UpdateRealm(r.Context(), realmID, req.Name, req.Description)
	if err != nil {
		if errors.Is(err, storage.ErrRealmNotFound) {
			s.respondError(w, http.StatusNotFound, "Realm not found")
			return
		}
		s.logger.Printf("Error updating realm: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, realm)
}

func (s *Server) HandleDeleteRealm(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/realms/")
	realmID := strings.Split(path, "/")[0]

	if _, err := uuid.Parse(realmID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid realm ID format")
		return
	}

	role, err := s.storage.GetUserRole(r.Context(), realmID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotInRealm) {
			s.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		s.logger.Printf("Error checking user role: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if !middleware.CanModifyRealm(role) {
		s.respondError(w, http.StatusForbidden, "Only bar managers can delete bars")
		return
	}

	if err := s.storage.DeleteRealm(r.Context(), realmID); err != nil {
		if errors.Is(err, storage.ErrRealmNotFound) {
			s.respondError(w, http.StatusNotFound, "Realm not found")
			return
		}
		s.logger.Printf("Error deleting realm: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"message": "Realm deleted successfully"})
}

func (s *Server) HandleAddRealmUser(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/realms/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		s.respondError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	realmID := parts[0]

	if _, err := uuid.Parse(realmID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid realm ID format")
		return
	}

	role, err := s.storage.GetUserRole(r.Context(), realmID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotInRealm) {
			s.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		s.logger.Printf("Error checking user role: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if !middleware.CanAddUsers(role) {
		s.respondError(w, http.StatusForbidden, "Only bar managers can add users")
		return
	}

	var req models.AddRealmUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.UserID == "" || req.Role == "" {
		s.respondError(w, http.StatusBadRequest, "user_id and role are required")
		return
	}

	targetUserID := req.UserID
	if _, err := uuid.Parse(targetUserID); err != nil {
		resolvedID, resolveErr := s.resolveUserIdentifier(targetUserID)
		if resolveErr != nil {
			if errors.Is(resolveErr, errUserNotFound) {
				s.respondError(w, http.StatusNotFound, "User not found")
			} else {
				s.logger.Printf("Error resolving user identifier: %v", resolveErr)
				s.respondError(w, http.StatusBadGateway, "Failed to fetch user info")
			}
			return
		}
		targetUserID = resolvedID
	}

	if err := s.storage.AddUserToRealm(r.Context(), realmID, targetUserID, req.Role); err != nil {
		if errors.Is(err, storage.ErrInvalidRole) {
			s.respondError(w, http.StatusBadRequest, "Invalid role")
			return
		}
		if errors.Is(err, storage.ErrUserAlreadyInRealm) {
			s.respondError(w, http.StatusConflict, "User already in realm")
			return
		}
		s.logger.Printf("Error adding user to realm: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusCreated, map[string]string{"message": "User added to realm"})
}

func (s *Server) HandleGetRealmUsers(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/realms/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		s.respondError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	realmID := parts[0]

	if _, err := uuid.Parse(realmID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid realm ID format")
		return
	}

	_, err := s.storage.GetUserRole(r.Context(), realmID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotInRealm) {
			s.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		s.logger.Printf("Error checking user role: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	limit := 20
	offset := 0
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	users, total, err := s.storage.GetRealmUsers(r.Context(), realmID, limit, offset)
	if err != nil {
		s.logger.Printf("Error getting realm users: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if users == nil {
		users = []*models.RealmUser{}
	}

	for _, user := range users {
		username, err := s.getUsernameFromAuthServer(user.UserID)
		if err == nil {
			user.Username = username
		}
	}

	response := models.PaginatedResponse{
		Data:   users,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) HandleRemoveRealmUser(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/realms/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		s.respondError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	realmID := parts[0]
	targetUserID := parts[2]

	if _, err := uuid.Parse(realmID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid realm ID format")
		return
	}

	role, err := s.storage.GetUserRole(r.Context(), realmID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotInRealm) {
			s.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		s.logger.Printf("Error checking user role: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if !middleware.CanAddUsers(role) {
		s.respondError(w, http.StatusForbidden, "Only bar managers can remove users")
		return
	}

	assets, err := s.storage.GetUserAssetsByRealm(r.Context(), realmID, targetUserID)
	if err != nil {
		s.logger.Printf("Error checking user assets: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if len(assets) > 0 {
		s.respondError(w, http.StatusConflict, "User owns assets in realm. Please reassign or delete their assets first.")
		return
	}

	if err := s.storage.RemoveUserFromRealm(r.Context(), realmID, targetUserID); err != nil {
		if errors.Is(err, storage.ErrUserNotInRealm) {
			s.respondError(w, http.StatusNotFound, "User not in realm")
			return
		}
		s.logger.Printf("Error removing user from realm: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"message": "User removed from realm"})
}

func (s *Server) HandleCreateAsset(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/realms/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		s.respondError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	realmID := parts[0]

	if _, err := uuid.Parse(realmID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid realm ID format")
		return
	}

	role, err := s.storage.GetUserRole(r.Context(), realmID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotInRealm) {
			s.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		s.logger.Printf("Error checking user role: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if !middleware.CanCreateAssets(role) {
		s.respondError(w, http.StatusForbidden, "Insufficient permissions to create assets")
		return
	}

	var req models.CreateAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.AssetType == "" {
		s.respondError(w, http.StatusBadRequest, "name and asset_type are required")
		return
	}

	ownerUserID := req.OwnerUserID
	if ownerUserID == "" {
		ownerUserID = userID
	}

	asset, err := s.storage.CreateAsset(r.Context(), realmID, req.Name, req.AssetType, req.Description, ownerUserID)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidAssetType) {
			s.respondError(w, http.StatusBadRequest, "Invalid asset type")
			return
		}
		s.logger.Printf("Error creating asset: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusCreated, asset)
}

func (s *Server) HandleGetAssets(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/realms/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		s.respondError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	realmID := parts[0]

	if _, err := uuid.Parse(realmID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid realm ID format")
		return
	}

	_, err := s.storage.GetUserRole(r.Context(), realmID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotInRealm) {
			s.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		s.logger.Printf("Error checking user role: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	assetType := r.URL.Query().Get("type")
	search := r.URL.Query().Get("search")
	limit := 20
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	assets, total, err := s.storage.GetAssetsByRealm(r.Context(), realmID, assetType, search, limit, offset)
	if err != nil {
		s.logger.Printf("Error getting assets: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if assets == nil {
		assets = []*models.Asset{}
	}

	response := models.PaginatedResponse{
		Data:   assets,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) HandleGetAsset(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/assets/")
	assetID := strings.Split(path, "/")[0]

	if _, err := uuid.Parse(assetID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid asset ID format")
		return
	}

	asset, err := s.storage.GetAsset(r.Context(), assetID)
	if err != nil {
		if errors.Is(err, storage.ErrAssetNotFound) {
			s.respondError(w, http.StatusNotFound, "Asset not found")
			return
		}
		s.logger.Printf("Error getting asset: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	_, err = s.storage.GetUserRole(r.Context(), asset.RealmID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotInRealm) {
			s.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		s.logger.Printf("Error checking user role: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, asset)
}

func (s *Server) HandleUpdateAsset(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/assets/")
	assetID := strings.Split(path, "/")[0]

	if _, err := uuid.Parse(assetID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid asset ID format")
		return
	}

	asset, err := s.storage.GetAsset(r.Context(), assetID)
	if err != nil {
		if errors.Is(err, storage.ErrAssetNotFound) {
			s.respondError(w, http.StatusNotFound, "Asset not found")
			return
		}
		s.logger.Printf("Error getting asset: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	role, err := s.storage.GetUserRole(r.Context(), asset.RealmID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotInRealm) {
			s.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		s.logger.Printf("Error checking user role: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if !middleware.CanModifyAsset(role, asset.OwnerUserID, userID) {
		s.respondError(w, http.StatusForbidden, "Insufficient permissions to modify this asset")
		return
	}

	var req models.UpdateAssetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.AssetType == "" {
		s.respondError(w, http.StatusBadRequest, "name and asset_type are required")
		return
	}

	updatedAsset, err := s.storage.UpdateAsset(r.Context(), assetID, req.Name, req.AssetType, req.Description, req.OwnerUserID)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidAssetType) {
			s.respondError(w, http.StatusBadRequest, "Invalid asset type")
			return
		}
		if errors.Is(err, storage.ErrAssetNotFound) {
			s.respondError(w, http.StatusNotFound, "Asset not found")
			return
		}
		s.logger.Printf("Error updating asset: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, updatedAsset)
}

func (s *Server) HandleDeleteAsset(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/assets/")
	assetID := strings.Split(path, "/")[0]

	if _, err := uuid.Parse(assetID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid asset ID format")
		return
	}

	asset, err := s.storage.GetAsset(r.Context(), assetID)
	if err != nil {
		if errors.Is(err, storage.ErrAssetNotFound) {
			s.respondError(w, http.StatusNotFound, "Asset not found")
			return
		}
		s.logger.Printf("Error getting asset: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	role, err := s.storage.GetUserRole(r.Context(), asset.RealmID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotInRealm) {
			s.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		s.logger.Printf("Error checking user role: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if !middleware.CanDeleteAsset(role, asset.OwnerUserID, userID) {
		s.respondError(w, http.StatusForbidden, "Insufficient permissions to delete this asset")
		return
	}

	if err := s.storage.DeleteAsset(r.Context(), assetID); err != nil {
		if errors.Is(err, storage.ErrAssetNotFound) {
			s.respondError(w, http.StatusNotFound, "Asset not found")
			return
		}
		s.logger.Printf("Error deleting asset: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"message": "Asset deleted successfully"})
}

func (s *Server) HandleGetUserAssets(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/realms/")
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		s.respondError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	realmID := parts[0]
	targetUserID := parts[2]

	if _, err := uuid.Parse(realmID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid realm ID format")
		return
	}

	role, err := s.storage.GetUserRole(r.Context(), realmID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotInRealm) {
			s.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		s.logger.Printf("Error checking user role: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if !middleware.CanAddUsers(role) {
		s.respondError(w, http.StatusForbidden, "Only bar managers can view user assets")
		return
	}

	assets, err := s.storage.GetUserAssetsByRealm(r.Context(), realmID, targetUserID)
	if err != nil {
		s.logger.Printf("Error getting user assets: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if assets == nil {
		assets = []*models.Asset{}
	}

	s.respondJSON(w, http.StatusOK, assets)
}

func (s *Server) HandleReassignUserAssets(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/realms/")
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		s.respondError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	realmID := parts[0]
	targetUserID := parts[2]

	if _, err := uuid.Parse(realmID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid realm ID format")
		return
	}

	role, err := s.storage.GetUserRole(r.Context(), realmID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotInRealm) {
			s.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		s.logger.Printf("Error checking user role: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if !middleware.CanAddUsers(role) {
		s.respondError(w, http.StatusForbidden, "Only bar managers can reassign assets")
		return
	}

	var req struct {
		NewOwnerID string `json:"new_owner_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.NewOwnerID == "" {
		s.respondError(w, http.StatusBadRequest, "new_owner_id is required")
		return
	}

	_, err = s.storage.GetUserRole(r.Context(), realmID, req.NewOwnerID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotInRealm) {
			s.respondError(w, http.StatusBadRequest, "New owner is not in realm")
			return
		}
		s.logger.Printf("Error checking new owner: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if err := s.storage.ReassignUserAssets(r.Context(), realmID, targetUserID, req.NewOwnerID); err != nil {
		s.logger.Printf("Error reassigning assets: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"message": "Assets reassigned successfully"})
}

func (s *Server) HandleDeleteUserAssets(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	path := strings.TrimPrefix(r.URL.Path, "/api/realms/")
	parts := strings.Split(path, "/")
	if len(parts) < 4 {
		s.respondError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	realmID := parts[0]
	targetUserID := parts[2]

	if _, err := uuid.Parse(realmID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid realm ID format")
		return
	}

	role, err := s.storage.GetUserRole(r.Context(), realmID, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotInRealm) {
			s.respondError(w, http.StatusForbidden, "Access denied")
			return
		}
		s.logger.Printf("Error checking user role: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if !middleware.CanAddUsers(role) {
		s.respondError(w, http.StatusForbidden, "Only bar managers can delete user assets")
		return
	}

	if err := s.storage.DeleteUserAssets(r.Context(), realmID, targetUserID); err != nil {
		s.logger.Printf("Error deleting user assets: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"message": "Assets deleted successfully"})
}
