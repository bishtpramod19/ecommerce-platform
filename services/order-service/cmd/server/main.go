package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	postgresAdapter "github.com/bishtpramod19/ecommerce-platform/services/order-service/internal/adapters/repository/postgres"
	"github.com/bishtpramod19/ecommerce-platform/services/order-service/internal/config"
	"github.com/bishtpramod19/ecommerce-platform/services/order-service/internal/grpc/client"
	"github.com/bishtpramod19/ecommerce-platform/services/order-service/internal/handler"
	"github.com/bishtpramod19/ecommerce-platform/services/order-service/internal/middleware"
	"github.com/bishtpramod19/ecommerce-platform/services/order-service/internal/service"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

func main() {
	// ─── Config ───────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("error loading config: %v\n", err)
		os.Exit(1)
	}

	// ─── Logger ───────────────────────────────────────────
	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", "order-service").
		Logger()

	logger.Info().Msg("starting order-service")

	// ─── PostgreSQL ───────────────────────────────────────
	db, err := postgresAdapter.NewPostgresDB(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("error connecting to PostgreSQL")
	}
	defer db.Close()
	logger.Info().Msg("connected to PostgreSQL")

	// ─── gRPC Clients (connect to downstream services) ────
	userClient, err := client.NewUserClient(cfg.UserServiceAddr)
	if err != nil {
		logger.Fatal().Err(err).Msg("error connecting to user-service")
	}
	logger.Info().Str("addr", cfg.UserServiceAddr).Msg("connected to user-service")

	productClient, err := client.NewProductClient(cfg.ProductServiceAddr)
	if err != nil {
		logger.Fatal().Err(err).Msg("error connecting to product-service")
	}
	logger.Info().Str("addr", cfg.ProductServiceAddr).Msg("connected to product-service")

	inventoryClient, err := client.NewInventoryClient(cfg.InventoryServiceAddr)
	if err != nil {
		logger.Fatal().Err(err).Msg("error connecting to inventory-service")
	}
	logger.Info().Str("addr", cfg.InventoryServiceAddr).Msg("connected to inventory-service")

	// ─── Wire Up Layers ───────────────────────────────────
	orderStore := postgresAdapter.NewOrderRepository(db)
	orderSvc := service.NewOrderService(orderStore, userClient, productClient, inventoryClient)
	orderHandler := handler.NewOrderHandler(orderSvc)

	// ─── HTTP Router ──────────────────────────────────────
	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(30 * time.Second))
	r.Use(middleware.MetricsMiddleware("order-service"))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "service": "order-service"}`))
	})

	r.Handle("/metrics", promhttp.Handler())

	r.Route("/v1/orders", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware(cfg.JWTSecret))
		r.Post("/", orderHandler.CreateOrder)
		r.Get("/", orderHandler.ListMyOrders)
		r.Get("/{id}", orderHandler.GetOrder)
	})

	// ─── HTTP Server ──────────────────────────────────────
	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info().Str("port", cfg.ServerPort).Msg("HTTP server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("HTTP server error")
		}
	}()

	<-quit
	logger.Info().Msg("shutting down order-service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("error shutting down HTTP server")
	}

	logger.Info().Msg("order-service stopped")
}
