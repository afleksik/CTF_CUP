package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"gateway-server/internal/middleware"
	"gateway-server/internal/models"
	"gateway-server/internal/storage"
	"gateway-server/internal/ti"
)

type Server struct {
	storage  *storage.Storage
	tiClient *ti.TIClient
	logger   *log.Logger
}

func NewServer(storage *storage.Storage, tiClient *ti.TIClient, logger *log.Logger) *Server {
	return &Server{
		storage:  storage,
		tiClient: tiClient,
		logger:   logger,
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

func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request, authServerURL string) {
	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	reqBody, _ := json.Marshal(loginReq)
	resp, err := http.Post(
		authServerURL+"/auth/login",
		"application/json",
		strings.NewReader(string(reqBody)),
	)
	if err != nil {
		s.logger.Printf("Failed to proxy login to auth server: %v", err)
		s.respondError(w, http.StatusBadGateway, "Authentication service unavailable")
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
		json.NewEncoder(w).Encode(result)
	}
}

func (s *Server) HandleCreateVS(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	var req models.CreateVSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" || req.Slug == "" || req.BackendURL == "" {
		s.respondError(w, http.StatusBadRequest, "name, slug, and backend_url are required")
		return
	}

	if !isValidSlug(req.Slug) {
		s.respondError(w, http.StatusBadRequest, "slug must contain only lowercase letters, numbers, and hyphens")
		return
	}

	if !strings.HasPrefix(req.BackendURL, "http://") && !strings.HasPrefix(req.BackendURL, "https://") {
		s.respondError(w, http.StatusBadRequest, "backend_url must start with http:// or https://")
		return
	}

	if req.TIMode != "" && req.TIMode != "disabled" && req.TIMode != "monitor" && req.TIMode != "block" {
		s.respondError(w, http.StatusBadRequest, "ti_mode must be 'disabled', 'monitor', or 'block'")
		return
	}
	if req.TIMode == "" {
		req.TIMode = "disabled"
	}

	if req.RateLimitEnabled {
		if req.RateLimitRequests <= 0 || req.RateLimitWindowSec <= 0 {
			s.respondError(w, http.StatusBadRequest, "rate_limit_requests and rate_limit_window_sec must be positive")
			return
		}
	}

	if req.LogRetentionMinutes <= 0 || req.LogRetentionMinutes > 30 {
		req.LogRetentionMinutes = 30 // default
	}

	vs, err := s.storage.CreateVirtualService(r.Context(), &req, userID)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			s.respondError(w, http.StatusConflict, "slug already exists")
			return
		}
		s.logger.Printf("Error creating VS: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusCreated, vs)
}

