package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	redisAdapter "github.com/bishtpramod19/ecommerce-platform/services/inventory-service/internal/adapters/lock/redis"
	postgresAdapter "github.com/bishtpramod19/ecommerce-platform/services/inventory-service/internal/adapters/repository/postgres"
	"github.com/bishtpramod19/ecommerce-platform/services/inventory-service/internal/config"
	grpcserver "github.com/bishtpramod19/ecommerce-platform/services/inventory-service/internal/grpc/server"
	"github.com/bishtpramod19/ecommerce-platform/services/inventory-service/internal/service"
	inventorypb "github.com/bishtpramod19/ecommerce-protos/inventory"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
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
	logger := zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", "inventory-service").
		Logger()

	logger.Info().Msg("starting inventory-service")

	// ─── PostgreSQL ───────────────────────────────────────
	db, err := postgresAdapter.NewPostgresDB(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("error connecting to PostgreSQL")
	}
	defer db.Close()
	logger.Info().Msg("connected to PostgreSQL")

	// ─── Redis ────────────────────────────────────────────
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})
	logger.Info().Msg("connected to Redis")

	// ─── Wire Up Layers ───────────────────────────────────
	inventoryStore := postgresAdapter.NewInventoryRepository(db)
	distributedLock := redisAdapter.NewRedisLock(redisClient)
	inventorySvc := service.NewInventoryService(inventoryStore, distributedLock, cfg.LockTTLSeconds)

	// ─── gRPC Server ──────────────────────────────────────
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		logger.Fatal().Err(err).Msgf("failed to listen on port %s", cfg.GRPCPort)
	}

	grpcSrv := grpc.NewServer()

	inventoryGRPCServer := grpcserver.NewInventoryGRPCServer(inventorySvc)
	inventorypb.RegisterInventoryServiceServer(grpcSrv, inventoryGRPCServer)

	reflection.Register(grpcSrv)

	// ─── Start Server ─────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info().Str("port", cfg.GRPCPort).Msg("gRPC server starting")
		if err := grpcSrv.Serve(lis); err != nil {
			logger.Fatal().Err(err).Msg("gRPC server error")
		}
	}()

	// Wait for shutdown signal
	<-quit
	logger.Info().Msg("shutting down inventory-service...")
	grpcSrv.GracefulStop()
	logger.Info().Msg("inventory-service stopped")
}
