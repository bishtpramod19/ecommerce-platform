package ports

import (
	"context"
	"time"
)

// DistributedLock is the PORT for distributed locking.
// Defines WHAT locking operations are needed.
// Knows nothing about Redis or any specific locking implementation.
type DistributedLock interface {
	// Acquire tries to acquire a lock for the given key.
	// Returns true if lock acquired, false if already locked by someone else.
	// TTL defines how long the lock is held before auto-expiry.
	Acquire(ctx context.Context, key string, ttl time.Duration) (bool, error)

	// Release releases a previously acquired lock.
	// Only the lock owner should release it.
	Release(ctx context.Context, key string) error

	// IsLocked checks if a key is currently locked.
	IsLocked(ctx context.Context, key string) (bool, error)
}
