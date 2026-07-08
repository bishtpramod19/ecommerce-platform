package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"database/sql"

	"github.com/bishtpramod19/ecommerce-platform/services/user-service/internal/adapters/repository/postgres"
	"github.com/bishtpramod19/ecommerce-platform/services/user-service/internal/config"
	"github.com/bishtpramod19/ecommerce-platform/services/user-service/internal/handler"
	"github.com/bishtpramod19/ecommerce-platform/services/user-service/internal/middleware"
	"github.com/bishtpramod19/ecommerce-platform/services/user-service/internal/service"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("error loading config: %v\n", err)
		os.Exit(1)
	}

	// Connect to database
	db, err := connectWithRetry(cfg)
	if err != nil {
		fmt.Printf("error connecting to database after retries: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Wire up layers
	// Repository (adapter) → Service → Handler
	userRepo := postgres.NewUserRepository(db)
	userSvc := service.NewUserService(userRepo, cfg.JWTSecret, cfg.JWTExpiryHours)
	userHandler := handler.NewUserHandler(userSvc)

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Logger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(30 * time.Second))

	// Health check (no auth required)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})

	// Public routes (no auth required)
	r.Route("/v1/auth", func(r chi.Router) {
		r.Post("/register", userHandler.Register)
		r.Post("/login", userHandler.Login)
	})

	// Protected routes (auth required)
	r.Route("/v1/users", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		r.Get("/{id}", userHandler.GetUser)
		r.Put("/{id}", userHandler.UpdateUser)
	})

	// Start server
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		fmt.Printf("user-service starting on port %s\n", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("error starting server: %v\n", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	<-quit
	fmt.Println("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("error shutting down server: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("server stopped")
}

// connectWithRetry retries database connection with exponential backoff.
// This handles cases where PostgreSQL starts after our service,
// or temporarily goes down and comes back up.
func connectWithRetry(cfg *config.Config) (*sql.DB, error) {
    var db *sql.DB
    var err error

    maxRetries := 10
    for i := 1; i <= maxRetries; i++ {
        db, err = postgres.NewPostgresDB(cfg)
        if err == nil {
            return db, nil
        }

        if i == maxRetries {
            break
        }

        // Exponential backoff: 2s, 4s, 8s, 16s... max 30s
        waitTime := time.Duration(min(2<<i, 30)) * time.Second
        fmt.Printf("DB connection attempt %d/%d failed: %v. Retrying in %v...\n",
            i, maxRetries, err, waitTime)
        time.Sleep(waitTime)
    }

    return nil, fmt.Errorf("failed to connect after %d attempts: %w", maxRetries, err)
}
