package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"gateway-server/internal/middleware"
	"gateway-server/internal/models"
	"gateway-server/internal/ratelimit"
	"gateway-server/internal/storage"
	"gateway-server/internal/ti"
)

const maxBodySize = 10 * 1024 * 1024 // 10MB
const maxLogBodySize = 10 * 1024     // 10KB for logging
const proxyTimeout = 30 * time.Second

type ProxyHandler struct {
	storage     *storage.Storage
	tiClient    *ti.TIClient
	rateLimiter *ratelimit.RateLimiter
}

func NewProxyHandler(storage *storage.Storage, tiClient *ti.TIClient, rateLimiter *ratelimit.RateLimiter) *ProxyHandler {
	return &ProxyHandler{
		storage:     storage,
		tiClient:    tiClient,
		rateLimiter: rateLimiter,
	}
}

func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/vs/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		http.Error(w, "Invalid virtual service path", http.StatusBadRequest)
		return
	}

	slug := pathParts[0]
	remainingPath := "/" + strings.Join(pathParts[1:], "/")

	vs, err := p.storage.GetVirtualServiceBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, "Virtual service not found", http.StatusNotFound)
		return
	}

	if !vs.IsActive {
		http.Error(w, "Virtual service is not active", http.StatusServiceUnavailable)
		return
	}

	var userID *string
	if vs.RequireAuth {
		uid := middleware.GetUserID(r.Context())
		if uid == "" {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		userID = &uid

		hasAccess, err := p.storage.UserHasAccessToVS(r.Context(), vs.ID, uid)
		if err != nil || !hasAccess {
			http.Error(w, "Access denied", http.StatusForbidden)
			return
		}
	}

	if vs.RateLimitEnabled && userID != nil {
		if !p.rateLimiter.Allow(vs.ID, *userID, vs.RateLimitRequests, vs.RateLimitWindowSec) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
	}

	var iocMatches []models.IOCMatch
	blocked := false

	if vs.TIMode != "disabled" {
		feedIDs, err := p.storage.GetActiveVSTIFeeds(r.Context(), vs.ID)
		if err == nil && len(feedIDs) > 0 {
			iocs := p.tiClient.GetCache().GetIOCsForFeeds(feedIDs)

			iocMatches = ti.AnalyzeRequest(r, iocs)

			if len(iocMatches) > 0 && vs.TIMode == "block" {
				blocked = true
				p.logTraffic(r.Context(), vs, r, nil, iocMatches, blocked, userID, startTime, remainingPath)
				http.Error(w, "Request blocked by security policy", http.StatusForbidden)
				return
			}
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

	targetURL, err := url.Parse(vs.BackendURL)
	if err != nil {
		http.Error(w, "Invalid backend URL", http.StatusInternalServerError)
		return
	}

	targetURL.Path = strings.TrimSuffix(targetURL.Path, "/") + remainingPath
	targetURL.RawQuery = r.URL.RawQuery

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = &http.Transport{
		ResponseHeaderTimeout: proxyTimeout,
		IdleConnTimeout:       90 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Allow self-signed certificates
		},
	}

	var respRecorder *responseRecorder
	if len(iocMatches) > 0 {
		respRecorder = &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           &bytes.Buffer{},
		}
		w = respRecorder
	}

	proxy.Director = func(req *http.Request) {
		req.Host = targetURL.Host
		req.URL.Host = targetURL.Host
		req.URL.Scheme = targetURL.Scheme
		req.URL.Path = targetURL.Path
		req.URL.RawQuery = targetURL.RawQuery

		authHeader := r.Header.Get("Authorization")
		req.Header = http.Header{}
		if authHeader != "" {
			req.Header.Set("Authorization", authHeader)
		}
	}

	proxy.ServeHTTP(w, r)

	if len(iocMatches) > 0 {
		var resp *http.Response
		if respRecorder != nil {
			resp = &http.Response{
				StatusCode: respRecorder.statusCode,
				Header:     respRecorder.Header(),
				Body:       io.NopCloser(bytes.NewReader(respRecorder.body.Bytes())),
			}
		}
		p.logTraffic(r.Context(), vs, r, resp, iocMatches, blocked, userID, startTime, remainingPath)
	}
}

func (p *ProxyHandler) logTraffic(ctx context.Context, vs *models.VirtualService, req *http.Request,
	resp *http.Response, iocMatches []models.IOCMatch, blocked bool, userID *string, startTime time.Time, path string) {

	reqHeaders, _ := json.Marshal(req.Header)

	reqBody := ""
	if req.Body != nil {
		bodyBytes, err := io.ReadAll(io.LimitReader(req.Body, maxLogBodySize))
		if err == nil {
			reqBody = string(bodyBytes)
			if len(bodyBytes) >= maxLogBodySize {
				reqBody += "... [truncated]"
			}
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}
	}

	statusCode := 0
	var respHeaders json.RawMessage
	respBody := ""

	if resp != nil {
		statusCode = resp.StatusCode
		respHeaders, _ = json.Marshal(resp.Header)

		if resp.Body != nil {
			bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxLogBodySize))
			if err == nil {
				respBody = string(bodyBytes)
				if len(bodyBytes) >= maxLogBodySize {
					respBody += "... [truncated]"
				}
			}
		}
	}

	iocMatchesJSON, _ := storage.MarshalIOCMatches(iocMatches)

	clientIP := getClientIP(req)

	responseTimeMs := int(time.Since(startTime).Milliseconds())

	log := &models.TrafficLog{
		VSID:            vs.ID,
		UserID:          userID,
		ClientIP:        clientIP,
		Method:          req.Method,
		Path:            path,
		RequestHeaders:  reqHeaders,
		RequestBody:     reqBody,
		StatusCode:      statusCode,
		ResponseHeaders: respHeaders,
		ResponseBody:    respBody,
		IOCMatches:      iocMatchesJSON,
		Blocked:         blocked,
		ResponseTimeMs:  responseTimeMs,
	}

	go p.storage.LogTraffic(context.Background(), log)
}

func getClientIP(req *http.Request) string {
	xff := req.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	xri := req.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	parts := strings.Split(req.RemoteAddr, ":")
	if len(parts) > 0 {
		return parts[0]
	}

	return req.RemoteAddr
}

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}
