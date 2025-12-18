package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gateway-server/internal/auth"
	"gateway-server/internal/handlers"
	"gateway-server/internal/middleware"
	"gateway-server/internal/oauth"
	"gateway-server/internal/proxy"
	"gateway-server/internal/ratelimit"
	"gateway-server/internal/storage"
	"gateway-server/internal/ti"
)

func main() {
	logger := log.New(os.Stdout, "[gateway] ", log.LstdFlags)

	port := getEnv("PORT", "8000")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "gateway")
	dbPassword := getEnv("DB_PASSWORD", "gateway")
	dbName := getEnv("DB_NAME", "gateway")
	authServerURL := getEnv("AUTH_SERVER_URL", "http://auth-server:8081")
	tiServerURL := getEnv("TI_SERVER_URL", "http://ti-server:8080")
	oauthClientID := getEnv("OAUTH_CLIENT_ID", "gateway-server")
	oauthClientSecret := getEnv("OAUTH_CLIENT_SECRET", "gateway-secret-dev")
	sessionSecret := getEnv("SESSION_SECRET", "gateway-session-secret")
	sessionCookieSecure := strings.ToLower(getEnv("SESSION_COOKIE_SECURE", "false")) == "true"

	connString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	ctx := context.Background()
	store, err := storage.NewStorage(ctx, connString)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer store.Close()

	logger.Println("Connected to database")

	verifier, err := auth.NewJWTVerifier(authServerURL)
	if err != nil {
		logger.Fatalf("Failed to initialize JWT verifier: %v", err)
	}
	logger.Println("JWT verifier initialized")

	oauthClient := oauth.NewOAuthClient(
		authServerURL,
		"", // publicAuthServerURL - will be derived dynamically from request
		oauthClientID,
		oauthClientSecret,
		"", // redirectURI - will be derived dynamically from request
	)
	logger.Println("OAuth client initialized")

	tiClient := ti.NewTIClient(tiServerURL)
	logger.Println("TI client initialized")

	rateLimiter := ratelimit.NewRateLimiter()
	logger.Println("Rate limiter initialized")

	server := handlers.NewServer(store, tiClient, logger)
	oauthHandler := handlers.NewOAuthHandler(oauthClient, []byte(sessionSecret), sessionCookieSecure, "")
	authMiddleware := middleware.NewAuthMiddleware(verifier)
	proxyHandler := proxy.NewProxyHandler(store, tiClient, rateLimiter)

	startBackgroundWorkers(ctx, store, tiClient, rateLimiter, logger)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			oauthHandler.HandleLogin(w, r)
		} else if r.Method == http.MethodPost {
			server.HandleLogin(w, r, authServerURL)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			oauthHandler.HandleCallback(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			oauthHandler.HandleLogout(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			oauthHandler.HandleRefresh(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/auth/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		proxyToAuthServer(w, r, authServerURL+"/auth/register")
	})

	mux.HandleFunc("/api/virtual-services", authMiddleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			server.HandleCreateVS(w, r)
		case http.MethodGet:
			server.HandleGetVSList(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/virtual-services/", authMiddleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/virtual-services/")
		parts := strings.Split(path, "/")

		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				server.HandleGetVS(w, r)
			case http.MethodPut:
				server.HandleUpdateVS(w, r)
			case http.MethodDelete:
				server.HandleDeleteVS(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		} else if len(parts) >= 2 {
			resource := parts[1]
			switch resource {
			case "users":
				if len(parts) == 2 {
					switch r.Method {
					case http.MethodPost:
						server.HandleAddUser(w, r)
					case http.MethodGet:
						server.HandleGetUsers(w, r)
					default:
						http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					}
				} else if len(parts) == 3 {
					if r.Method == http.MethodDelete {
						server.HandleRemoveUser(w, r)
					} else {
						http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					}
				}
			case "ti-feeds":
				if len(parts) == 2 {
					switch r.Method {
					case http.MethodPost:
						server.HandleAttachFeed(w, r)
					case http.MethodGet:
						server.HandleGetFeeds(w, r)
					default:
						http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					}
				} else if len(parts) == 3 {
					switch r.Method {
					case http.MethodPut:
						server.HandleToggleFeed(w, r)
					case http.MethodDelete:
						server.HandleDetachFeed(w, r)
					default:
						http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					}
				}
			case "logs":
				if len(parts) == 2 && r.Method == http.MethodGet {
					server.HandleGetLogs(w, r)
				} else {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
			default:
				http.Error(w, "Not found", http.StatusNotFound)
			}
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))

	mux.HandleFunc("/api/ti-feeds", authMiddleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			server.HandleGetAvailableTIFeeds(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/vs/", authMiddleware.OptionalAuth(proxyHandler.ServeHTTP))

	mux.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, ".") {
			http.StripPrefix("/assets/", http.FileServer(http.Dir("./static/assets"))).ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, "./static/index.html")
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") ||
			strings.HasPrefix(r.URL.Path, "/auth/") ||
			strings.HasPrefix(r.URL.Path, "/vs/") ||
			strings.HasPrefix(r.URL.Path, "/health") {
			http.NotFound(w, r)
			return
		}

		if strings.Contains(r.URL.Path, ".") && r.URL.Path != "/" {
			http.FileServer(http.Dir("./static")).ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, "./static/index.html")
	})

	addr := ":" + port
	logger.Printf("Starting Gateway server on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Fatalf("Server failed: %v", err)
	}
}

func startBackgroundWorkers(ctx context.Context, store *storage.Storage, tiClient *ti.TIClient, rateLimiter *ratelimit.RateLimiter, logger *log.Logger) {
	go func() {
		logger.Println("Starting IOC cache updater worker")

		feeds := getAllActiveFeeds(ctx, store)
		if len(feeds) > 0 {
			if err := tiClient.UpdateCacheWithKeys(feeds); err != nil {
				logger.Printf("Error during initial IOC cache update: %v", err)
			} else {
				logger.Printf("Initial IOC cache updated with %d feeds", len(feeds))
			}
		}

		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				feeds := getAllActiveFeeds(ctx, store)
				if len(feeds) > 0 {
					if err := tiClient.UpdateCacheWithKeys(feeds); err != nil {
						logger.Printf("Error updating IOC cache: %v", err)
					} else {
						logger.Printf("IOC cache updated with %d feeds", len(feeds))
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		logger.Println("Starting old logs cleaner worker")
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := store.CleanOldLogs(ctx); err != nil {
					logger.Printf("Error cleaning old logs: %v", err)
					continue
				}

				if err := store.CleanExpiredData(ctx, time.Hour); err != nil {
					logger.Printf("Error removing expired gateway data: %v", err)
					continue
				}

				logger.Println("Gateway data cleanup finished")
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		logger.Println("Starting rate limiter cleanup worker")
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				rateLimiter.Cleanup(30 * time.Minute)
				logger.Println("Rate limiter cleaned up")
			case <-ctx.Done():
				return
			}
		}
	}()
}

func getAllActiveFeeds(ctx context.Context, store *storage.Storage) []ti.FeedWithKey {
	feeds, err := store.GetAllActiveVSTIFeeds(ctx)
	if err != nil {
		return []ti.FeedWithKey{}
	}

	var result []ti.FeedWithKey
	for _, feed := range feeds {
		apiKey := ""
		if feed.APIKey != nil {
			apiKey = *feed.APIKey
		}
		result = append(result, ti.FeedWithKey{
			FeedID: feed.FeedID,
			APIKey: apiKey,
		})
	}

	return result
}

func proxyToAuthServer(w http.ResponseWriter, r *http.Request, targetURL string) {
	client := &http.Client{Timeout: 10 * time.Second}

	proxyReq, err := http.NewRequest(r.Method, targetURL, r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	for key, values := range r.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, "Failed to reach auth server", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			w.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
