package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"ti-server/internal/models"
	"ti-server/internal/storage"
)

type Server struct {
	storage *storage.Storage
	logger  *log.Logger
}

func NewServer(storage *storage.Storage, logger *log.Logger) *Server {
	return &Server{
		storage: storage,
		logger:  logger,
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

func (s *Server) getAPIKey(r *http.Request) string {
	return r.Header.Get("X-API-Key")
}

func (s *Server) HandleGetFeeds(w http.ResponseWriter, r *http.Request) {
	limit := 10
	offset := 0
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
			if limit > 100 {
				limit = 100
			}
		}
	}
	if offsetParam := r.URL.Query().Get("offset"); offsetParam != "" {
		if parsedOffset, err := strconv.Atoi(offsetParam); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	isPublicParam := r.URL.Query().Get("is_public")

	visibility := "all"
	switch isPublicParam {
	case "true":
		visibility = "public"
	case "false":
		visibility = "private"
	}

	feeds, total, err := s.storage.GetFeeds(r.Context(), visibility, limit, offset)

	if err != nil {
		s.logger.Printf("Error getting feeds: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if feeds == nil {
		feeds = []*models.Feed{}
	}

	if feeds == nil {
		feeds = []*models.Feed{}
	}

	response := models.FeedsResponse{
		Items:  feeds,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) HandleCreateFeed(w http.ResponseWriter, r *http.Request) {
	var req models.CreateFeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Name == "" {
		s.respondError(w, http.StatusBadRequest, "Name is required")
		return
	}

	feed, err := s.storage.CreateFeed(r.Context(), req.Name, req.Description, req.IsPublic)
	if err != nil {
		s.logger.Printf("Error creating feed: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusCreated, feed)
}

func (s *Server) HandleGetFeed(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/feeds/")
	feedID := strings.Split(path, "/")[0]

	if feedID == "" {
		s.respondError(w, http.StatusBadRequest, "Feed ID is required")
		return
	}

	if _, err := uuid.Parse(feedID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid feed ID format")
		return
	}

	apiKey := s.getAPIKey(r)

	feed, err := s.storage.GetFeed(r.Context(), feedID, apiKey)
	if err != nil {
		if errors.Is(err, storage.ErrFeedNotFound) {
			s.respondError(w, http.StatusNotFound, "Feed not found")
			return
		}
		if errors.Is(err, storage.ErrUnauthorized) {
			s.respondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		s.logger.Printf("Error getting feed: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusOK, feed)
}

func (s *Server) HandleAddIOC(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/feeds/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		s.respondError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	feedID := parts[0]

	if _, err := uuid.Parse(feedID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid feed ID format")
		return
	}

	var req models.AddIOCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Type == "" || req.Value == "" || req.Severity == "" {
		s.respondError(w, http.StatusBadRequest, "Type, value and severity are required")
		return
	}

	apiKey := s.getAPIKey(r)

	ioc, err := s.storage.AddIOC(r.Context(), feedID, req.Type, req.Value, req.Severity, req.Description, apiKey)
	if err != nil {
		if errors.Is(err, storage.ErrFeedNotFound) {
			s.respondError(w, http.StatusNotFound, "Feed not found")
			return
		}
		if errors.Is(err, storage.ErrUnauthorized) {
			s.respondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		if errors.Is(err, storage.ErrInvalidIOCType) {
			s.respondError(w, http.StatusBadRequest, "Invalid IOC type")
			return
		}
		if errors.Is(err, storage.ErrInvalidSeverity) {
			s.respondError(w, http.StatusBadRequest, "Invalid severity")
			return
		}
		s.logger.Printf("Error adding IOC: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	s.respondJSON(w, http.StatusCreated, ioc)
}

func (s *Server) HandleGetIOCs(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/feeds/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		s.respondError(w, http.StatusBadRequest, "Invalid path")
		return
	}
	feedID := parts[0]

	if _, err := uuid.Parse(feedID); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid feed ID format")
		return
	}

	limit := 100
	offset := 0

	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsedLimit, err := strconv.Atoi(limitParam); err == nil && parsedLimit > 0 {
			limit = parsedLimit
			if limit > 1000 {
				limit = 1000
			}
		}
	}

	if offsetParam := r.URL.Query().Get("offset"); offsetParam != "" {
		if parsedOffset, err := strconv.Atoi(offsetParam); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	apiKey := s.getAPIKey(r)

	iocs, err := s.storage.GetIOCs(r.Context(), feedID, apiKey, limit, offset)
	if err != nil {
		if errors.Is(err, storage.ErrFeedNotFound) {
			s.respondError(w, http.StatusNotFound, "Feed not found")
			return
		}
		if errors.Is(err, storage.ErrUnauthorized) {
			s.respondError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}
		s.logger.Printf("Error getting IOCs: %v", err)
		s.respondError(w, http.StatusInternalServerError, "Internal server error")
		return
	}

	if iocs == nil {
		iocs = []*models.IOC{}
	}

	s.respondJSON(w, http.StatusOK, iocs)
}