func (s *Server) HandleGetVSList(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	services, err := s.storage.GetVirtualServicesByUser(r.Context(), userID)
	if err != nil {
		s.logger.Printf("Error getting VS list: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if services == nil {
		services = []*models.VirtualService{}
	}

	s.respondJSON(w, http.StatusOK, services)
}

func (s *Server) HandleGetVS(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	vsID := extractIDFromPath(r.URL.Path, "/api/virtual-services/")

	if _, err := uuid.Parse(vsID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid VS ID format")
		return
	}

	vs, err := s.storage.GetVirtualService(r.Context(), vsID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Virtual service not found")
		return
	}

	hasAccess, err := s.storage.UserHasAccessToVS(r.Context(), vsID, userID)
	if err != nil || !hasAccess {
		s.respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	tiFeeds, err := s.storage.GetVSTIFeeds(r.Context(), vsID)
	if err != nil {
		s.logger.Printf("Error fetching TI feeds: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	var feedInfos []models.TIFeedInfo
	for _, feed := range tiFeeds {
		var apiKey string
		if feed.APIKey != nil {
			apiKey = *feed.APIKey
		}

		feedData, err := s.tiClient.FetchFeedWithKey(feed.FeedID, apiKey)
		if err != nil {
			s.logger.Printf("Warning: Could not fetch feed %s: %v", feed.FeedID, err)
			feedInfos = append(feedInfos, models.TIFeedInfo{
				FeedID:   feed.FeedID,
				FeedName: feed.FeedID,
				IsActive: feed.IsActive,
				AddedAt:  feed.AddedAt,
			})
			continue
		}

		feedInfos = append(feedInfos, models.TIFeedInfo{
			FeedID:   feed.FeedID,
			FeedName: feedData.Name,
			IsActive: feed.IsActive,
			AddedAt:  feed.AddedAt,
		})
	}

	response := models.VSWithFeedsResponse{
		VirtualService: *vs,
		TIFeeds:        feedInfos,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) HandleUpdateVS(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	vsID := extractIDFromPath(r.URL.Path, "/api/virtual-services/")

	if _, err := uuid.Parse(vsID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid VS ID format")
		return
	}

	vs, err := s.storage.GetVirtualService(r.Context(), vsID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Virtual service not found")
		return
	}

	if vs.OwnerUserID != userID {
		s.respondError(w, http.StatusForbidden, "Only owner can update virtual service")
		return
	}

	var req models.UpdateVSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.TIMode != nil && *req.TIMode != "disabled" && *req.TIMode != "monitor" && *req.TIMode != "block" {
		s.respondError(w, http.StatusBadRequest, "ti_mode must be 'disabled', 'monitor', or 'block'")
		return
	}

	if req.BackendURL != nil {
		if !strings.HasPrefix(*req.BackendURL, "http://") && !strings.HasPrefix(*req.BackendURL, "https://") {
			s.respondError(w, http.StatusBadRequest, "backend_url must start with http:// or https://")
			return
		}
	}

	if req.LogRetentionMinutes != nil && (*req.LogRetentionMinutes <= 0 || *req.LogRetentionMinutes > 30) {
		s.respondError(w, http.StatusBadRequest, "log_retention_minutes must be between 1 and 30")
		return
	}

	if req.RateLimitEnabled != nil && *req.RateLimitEnabled {
		if req.RequireAuth != nil && !*req.RequireAuth {
			s.respondError(w, http.StatusBadRequest, "rate limiting requires authentication to be enabled")
			return
		}
	}

	err = s.storage.UpdateVirtualService(r.Context(), vsID, &req)
	if err != nil {
		s.logger.Printf("Error updating VS: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	updatedVS, _ := s.storage.GetVirtualService(r.Context(), vsID)
	s.respondJSON(w, http.StatusOK, updatedVS)
}

func (s *Server) HandleDeleteVS(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	vsID := extractIDFromPath(r.URL.Path, "/api/virtual-services/")

	if _, err := uuid.Parse(vsID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid VS ID format")
		return
	}

	vs, err := s.storage.GetVirtualService(r.Context(), vsID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Virtual service not found")
		return
	}

	if vs.OwnerUserID != userID {
		s.respondError(w, http.StatusForbidden, "Only owner can delete virtual service")
		return
	}

	err = s.storage.DeleteVirtualService(r.Context(), vsID)
	if err != nil {
		s.logger.Printf("Error deleting VS: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"message": "Virtual service deleted"})
}

func (s *Server) HandleAddUser(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	vsID := extractVSIDFromPath(r.URL.Path)

	if _, err := uuid.Parse(vsID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid VS ID format")
		return
	}

	vs, err := s.storage.GetVirtualService(r.Context(), vsID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Virtual service not found")
		return
	}

	if vs.OwnerUserID != userID {
		s.respondError(w, http.StatusForbidden, "Only owner can add users")
		return
	}

	var req models.AddUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.UserID == "" {
		s.respondError(w, http.StatusBadRequest, "user_id is required")
		return
	}

	err = s.storage.AddVSUser(r.Context(), vsID, req.UserID, userID)
	if err != nil {
		s.logger.Printf("Error adding user: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"message": "User added to virtual service"})
}

func (s *Server) HandleRemoveUser(w http.ResponseWriter, r *http.Request) {
	ownerID := middleware.GetUserID(r.Context())
	path := r.URL.Path

	parts := strings.Split(strings.TrimPrefix(path, "/api/virtual-services/"), "/")
	if len(parts) < 3 {
		s.respondError(w, http.StatusBadRequest, "Invalid path")
		return
	}

	vsID := parts[0]
	targetUserID := parts[2]

	if _, err := uuid.Parse(vsID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid VS ID format")
		return
	}

	vs, err := s.storage.GetVirtualService(r.Context(), vsID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Virtual service not found")
		return
	}

	if vs.OwnerUserID != ownerID {
		s.respondError(w, http.StatusForbidden, "Only owner can remove users")
		return
	}

	if targetUserID == vs.OwnerUserID {
		s.respondError(w, http.StatusBadRequest, "Cannot remove owner from virtual service")
		return
	}

	err = s.storage.RemoveVSUser(r.Context(), vsID, targetUserID)
	if err != nil {
		s.logger.Printf("Error removing user: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"message": "User removed from virtual service"})
}

func (s *Server) HandleGetUsers(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	vsID := extractVSIDFromPath(r.URL.Path)

	if _, err := uuid.Parse(vsID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid VS ID format")
		return
	}

	hasAccess, err := s.storage.UserHasAccessToVS(r.Context(), vsID, userID)
	if err != nil || !hasAccess {
		s.respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	users, err := s.storage.GetVSUsers(r.Context(), vsID)
	if err != nil {
		s.logger.Printf("Error getting users: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if users == nil {
		users = []*models.VirtualServiceUser{}
	}

	s.respondJSON(w, http.StatusOK, users)
}

func (s *Server) HandleAttachFeed(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	vsID := extractVSIDFromPath(r.URL.Path)

	if _, err := uuid.Parse(vsID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid VS ID format")
		return
	}

	vs, err := s.storage.GetVirtualService(r.Context(), vsID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Virtual service not found")
		return
	}

	if vs.OwnerUserID != userID {
		s.respondError(w, http.StatusForbidden, "Only owner can attach TI feeds")
		return
	}

	var req models.AttachTIFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.FeedID == "" {
		s.respondError(w, http.StatusBadRequest, "feed_id is required")
		return
	}

	if _, err := uuid.Parse(req.FeedID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid feed ID format")
		return
	}

	var apiKeyStr string
	if req.APIKey != nil {
		apiKeyStr = *req.APIKey
	}
	_, err = s.tiClient.FetchFeedWithKey(req.FeedID, apiKeyStr)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "TI feed not found or invalid API key")
		return
	}

	err = s.storage.AttachTIFeed(r.Context(), vsID, req.FeedID, req.APIKey)
	if err != nil {
		s.logger.Printf("Error attaching feed: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	feeds := []ti.FeedWithKey{{
		FeedID: req.FeedID,
		APIKey: apiKeyStr,
	}}
	if err := s.tiClient.UpdateCacheWithKeys(feeds); err != nil {
		s.logger.Printf("Error refreshing TI cache for feed %s: %v", req.FeedID, err)
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"message": "TI feed attached"})
}

func (s *Server) HandleDetachFeed(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	path := r.URL.Path

	parts := strings.Split(strings.TrimPrefix(path, "/api/virtual-services/"), "/")
	if len(parts) < 3 {
		s.respondError(w, http.StatusBadRequest, "Invalid path")
		return
	}

	vsID := parts[0]
	feedID := parts[2]

	if _, err := uuid.Parse(vsID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid VS ID format")
		return
	}

	vs, err := s.storage.GetVirtualService(r.Context(), vsID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Virtual service not found")
		return
	}

	if vs.OwnerUserID != userID {
		s.respondError(w, http.StatusForbidden, "Only owner can detach TI feeds")
		return
	}

	err = s.storage.DetachTIFeed(r.Context(), vsID, feedID)
	if err != nil {
		s.logger.Printf("Error detaching feed: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"message": "TI feed detached"})
}

func (s *Server) HandleToggleFeed(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	path := r.URL.Path

	parts := strings.Split(strings.TrimPrefix(path, "/api/virtual-services/"), "/")
	if len(parts) < 3 {
		s.respondError(w, http.StatusBadRequest, "Invalid path")
		return
	}

	vsID := parts[0]
	feedID := parts[2]

	if _, err := uuid.Parse(vsID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid VS ID format")
		return
	}

	vs, err := s.storage.GetVirtualService(r.Context(), vsID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Virtual service not found")
		return
	}

	if vs.OwnerUserID != userID {
		s.respondError(w, http.StatusForbidden, "Only owner can toggle TI feeds")
		return
	}

	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err = s.storage.ToggleTIFeed(r.Context(), vsID, feedID, req.IsActive)
	if err != nil {
		s.logger.Printf("Error toggling feed: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"message": "TI feed updated"})
}

func (s *Server) HandleGetFeeds(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	vsID := extractVSIDFromPath(r.URL.Path)

	if _, err := uuid.Parse(vsID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid VS ID format")
		return
	}

	hasAccess, err := s.storage.UserHasAccessToVS(r.Context(), vsID, userID)
	if err != nil || !hasAccess {
		s.respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	feeds, err := s.storage.GetVSTIFeeds(r.Context(), vsID)
	if err != nil {
		s.logger.Printf("Error getting feeds: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	var feedInfos []models.TIFeedInfo
	for _, feed := range feeds {
		info := models.TIFeedInfo{
			FeedID:   feed.FeedID,
			IsActive: feed.IsActive,
			AddedAt:  feed.AddedAt,
		}

		var apiKeyStr string
		if feed.APIKey != nil {
			apiKeyStr = *feed.APIKey
		}
		if tiFeed, err := s.tiClient.FetchFeedWithKey(feed.FeedID, apiKeyStr); err == nil {
			info.FeedName = tiFeed.Name
		}

		feedInfos = append(feedInfos, info)
	}

	if feedInfos == nil {
		feedInfos = []models.TIFeedInfo{}
	}

	s.respondJSON(w, http.StatusOK, feedInfos)
}

func (s *Server) HandleGetAvailableTIFeeds(w http.ResponseWriter, r *http.Request) {

	resp, err := http.Get(s.tiClient.GetServerURL() + "/feeds")
	if err != nil {
		s.respondError(w, http.StatusBadGateway, "Failed to fetch feeds from TI server")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.respondError(w, http.StatusBadGateway, "TI server returned error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	var feeds []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&feeds); err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to parse TI feeds")
		return
	}

	json.NewEncoder(w).Encode(feeds)
}

func (s *Server) HandleGetLogs(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	vsID := extractVSIDFromPath(r.URL.Path)

	if _, err := uuid.Parse(vsID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid VS ID format")
		return
	}

	hasAccess, err := s.storage.UserHasAccessToVS(r.Context(), vsID, userID)
	if err != nil || !hasAccess {
		s.respondError(w, http.StatusForbidden, "Access denied")
		return
	}

	limit := 20
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	var blocked *bool
	if blockedStr := r.URL.Query().Get("blocked"); blockedStr != "" {
		b := blockedStr == "true"
		blocked = &b
	}

	logs, total, err := s.storage.GetTrafficLogs(r.Context(), vsID, limit, offset, blocked)
	if err != nil {
		s.logger.Printf("Error getting logs: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if logs == nil {
		logs = []*models.TrafficLog{}
	}

	response := models.LogsResponse{
		Data:   make([]models.TrafficLog, 0),
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}

	for _, log := range logs {
		response.Data = append(response.Data, *log)
	}

	s.respondJSON(w, http.StatusOK, response)
}

func extractIDFromPath(path, prefix string) string {
	trimmed := strings.TrimPrefix(path, prefix)
	parts := strings.Split(trimmed, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func extractVSIDFromPath(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/api/virtual-services/"), "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

var slugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

func isValidSlug(slug string) bool {
	return slugRegex.MatchString(slug)
}
