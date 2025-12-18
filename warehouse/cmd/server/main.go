package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"warehouse/internal/auth"
	"warehouse/internal/gateway"
	"warehouse/internal/handlers"
	"warehouse/internal/middleware"
	"warehouse/internal/oauth"
	"warehouse/internal/storage"
)

func main() {
	logger := log.New(os.Stdout, "[warehouse] ", log.LstdFlags)

	port := getEnv("PORT", "8082")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "assetserver")
	dbPassword := getEnv("DB_PASSWORD", "assetserver")
	dbName := getEnv("DB_NAME", "assetserver")
	authServerURL := getEnv("AUTH_SERVER_URL", "http://auth-server:8081")
	gatewayServerURL := getEnv("GATEWAY_SERVER_URL", "http://gateway-server:8000")
	assetServerURL := getEnv("ASSET_SERVER_URL", "http://warehouse:8082")
	sessionSecret := getEnv("SESSION_SECRET", "dev-session-secret")
	sessionCookieSecure := strings.ToLower(getEnv("SESSION_COOKIE_SECURE", "false")) == "true"
	oauthClientID := getEnv("OAUTH_CLIENT_ID", "warehouse")
	oauthClientSecret := getEnv("OAUTH_CLIENT_SECRET", "warehouse-secret-dev")

	connString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	ctx := context.Background()
	store, err := storage.NewStorage(ctx, connString)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer store.Close()

	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()
	startCleanupWorker(cleanupCtx, store, logger)

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
	oauthHandler := handlers.NewOAuthHandler(oauthClient, []byte(sessionSecret), sessionCookieSecure, "")
	logger.Println("OAuth client initialized")

	gatewayClient := gateway.NewGatewayClient(gatewayServerURL)
	logger.Println("Gateway client initialized")

	server := handlers.NewServer(store, logger, authServerURL)
	authMiddleware := middleware.NewAuthMiddleware(verifier)

	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/api/users/search", func(w http.ResponseWriter, r *http.Request) {
		authMiddleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				server.HandleSearchUsers(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})(w, r)
	})

	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			oauthHandler.HandleLogin(w, r)
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

	mux.HandleFunc("/api/realms", func(w http.ResponseWriter, r *http.Request) {
		authMiddleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				server.HandleCreateRealm(w, r)
			} else if r.Method == http.MethodGet {
				server.HandleGetRealms(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})(w, r)
	})

	mux.HandleFunc("/api/realms/", func(w http.ResponseWriter, r *http.Request) {
		authMiddleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			path := strings.TrimPrefix(r.URL.Path, "/api/realms/")
			parts := strings.Split(path, "/")

			if len(parts) == 1 {
				if r.Method == http.MethodGet {
					server.HandleGetRealm(w, r)
				} else if r.Method == http.MethodPut {
					server.HandleUpdateRealm(w, r)
				} else if r.Method == http.MethodDelete {
					server.HandleDeleteRealm(w, r)
				} else {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
			} else if len(parts) >= 2 {
				if parts[1] == "users" {
					if len(parts) == 2 {
						if r.Method == http.MethodPost {
							server.HandleAddRealmUser(w, r)
						} else if r.Method == http.MethodGet {
							server.HandleGetRealmUsers(w, r)
						} else {
							http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
						}
					} else if len(parts) == 3 {
						if r.Method == http.MethodDelete {
							server.HandleRemoveRealmUser(w, r)
						} else {
							http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
						}
					} else if len(parts) == 4 && parts[3] == "assets" {
						if r.Method == http.MethodGet {
							server.HandleGetUserAssets(w, r)
						} else if r.Method == http.MethodPost {
							server.HandleReassignUserAssets(w, r)
						} else if r.Method == http.MethodDelete {
							server.HandleDeleteUserAssets(w, r)
						} else {
							http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
						}
					}
				} else if parts[1] == "assets" {
					if r.Method == http.MethodPost {
						server.HandleCreateAsset(w, r)
					} else if r.Method == http.MethodGet {
						server.HandleGetAssets(w, r)
					} else {
						http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					}
				} else if parts[1] == "gateway-protection" {
					if r.Method == http.MethodPost {
						server.HandleCreateGatewayProtection(w, r, gatewayClient, assetServerURL)
					} else if r.Method == http.MethodDelete {
						server.HandleRemoveGatewayProtection(w, r, gatewayClient)
					} else {
						http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					}
				} else {
					http.Error(w, "Not found", http.StatusNotFound)
				}
			} else {
				http.Error(w, "Not found", http.StatusNotFound)
			}
		})(w, r)
	})

	mux.HandleFunc("/api/assets/", func(w http.ResponseWriter, r *http.Request) {
		authMiddleware.RequireAuth(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				server.HandleGetAsset(w, r)
			} else if r.Method == http.MethodPut {
				server.HandleUpdateAsset(w, r)
			} else if r.Method == http.MethodDelete {
				server.HandleDeleteAsset(w, r)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
		})(w, r)
	})

	mux.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, ".") {
			http.StripPrefix("/assets/", http.FileServer(http.Dir("./static/assets"))).ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, "./static/index.html")
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, ".") && r.URL.Path != "/" {
			http.FileServer(http.Dir("./static")).ServeHTTP(w, r)
			return
		}

		http.ServeFile(w, r, "./static/index.html")
	})

	addr := ":" + port
	logger.Printf("Starting server on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		logger.Fatalf("Server failed: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func startCleanupWorker(ctx context.Context, store *storage.Storage, logger *log.Logger) {
	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := store.CleanExpiredData(context.Background(), time.Hour); err != nil {
					logger.Printf("Failed to clean warehouse data: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
