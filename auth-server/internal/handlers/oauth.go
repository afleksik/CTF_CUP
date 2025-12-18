package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"auth-server/internal/auth"
	"auth-server/internal/models"
)

const (
	csrfCookieName = "oauth_csrf"
	csrfCookiePath = "/auth" // Broader path to work with ingress rewrite
)

func (s *Server) HandleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := r.URL.Query().Get("client_id")
	rawRedirectURI := r.URL.Query().Get("redirect_uri")
	responseType := r.URL.Query().Get("response_type")
	state := r.URL.Query().Get("state")
	codeChallenge := r.URL.Query().Get("code_challenge")
	codeChallengeMethod := r.URL.Query().Get("code_challenge_method")

	if clientID == "" || rawRedirectURI == "" || responseType == "" {
		http.Error(w, "Missing required parameters", http.StatusBadRequest)
		return
	}

	if responseType != "code" {
		http.Error(w, "Unsupported response_type", http.StatusBadRequest)
		return
	}

	client, err := s.storage.GetOAuthClient(r.Context(), clientID)
	if err != nil {
		http.Error(w, "Invalid client_id", http.StatusBadRequest)
		return
	}

	redirectURI, err := matchRedirectURI(rawRedirectURI, client.RedirectURIs)
	if err != nil {
		http.Error(w, "Invalid redirect_uri", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodGet {
		s.renderAuthorizeForm(w, r, clientID, redirectURI, state, codeChallenge, codeChallengeMethod, client.Name, "")
		return
	}

	if err := s.verifyCSRFToken(r); err != nil {
		http.Error(w, "Invalid CSRF token", http.StatusBadRequest)
		return
	}
	s.clearCSRFCookie(w)

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		s.renderAuthorizeForm(w, r, clientID, redirectURI, state, codeChallenge, codeChallengeMethod, client.Name, "Username and password are required")
		return
	}

	user, err := s.storage.GetUserByUsername(r.Context(), username)
	if err != nil {
		s.renderAuthorizeForm(w, r, clientID, redirectURI, state, codeChallenge, codeChallengeMethod, client.Name, "Invalid username or password")
		return
	}

	if !auth.CheckPassword(password, user.PasswordHash) {
		s.renderAuthorizeForm(w, r, clientID, redirectURI, state, codeChallenge, codeChallengeMethod, client.Name, "Invalid username or password")
		return
	}

	code, err := generateRandomString(32)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var codeChallengePtr *string
	var codeChallengeMethodPtr *string
	if codeChallenge != "" {
		if codeChallengeMethod == "" {
			codeChallengeMethod = "S256"
		}
		if codeChallengeMethod != "S256" {
			http.Error(w, "Unsupported code_challenge_method", http.StatusBadRequest)
			return
		}
		codeChallengePtr = &codeChallenge
		method := codeChallengeMethod
		codeChallengeMethodPtr = &method
	} else if codeChallengeMethod != "" {
		http.Error(w, "code_challenge required when code_challenge_method provided", http.StatusBadRequest)
		return
	}

	authCode := &models.AuthCode{
		Code:                code,
		UserID:              user.ID,
		ClientID:            clientID,
		RedirectURI:         redirectURI,
		State:               state,
		CodeChallenge:       codeChallengePtr,
		CodeChallengeMethod: codeChallengeMethodPtr,
		ExpiresAt:           time.Now().Add(10 * time.Minute),
		CreatedAt:           time.Now(),
	}

	if err := s.storage.CreateAuthCode(r.Context(), authCode); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	redirectURL, err := url.Parse(redirectURI)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	query := redirectURL.Query()
	query.Set("code", code)
	if state != "" {
		query.Set("state", state)
	}
	redirectURL.RawQuery = query.Encode()
	s.clearCSRFCookie(w)
	http.Redirect(w, r, redirectURL.String(), http.StatusFound)
}

