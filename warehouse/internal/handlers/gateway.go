package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"warehouse/internal/gateway"
	"warehouse/internal/middleware"
	"warehouse/internal/models"
)

func (s *Server) HandleCreateGatewayProtection(w http.ResponseWriter, r *http.Request, gatewayClient *gateway.GatewayClient, assetServerURL string) {
	path := strings.TrimPrefix(r.URL.Path, "/api/realms/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		s.respondError(w, http.StatusBadRequest, "Invalid URL")
		return
	}
	realmID := parts[0]

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		s.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.CreateGatewayProtectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Slug == "" {
		s.respondError(w, http.StatusBadRequest, "Slug is required")
		return
	}

	realm, err := s.storage.GetRealm(r.Context(), realmID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Realm not found")
		return
	}

	if realm.OwnerUserID != userID {
		s.respondError(w, http.StatusForbidden, "Only realm owner can create gateway protection")
		return
	}

	if realm.GatewayProtected {
		s.respondError(w, http.StatusBadRequest, "Realm is already protected by gateway")
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		s.respondError(w, http.StatusUnauthorized, "Authorization header required")
		return
	}
	accessToken := strings.TrimPrefix(authHeader, "Bearer ")

	vsReq := &gateway.CreateVSRequest{
		Name:                fmt.Sprintf("Realm: %s", realm.Name),
		Slug:                req.Slug,
		BackendURL:          fmt.Sprintf("%s/api/realms/%s", assetServerURL, realmID),
		RequireAuth:         req.RequireAuth,
		TIMode:              req.TIMode,
		RateLimitEnabled:    req.RateLimitEnabled,
		RateLimitRequests:   req.RateLimitRequests,
		RateLimitWindowSec:  req.RateLimitWindowSec,
		LogRetentionMinutes: req.LogRetentionMinutes,
	}

	vs, err := gatewayClient.CreateVirtualService(accessToken, vsReq)
	if err != nil {
		s.logger.Printf("Failed to create VS in gateway: %v", err)
		s.respondError(w, http.StatusBadGateway, "Failed to create gateway protection: "+err.Error())
		return
	}

	if err := s.storage.UpdateRealmGatewayInfo(r.Context(), realmID, vs.ID, vs.Slug); err != nil {
		_ = gatewayClient.DeleteVirtualService(accessToken, vs.ID)
		s.respondError(w, http.StatusInternalServerError, "Failed to update realm")
		return
	}

	response := models.GatewayProtectionResponse{
		VSID:        vs.ID,
		VSSlug:      vs.Slug,
		PublicURL:   fmt.Sprintf("/vs/%s", vs.Slug),
		BackendURL:  vs.BackendURL,
		IsProtected: true,
	}

	s.respondJSON(w, http.StatusCreated, response)
}

func (s *Server) HandleRemoveGatewayProtection(w http.ResponseWriter, r *http.Request, gatewayClient *gateway.GatewayClient) {
	path := strings.TrimPrefix(r.URL.Path, "/api/realms/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		s.respondError(w, http.StatusBadRequest, "Invalid URL")
		return
	}
	realmID := parts[0]

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		s.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	realm, err := s.storage.GetRealm(r.Context(), realmID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Realm not found")
		return
	}

	if realm.OwnerUserID != userID {
		s.respondError(w, http.StatusForbidden, "Only realm owner can remove gateway protection")
		return
	}

	if !realm.GatewayProtected || realm.GatewayVSID == nil {
		s.respondError(w, http.StatusBadRequest, "Realm is not protected by gateway")
		return
	}

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		s.respondError(w, http.StatusUnauthorized, "Authorization header required")
		return
	}
	accessToken := strings.TrimPrefix(authHeader, "Bearer ")

	if err := gatewayClient.DeleteVirtualService(accessToken, *realm.GatewayVSID); err != nil {
		s.logger.Printf("Failed to delete VS from gateway: %v", err)
	}

	if err := s.storage.RemoveRealmGatewayInfo(r.Context(), realmID); err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to update realm")
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]string{"message": "Gateway protection removed"})
}
