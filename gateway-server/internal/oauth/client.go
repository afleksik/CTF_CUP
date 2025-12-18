package oauth

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type OAuthClient struct {
	authServerURL       string
	publicAuthServerURL string
	clientID            string
	clientSecret        string
	redirectURI         string
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
}

func NewOAuthClient(authServerURL, publicAuthServerURL, clientID, clientSecret, redirectURI string) *OAuthClient {
	if publicAuthServerURL == "" {
		publicAuthServerURL = authServerURL
	}
	return &OAuthClient{
		authServerURL:       authServerURL,
		publicAuthServerURL: publicAuthServerURL,
		clientID:            clientID,
		clientSecret:        clientSecret,
		redirectURI:         redirectURI,
	}
}

func GenerateCodeVerifier() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func GenerateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

func (c *OAuthClient) GetAuthorizationURL(state, codeChallenge, redirectOverride string) string {
	params := url.Values{}
	params.Set("client_id", c.clientID)
	params.Set("redirect_uri", c.selectRedirectURI(redirectOverride))
	params.Set("response_type", "code")
	params.Set("state", state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")

	baseURL := c.publicAuthServerURL
	if baseURL == "" {
		baseURL = c.authServerURL
	}

	// If redirectOverride is provided, derive auth server URL from redirect URI
	// This allows dynamic auth server URL based on the request
	if redirectOverride != "" {
		if parsed, err := url.Parse(redirectOverride); err == nil {
			// Extract scheme and host from redirect URI
			// e.g., https://example.com/gateway/auth/callback -> https://example.com
			// We don't add /auth here because it's added below
			baseURL = parsed.Scheme + "://" + parsed.Host
		}
	}

	return baseURL + "/auth/auth/authorize?" + params.Encode()
}

func (c *OAuthClient) ExchangeCodeForToken(code, codeVerifier, redirectOverride string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", c.selectRedirectURI(redirectOverride))
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("code_verifier", codeVerifier)

	req, err := http.NewRequest(http.MethodPost, c.authServerURL+"/auth/token", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token exchange failed: %s - %s", resp.Status, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tokenResp, nil
}

func (c *OAuthClient) selectRedirectURI(override string) string {
	if override != "" {
		return override
	}
	return c.redirectURI
}

func (c *OAuthClient) DefaultRedirectURI() string {
	return c.redirectURI
}

func (c *OAuthClient) RefreshAccessToken(refreshToken string) (*TokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)

	req, err := http.NewRequest(http.MethodPost, c.authServerURL+"/auth/refresh", bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed: %s - %s", resp.Status, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tokenResp, nil
}