func (s *Server) HandleToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	grantType := r.FormValue("grant_type")
	code := r.FormValue("code")
	redirectURI := r.FormValue("redirect_uri")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	codeVerifier := r.FormValue("code_verifier")

	if grantType != "authorization_code" {
		s.respondError(w, http.StatusBadRequest, "Unsupported grant_type")
		return
	}

	client, err := s.storage.GetOAuthClient(r.Context(), clientID)
	if err != nil {
		s.respondError(w, http.StatusUnauthorized, "Invalid client")
		return
	}

	if client.ClientSecret != clientSecret {
		s.respondError(w, http.StatusUnauthorized, "Invalid client credentials")
		return
	}

	authCode, err := s.storage.GetAuthCode(r.Context(), code)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if authCode.ClientID != clientID {
		s.respondError(w, http.StatusBadRequest, "Invalid code")
		return
	}

	normalizedRedirectURI, err := normalizeRedirectURI(redirectURI)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid redirect_uri")
		return
	}

	if authCode.RedirectURI != normalizedRedirectURI {
		s.respondError(w, http.StatusBadRequest, "Invalid redirect_uri")
		return
	}

	if authCode.CodeChallenge != nil && *authCode.CodeChallenge != "" {
		if codeVerifier == "" {
			s.respondError(w, http.StatusBadRequest, "code_verifier required")
			return
		}

		method := "S256"
		if authCode.CodeChallengeMethod != nil && *authCode.CodeChallengeMethod != "" {
			method = *authCode.CodeChallengeMethod
		}

		if method != "S256" {
			s.respondError(w, http.StatusBadRequest, "Unsupported code_challenge_method")
			return
		}

		hash := sha256.Sum256([]byte(codeVerifier))
		challenge := base64.RawURLEncoding.EncodeToString(hash[:])
		valid := challenge == *authCode.CodeChallenge

		if !valid {
			s.respondError(w, http.StatusBadRequest, "Invalid code_verifier")
			return
		}
	}

	user, err := s.storage.GetUserByID(r.Context(), authCode.UserID)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "User not found")
		return
	}

	accessToken, err := s.jwtManager.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	refreshTokenStr, err := generateRandomString(64)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to generate refresh token")
		return
	}

	refreshToken := &models.RefreshToken{
		Token:     refreshTokenStr,
		UserID:    user.ID,
		ClientID:  clientID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := s.storage.CreateRefreshToken(r.Context(), refreshToken); err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to create refresh token")
		return
	}

	response := models.TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour
		RefreshToken: refreshTokenStr,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	grantType := r.FormValue("grant_type")
	refreshTokenStr := r.FormValue("refresh_token")
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	if refreshTokenStr == "" {
		s.respondError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	if grantType != "refresh_token" {
		s.respondError(w, http.StatusBadRequest, "Unsupported grant_type")
		return
	}

	client, err := s.storage.GetOAuthClient(r.Context(), clientID)
	if err != nil {
		s.respondError(w, http.StatusUnauthorized, "Invalid client")
		return
	}

	if client.ClientSecret != clientSecret {
		s.respondError(w, http.StatusUnauthorized, "Invalid client credentials")
		return
	}

	refreshToken, err := s.storage.GetRefreshToken(r.Context(), refreshTokenStr)
	if err != nil {
		s.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if refreshToken.ClientID != clientID {
		s.respondError(w, http.StatusBadRequest, "Invalid refresh token")
		return
	}

	user, err := s.storage.GetUserByID(r.Context(), refreshToken.UserID)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "User not found")
		return
	}

	accessToken, err := s.jwtManager.GenerateToken(user.ID, user.Username, user.Email)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	if err := s.storage.RevokeRefreshToken(r.Context(), refreshToken.Token); err != nil {
		s.logger.Printf("Error revoking refresh token: %v", err)
	}

	newRefreshTokenValue, err := generateRandomString(64)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to generate refresh token")
		return
	}

	newRefreshToken := &models.RefreshToken{
		Token:     newRefreshTokenValue,
		UserID:    user.ID,
		ClientID:  clientID,
		ExpiresAt: time.Now().Add(1 * time.Hour),
		CreatedAt: time.Now(),
	}

	if err := s.storage.CreateRefreshToken(r.Context(), newRefreshToken); err != nil {
		s.respondError(w, http.StatusInternalServerError, "Failed to create refresh token")
		return
	}

	response := models.TokenResponse{
		AccessToken:  accessToken,
		TokenType:    "Bearer",
		ExpiresIn:    3600, // 1 hour
		RefreshToken: newRefreshTokenValue,
	}

	s.respondJSON(w, http.StatusOK, response)
}

func (s *Server) renderAuthorizeForm(w http.ResponseWriter, r *http.Request, clientID, redirectURI, state, codeChallenge, codeChallengeMethod, clientName, errorMessage string) {
	csrfToken, err := s.issueCSRFCookie(w)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	s.serveAuthorizePage(w, r, clientID, redirectURI, state, codeChallenge, codeChallengeMethod, clientName, csrfToken, errorMessage)
}

