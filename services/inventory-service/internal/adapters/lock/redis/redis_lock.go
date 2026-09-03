package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/bishtpramod19/ecommerce-platform/services/inventory-service/internal/ports"
	"github.com/redis/go-redis/v9"
)

const (
	// lockPrefix namespaces all lock keys
	// Prevents conflicts with other Redis data
	lockPrefix = "inventory:lock:"
)

// redisLock implements ports.DistributedLock using Redis SET NX.
type redisLock struct {
	client *redis.Client
}

// NewRedisLock creates a new Redis-based distributed lock.
func NewRedisLock(client *redis.Client) ports.DistributedLock {
	return &redisLock{client: client}
}

// Acquire tries to acquire a distributed lock using Redis SET NX.
//
// SET NX = "SET if Not eXists"
// This is ATOMIC — only ONE caller can set the key at a time.
// If key already exists → returns false (someone else has the lock)
// If key doesn't exist → sets it with TTL → returns true (we got the lock)
//
// The TTL (time-to-live) is CRITICAL:
// If service crashes after acquiring lock but before releasing:
//
//	Without TTL: lock held forever → deadlock!
//	With TTL:    Redis auto-deletes key after TTL → lock released automatically ✅
func (r *redisLock) Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	fullKey := lockPrefix + key

	// SET NX EX → atomic "set if not exists with expiry"
	// Returns true if key was set (we got the lock)
	// Returns false if key already existed (someone else has it)
	result, err := r.client.SetNX(ctx, fullKey, "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("error acquiring lock for key %s: %w", key, err)
	}

	return result, nil
}

// Release removes the lock key from Redis.
func (r *redisLock) Release(ctx context.Context, key string) error {
	fullKey := lockPrefix + key

	if err := r.client.Del(ctx, fullKey).Err(); err != nil {
		return fmt.Errorf("error releasing lock for key %s: %w", key, err)
	}

	return nil
}

// IsLocked checks if a lock key currently exists in Redis.
func (r *redisLock) IsLocked(ctx context.Context, key string) (bool, error) {
	fullKey := lockPrefix + key

	exists, err := r.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("error checking lock for key %s: %w", key, err)
	}

	return exists > 0, nil
}
