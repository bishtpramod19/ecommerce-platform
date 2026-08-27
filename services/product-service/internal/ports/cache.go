package ports

import (
	"context"
	"time"

	"github.com/bishtpramod19/ecommerce-platform/services/product-service/internal/model"
)

type ProductCache interface {
	// Get retrieves a product from cache by ID.
	// Returns nil, nil if not found (cache miss).
	Get(ctx context.Context, key string) (*model.Product, error)

	// Set stores a product in cache with TTL.
	Set(ctx context.Context, key string, product *model.Product, ttl time.Duration) error

	// Delete removes a product from cache.
	// Called when product is updated or deleted.
	Delete(ctx context.Context, key string) error

	// DeleteByPattern removes all cache entries matching a pattern.
	// Used to invalidate cache when product list changes.
	DeleteByPattern(ctx context.Context, pattern string) error
}
