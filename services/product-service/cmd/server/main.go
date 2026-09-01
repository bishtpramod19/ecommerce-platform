package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	redisAdapter "github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/adapters/cache/redis"
	mongoAdapter "github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/adapters/repository/mongodb"
	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/config"
	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/grpc/client"
	grpcserver "github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/grpc/server"
	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/handler"
	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/middleware"
	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/service"
	productpb "github.com/bishtpramod19/ecommerce-protos/product"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// ─── Config ───────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("error loading config: %v\n", err)
		os.Exit(1)
	}

	// ─── Logger ───────────────────────────────────────────
	// zerolog outputs structured JSON logs
	// Foundation for Loki log aggregation (Phase 1.5)
	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", "product-service").
		Logger()

	log.Logger = logger
	logger.Info().Msg("starting product-service")

	// ─── MongoDB ──────────────────────────────────────────
	mongoDB, err := mongoAdapter.NewMongoDBClient(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("error connecting to MongoDB")
	}
	logger.Info().Msg("connected to MongoDB")

	// ─── Redis ────────────────────────────────────────────
	redisClient, err := redisAdapter.NewRedisClient(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("error connecting to Redis")
	}
	logger.Info().Msg("connected to Redis")

	// ─── gRPC Client (calls user-service) ─────────────────
	userClient, err := client.NewUserServiceClient(cfg.UserServiceAddr)
	if err != nil {
		logger.Fatal().Err(err).Msg("error connecting to user-service")
	}
	logger.Info().Str("addr", cfg.UserServiceAddr).Msg("connected to user-service gRPC")

	// ─── Wire Up Layers ───────────────────────────────────
	productStore := mongoAdapter.NewProductRepository(mongoDB)
	productCache := redisAdapter.NewProductCache(redisClient)
	productSvc := service.NewProductService(productStore, productCache)
	productHandler := handler.NewProductHandler(productSvc)

	// ─── Rate Limiter ─────────────────────────────────────
	// 100 requests burst, 10 requests/sec sustained
	rateLimiter := middleware.NewRateLimiter(100, 10)

	// ─── Start gRPC Server ────────────────────────────────
	go func() {
		if err := startGRPCServer(cfg, productSvc, logger); err != nil {
			logger.Fatal().Err(err).Msg("gRPC server error")
		}
	}()

	// ─── HTTP Router ──────────────────────────────────────
	r := chi.NewRouter()

	// Global middleware
	r.Use(chiMiddleware.RequestID)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.Timeout(30 * time.Second))
	r.Use(middleware.LoggingMiddleware(logger))
	r.Use(middleware.RateLimitMiddleware(rateLimiter))

	// Health check (public)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok", "service": "product-service"}`))
	})

	// Public routes (no auth required)
	r.Route("/v1/products", func(r chi.Router) {
		r.Get("/", productHandler.ListProducts)
		r.Get("/search", productHandler.SearchProducts)
		r.Get("/{id}", productHandler.GetProduct)

		// Protected routes (auth required)
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(cfg.JWTSecret))
			r.Post("/", productHandler.CreateProduct)
			r.Put("/{id}", productHandler.UpdateProduct)
			r.Delete("/{id}", productHandler.DeleteProduct)
		})
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

	// Wait for shutdown signal
	<-quit
	logger.Info().Msg("shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal().Err(err).Msg("error shutting down HTTP server")
	}

	logger.Info().Msg("servers stopped")
	_ = userClient
}

// startGRPCServer starts the gRPC server on configured port.
func startGRPCServer(cfg *config.Config, productSvc *service.ProductService, logger zerolog.Logger) error {
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", cfg.GRPCPort, err)
	}

	grpcSrv := grpc.NewServer()

	// Register product gRPC server
	productGRPCServer := grpcserver.NewProductGRPCServer(productSvc)
	productpb.RegisterProductServiceServer(grpcSrv, productGRPCServer)

	// Enable reflection (for grpcurl testing)
	reflection.Register(grpcSrv)

	logger.Info().Str("port", cfg.GRPCPort).Msg("gRPC server starting")
	return grpcSrv.Serve(lis)
}
