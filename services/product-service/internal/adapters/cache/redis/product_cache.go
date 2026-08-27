package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/model"
	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/ports"
	"github.com/redis/go-redis/v9"
)

// productCache is the Redis adapter implementing ports.ProductCache
type productCache struct {
	client *redis.Client
}

// NewProductCache creates a new Redis cache adapter
func NewProductCache(client *redis.Client) ports.ProductCache {
	return &productCache{client: client}
}

// Get retrieves a product from Redis cache
// Returns nil, nil on cache miss (not an error!)
func (c *productCache) Get(ctx context.Context, key string) (*model.Product, error) {
	data, err := c.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		// Cache miss - not an error, just not cached yet
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error getting from cache: %w", err)
	}

	// Deserialize JSON back to Product struct
	var product model.Product
	if err := json.Unmarshal(data, &product); err != nil {
		return nil, fmt.Errorf("error deserializing product: %w", err)
	}

	return &product, nil
}

// Set stores a product in Redis cache with TTL
func (c *productCache) Set(ctx context.Context, key string, product *model.Product, ttl time.Duration) error {
	// Serialize Product struct to JSON
	data, err := json.Marshal(product)
	if err != nil {
		return fmt.Errorf("error serializing product: %w", err)
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("error setting cache: %w", err)
	}

	return nil
}

// Delete removes a specific key from Redis cache
func (c *productCache) Delete(ctx context.Context, key string) error {
	if err := c.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("error deleting from cache: %w", err)
	}
	return nil
}

// DeleteByPattern removes all keys matching a pattern
// Example: "products:list:*" deletes all list cache entries
func (c *productCache) DeleteByPattern(ctx context.Context, pattern string) error {
	// SCAN is production-safe (non-blocking, iterates in chunks)
	// Never use KEYS in production - it blocks Redis!
	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("error scanning keys: %w", err)
		}

		// Delete found keys
		if len(keys) > 0 {
			if err := c.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("error deleting keys: %w", err)
			}
		}

		cursor = nextCursor
		// cursor = 0 means scan is complete
		if cursor == 0 {
			break
		}
	}

	return nil
}
