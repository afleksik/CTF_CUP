package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"auth-server/internal/auth"
	"auth-server/internal/handlers"
	"auth-server/internal/middleware"
	"auth-server/internal/storage"
)

func main() {
	logger := log.New(os.Stdout, "[auth-server] ", log.LstdFlags)

	port := getEnv("PORT", "8081")
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "authserver")
	dbPassword := getEnv("DB_PASSWORD", "authserver")
	dbName := getEnv("DB_NAME", "authserver")
	jwtKeyPath := getEnv("JWT_KEY_PATH", "/app/keys/jwt_key.pem")
	warehouseClientSecret := getEnv("WAREHOUSE_CLIENT_SECRET", "warehouse-secret-dev")
	gatewayClientSecret := getEnv("GATEWAY_CLIENT_SECRET", "gateway-secret-dev")

	connString := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	ctx := context.Background()
	userStorage, err := storage.NewStorage(ctx, connString)
	if err != nil {
		logger.Fatalf("Failed to connect to database: %v", err)
	}
	defer userStorage.Close()

	cleanupCtx, cleanupCancel := context.WithCancel(context.Background())
	defer cleanupCancel()
	startCleanupWorker(cleanupCtx, userStorage, logger)

	logger.Println("Connected to database")

	if err := syncOAuthClientSecrets(ctx, userStorage, map[string]string{
		"warehouse":      warehouseClientSecret,
		"gateway-server": gatewayClientSecret,
	}); err != nil {
		logger.Fatalf("Failed to sync OAuth client secrets: %v", err)
	}
	logger.Println("OAuth client secrets synchronized")

	privateKey, err := auth.GenerateOrLoadRSAKey(jwtKeyPath)
	if err != nil {
		logger.Fatalf("Failed to load RSA key: %v", err)
	}
	logger.Println("RSA key loaded")

	publicKeyPEM, err := auth.GetPublicKeyPEM(privateKey)
	if err != nil {
		logger.Fatalf("Failed to get public key: %v", err)
	}

	jwtManager := auth.NewJWTManager(privateKey)

	server := handlers.NewServer(userStorage, jwtManager, publicKeyPEM, logger)
	authMiddleware := middleware.NewAuthMiddleware(jwtManager)

	mux := http.NewServeMux()

	mux.HandleFunc("/auth/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			server.HandleRegister(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			server.HandleLogin(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/auth/authorize", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodPost {
			server.HandleAuthorize(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/auth/token", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			server.HandleToken(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/auth/refresh", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			server.HandleRefresh(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/auth/public-key", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			server.HandleGetPublicKey(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/auth/profile", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			authMiddleware.RequireAuth(server.HandleGetProfile)(w, r)
		} else if r.Method == http.MethodPut {
			authMiddleware.RequireAuth(server.HandleUpdateProfile)(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			authMiddleware.RequireAuth(server.HandleLogout)(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/users/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			authMiddleware.InternalOnly(server.HandleSearchUsers)(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			authMiddleware.RequireAuth(server.HandleGetUsers)(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/users/")
		if path != "" && r.Method == http.MethodGet {
			authMiddleware.RequireAuth(server.HandleGetUser)(w, r)
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/assets/", fs)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/auth") && !strings.HasPrefix(r.URL.Path, "/users") && !strings.HasPrefix(r.URL.Path, "/health") && !strings.HasPrefix(r.URL.Path, "/assets") {
			http.ServeFile(w, r, "./static/index.html")
		} else if strings.HasPrefix(r.URL.Path, "/assets") {
			fs.ServeHTTP(w, r)
		}
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

func syncOAuthClientSecrets(ctx context.Context, store *storage.Storage, secrets map[string]string) error {
	for clientID, secret := range secrets {
		if secret == "" {
			continue
		}
		if err := store.UpdateOAuthClientSecret(ctx, clientID, secret); err != nil {
			return fmt.Errorf("update secret for %s: %w", clientID, err)
		}
	}
	return nil
}

func startCleanupWorker(ctx context.Context, store *storage.Storage, logger *log.Logger) {
	ticker := time.NewTicker(5 * time.Minute)

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := store.CleanExpiredData(context.Background(), time.Hour); err != nil {
					logger.Printf("Failed to clean auth data: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