func (s *Server) serveAuthorizePage(w http.ResponseWriter, r *http.Request, clientID, redirectURI, state, codeChallenge, codeChallengeMethod, clientName, csrfToken, errorMessage string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)

	// Build form action URL - need to prepend /auth for ingress rewrite
	// Ingress rewrites /auth/auth/authorize -> /auth/authorize
	actionURL := "/auth" + r.URL.String()

	html := `<!DOCTYPE html>
<html>
<head>
    <title>Authorization Required</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            margin: 0;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            padding: 40px;
            max-width: 400px;
            width: 100%;
        }
        h1 {
            margin: 0 0 10px 0;
            color: #333;
            font-size: 24px;
        }
        p {
            color: #666;
            margin: 0 0 30px 0;
        }
        .client-name {
            font-weight: 600;
            color: #667eea;
        }
        .form-group {
            margin-bottom: 20px;
        }
        label {
            display: block;
            margin-bottom: 5px;
            color: #333;
            font-weight: 500;
        }
        input[type="text"],
        input[type="password"] {
            width: 100%;
            padding: 12px;
            border: 1px solid #ddd;
            border-radius: 6px;
            font-size: 14px;
            box-sizing: border-box;
        }
        input[type="text"]:focus,
        input[type="password"]:focus {
            outline: none;
            border-color: #667eea;
        }
        button {
            width: 100%;
            padding: 12px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 6px;
            font-size: 16px;
            font-weight: 600;
            cursor: pointer;
            transition: transform 0.2s;
        }
        button:hover {
            transform: translateY(-2px);
        }
        .error {
            color: #d14343;
            background: #ffe6e6;
            border-radius: 6px;
            padding: 10px 12px;
            margin-bottom: 20px;
            font-size: 14px;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Sign In</h1>
        <p><span class="client-name">` + clientName + `</span> wants to access your account</p>
        ` + renderAuthorizeError(errorMessage) + `

        <form method="POST" action="` + actionURL + `">
            <input type="hidden" name="csrf_token" value="` + csrfToken + `">
            <div class="form-group">
                <label for="username">Username</label>
                <input type="text" id="username" name="username" required autofocus>
            </div>

            <div class="form-group">
                <label for="password">Password</label>
                <input type="password" id="password" name="password" required>
            </div>

            <button type="submit">Authorize</button>
        </form>
    </div>
</body>
</html>`

	w.Write([]byte(html))
}

func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

func renderAuthorizeError(message string) string {
	if message == "" {
		return ""
	}
	return `<div class="error">` + message + `</div>`
}

func (s *Server) issueCSRFCookie(w http.ResponseWriter) (string, error) {
	token, err := generateRandomString(32)
	if err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     csrfCookiePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return token, nil
}

func (s *Server) verifyCSRFToken(r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	formToken := r.FormValue("csrf_token")
	if formToken == "" {
		return errors.New("missing csrf token")
	}
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(formToken), []byte(cookie.Value)) != 1 {
		return errors.New("csrf token mismatch")
	}
	return nil
}

func (s *Server) clearCSRFCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    "",
		Path:     csrfCookiePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func matchRedirectURI(requested string, allowed []string) (string, error) {
	normalizedRequested, err := normalizeRedirectURI(requested)
	if err != nil {
		return "", err
	}

	requestedURL, err := url.Parse(normalizedRequested)
	if err != nil {
		return "", err
	}

	for _, candidate := range allowed {
		// Handle wildcard pattern: */path
		if strings.HasPrefix(candidate, "*/") {
			wildcardPath := strings.TrimPrefix(candidate, "*")
			if requestedURL.Path == wildcardPath {
				return normalizedRequested, nil
			}
			continue
		}

		normalizedCandidate, err := normalizeRedirectURI(candidate)
		if err != nil {
			continue
		}
		if normalizedRequested == normalizedCandidate {
			return normalizedRequested, nil
		}
	}
	return "", errors.New("redirect URI not registered for client")
}

func normalizeRedirectURI(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("redirect_uri must use http or https")
	}
	if parsed.Host == "" {
		return "", errors.New("redirect_uri missing host")
	}
	if parsed.User != nil {
		return "", errors.New("redirect_uri must not include user info")
	}
	if parsed.Fragment != "" {
		return "", errors.New("redirect_uri must not include fragment")
	}

	path := parsed.EscapedPath()
	if path == "" {
		path = "/"
	}

	normalized := parsed.Scheme + "://" + parsed.Host + path
	if parsed.RawQuery != "" {
		normalized += "?" + parsed.RawQuery
	}

	return normalized, nil
}
