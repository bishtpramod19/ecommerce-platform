package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/config"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates and verifies a Redis client connection.
func NewRedisClient(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0, // default DB

		// Connection pool settings
		PoolSize:     20,
		MinIdleConns: 5,

		// Timeouts
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("error connecting to Redis: %w", err)
	}

	log.Println("Successfully connected to Redis")
	return client, nil
}
