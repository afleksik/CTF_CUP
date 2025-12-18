package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"gateway-server/internal/oauth"
)

type OAuthHandler struct {
	client        *oauth.OAuthClient
	stateStore    *stateStore
	cookieSecret  []byte
	cookieSecure  bool
	publicBaseURL string
}

func NewOAuthHandler(client *oauth.OAuthClient, cookieSecret []byte, cookieSecure bool, publicBaseURL string) *OAuthHandler {
	return &OAuthHandler{
		client:        client,
		stateStore:    newStateStore(),
		cookieSecret:  cookieSecret,
		cookieSecure:  cookieSecure,
		publicBaseURL: strings.TrimSuffix(publicBaseURL, "/"),
	}
}

func (h *OAuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := generateRandomString(32)
	if err != nil {
		http.Error(w, "Failed to generate state", http.StatusInternalServerError)
		return
	}

	codeVerifier, err := oauth.GenerateCodeVerifier()
	if err != nil {
		http.Error(w, "Failed to generate code verifier", http.StatusInternalServerError)
		return
	}
	codeChallenge := oauth.GenerateCodeChallenge(codeVerifier)

	h.stateStore.Save(state, codeVerifier, 10*time.Minute)
	h.writeSessionCookie(w, &oauthSession{
		State:        state,
		CodeVerifier: codeVerifier,
		ExpiresAt:    time.Now().Add(10 * time.Minute).Unix(),
	})

	redirectURI := h.getRedirectURI(r)
	authURL := h.client.GetAuthorizationURL(state, codeChallenge, redirectURI)
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (h *OAuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	queryState := r.URL.Query().Get("state")
	code := r.URL.Query().Get("code")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		http.Error(w, "Authorization failed: "+errorParam, http.StatusBadRequest)
		return
	}

	if code == "" || queryState == "" {
		http.Error(w, "Missing authorization data", http.StatusBadRequest)
		return
	}

	codeVerifier, err := h.consumeVerifierFromRequest(w, r, queryState)
	if err != nil {
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	redirectURI := h.getRedirectURI(r)
	tokenResp, err := h.client.ExchangeCodeForToken(code, codeVerifier, redirectURI)
	if err != nil {
		http.Error(w, "Failed to exchange code for token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    tokenResp.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   tokenResp.ExpiresIn,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    tokenResp.RefreshToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   tokenResp.ExpiresIn,
	})

	w.Header().Set("Content-Type", "text/html")
	html := `<!DOCTYPE html>
<html>
<head><title>Login Success</title></head>
<body>
<script>
	localStorage.setItem('auth_token', '` + tokenResp.AccessToken + `');
	const parts = '` + tokenResp.AccessToken + `'.split('.');
	if (parts.length === 3) {
		try {
			const payload = JSON.parse(atob(parts[1]));
			localStorage.setItem('auth_user', JSON.stringify({
				id: payload.user_id || payload.sub,
				username: payload.username,
				email: payload.email
			}));
		} catch(e) {
			console.error('Failed to parse token:', e);
		}
	}
	window.location.href = '/gateway/';
</script>
</body>
</html>`
	w.Write([]byte(html))
}

func (h *OAuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		MaxAge:   -1,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		MaxAge:   -1,
	})

	h.clearSessionCookie(w)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

func (h *OAuthHandler) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "No refresh token found", http.StatusUnauthorized)
		return
	}

	tokenResp, err := h.client.RefreshAccessToken(cookie.Value)
	if err != nil {
		http.Error(w, "Failed to refresh token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    tokenResp.AccessToken,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   tokenResp.ExpiresIn,
	})

	if tokenResp.RefreshToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "refresh_token",
			Value:    tokenResp.RefreshToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   h.cookieSecure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   tokenResp.ExpiresIn,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "success",
	})
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

func (h *OAuthHandler) getRedirectURI(r *http.Request) string {
	if h.publicBaseURL != "" {
		return h.publicBaseURL + "/auth/callback"
	}

	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = strings.TrimSpace(strings.Split(proto, ",")[0])
	} else if r.TLS != nil {
		scheme = "https"
	}

	host := r.Host
	if forwardedHost := r.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
		host = strings.TrimSpace(strings.Split(forwardedHost, ",")[0])
	}

	if host == "" {
		return h.client.DefaultRedirectURI()
	}

	return fmt.Sprintf("%s://%s/gateway/auth/callback", scheme, host)
}

type stateEntry struct {
	codeVerifier string
	expiresAt    time.Time
}

type stateStore struct {
	mu    sync.Mutex
	items map[string]stateEntry
}

func newStateStore() *stateStore {
	return &stateStore{
		items: make(map[string]stateEntry),
	}
}

func (s *stateStore) Save(state, codeVerifier string, ttl time.Duration) {
	if state == "" || codeVerifier == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(time.Now())
	s.items[state] = stateEntry{
		codeVerifier: codeVerifier,
		expiresAt:    time.Now().Add(ttl),
	}
}

func (s *stateStore) Consume(state string) (string, bool) {
	if state == "" {
		return "", false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupLocked(time.Now())

	entry, ok := s.items[state]
	if !ok {
		return "", false
	}
	delete(s.items, state)

	if time.Now().After(entry.expiresAt) {
		return "", false
	}

	return entry.codeVerifier, true
}

func (s *stateStore) cleanupLocked(now time.Time) {
	for key, entry := range s.items {
		if now.After(entry.expiresAt) {
			delete(s.items, key)
		}
	}
}

type oauthSession struct {
	State        string `json:"state"`
	CodeVerifier string `json:"code_verifier"`
	ExpiresAt    int64  `json:"exp"`
}

func (h *OAuthHandler) consumeVerifierFromRequest(w http.ResponseWriter, r *http.Request, queryState string) (string, error) {
	if sess, err := h.readSessionCookie(r); err == nil && sess.State == queryState && time.Now().Unix() <= sess.ExpiresAt {
		h.clearSessionCookie(w)
		return sess.CodeVerifier, nil
	}

	if verifier, ok := h.stateStore.Consume(queryState); ok {
		h.clearSessionCookie(w)
		return verifier, nil
	}
	return "", fmt.Errorf("state not found")
}

func (h *OAuthHandler) writeSessionCookie(w http.ResponseWriter, sess *oauthSession) {
	payload, err := json.Marshal(sess)
	if err != nil {
		return
	}

	value := h.signCookiePayload(payload)
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_session",
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sess.ExpiresAt - time.Now().Unix()),
	})
}

func (h *OAuthHandler) readSessionCookie(r *http.Request) (*oauthSession, error) {
	cookie, err := r.Cookie("oauth_session")
	if err != nil {
		return nil, err
	}

	payload, err := h.verifyCookiePayload(cookie.Value)
	if err != nil {
		return nil, err
	}

	var sess oauthSession
	if err := json.Unmarshal(payload, &sess); err != nil {
		return nil, err
	}
	return &sess, nil
}

func (h *OAuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "oauth_session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (h *OAuthHandler) signCookiePayload(payload []byte) string {
	mac := hmac.New(sha256.New, h.cookieSecret)
	mac.Write(payload)
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func (h *OAuthHandler) verifyCookiePayload(value string) ([]byte, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid cookie format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}

	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	mac := hmac.New(sha256.New, h.cookieSecret)
	mac.Write(payload)
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return nil, fmt.Errorf("invalid signature")
	}

	return payload, nil
}
