package gateway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GatewayClient struct {
	baseURL string
	client  *http.Client
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

type VirtualService struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Slug       string `json:"slug"`
	BackendURL string `json:"backend_url"`
	IsActive   bool   `json:"is_active"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func NewGatewayClient(baseURL string) *GatewayClient {
	return &GatewayClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *GatewayClient) CreateVirtualService(accessToken string, req *CreateVSRequest) (*VirtualService, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/virtual-services", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("gateway API error: %s - %s", resp.Status, string(bodyBytes))
		}
		return nil, fmt.Errorf("gateway API error: %s", errResp.Error)
	}

	var vs VirtualService
	if err := json.NewDecoder(resp.Body).Decode(&vs); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &vs, nil
}

func (c *GatewayClient) DeleteVirtualService(accessToken string, vsID string) error {
	httpReq, err := http.NewRequest(http.MethodDelete, c.baseURL+"/api/virtual-services/"+vsID, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("gateway API error: %s - %s", resp.Status, string(bodyBytes))
		}
		return fmt.Errorf("gateway API error: %s", errResp.Error)
	}

	return nil
}
