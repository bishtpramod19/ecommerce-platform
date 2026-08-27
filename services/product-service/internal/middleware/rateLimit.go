package middleware

import (
	"net/http"
	"sync"
	"time"
)

// tokenBucket represents a rate limiter for a single client (IP address).
// Uses the token bucket algorithm:
//   - Bucket holds maximum N tokens
//   - Each request consumes 1 token
//   - Tokens refill at a fixed rate
//   - If bucket empty → request rejected (429)
type tokenBucket struct {
	tokens     float64   // current tokens available
	maxTokens  float64   // maximum tokens (burst limit)
	refillRate float64   // tokens added per second
	lastRefill time.Time // when we last added tokens
	mu         sync.Mutex
}

// allow checks if a request is allowed and consumes a token.
func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()

	// Refill tokens based on time elapsed since last refill
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate
	tb.lastRefill = now

	// Cap tokens at maximum (can't accumulate more than bucket size)
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}

	// Check if request is allowed
	if tb.tokens < 1 {
		return false // bucket empty → rate limited
	}

	// Consume one token
	tb.tokens--
	return true
}

// RateLimiter manages token buckets per client IP.
type RateLimiter struct {
	buckets    map[string]*tokenBucket
	mu         sync.RWMutex
	maxTokens  float64
	refillRate float64
}

// NewRateLimiter creates a new rate limiter.
// maxRequests: max burst (e.g., 100 requests at once)
// refillRate:  requests per second after burst (e.g., 10/sec)
func NewRateLimiter(maxRequests float64, refillRate float64) *RateLimiter {
	rl := &RateLimiter{
		buckets:    make(map[string]*tokenBucket),
		maxTokens:  maxRequests,
		refillRate: refillRate,
	}

	// Cleanup goroutine: remove stale buckets every 5 minutes
	// Prevents memory leak (new IP every request would fill memory)
	go rl.cleanup()

	return rl
}

// getBucket returns the token bucket for a given IP, creating one if needed.
func (rl *RateLimiter) getBucket(ip string) *tokenBucket {
	// Try read lock first (fast path for existing IPs)
	rl.mu.RLock()
	bucket, exists := rl.buckets[ip]
	rl.mu.RUnlock()

	if exists {
		return bucket
	}

	// Write lock to create new bucket
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check (another goroutine might have created it)
	if bucket, exists = rl.buckets[ip]; exists {
		return bucket
	}

	bucket = &tokenBucket{
		tokens:     rl.maxTokens, // start with full bucket
		maxTokens:  rl.maxTokens,
		refillRate: rl.refillRate,
		lastRefill: time.Now(),
	}
	rl.buckets[ip] = bucket
	return bucket
}

// cleanup removes stale buckets periodically to prevent memory leaks.
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		for ip, bucket := range rl.buckets {
			bucket.mu.Lock()
			// Remove buckets inactive for more than 10 minutes
			if time.Since(bucket.lastRefill) > 10*time.Minute {
				delete(rl.buckets, ip)
			}
			bucket.mu.Unlock()
		}
		rl.mu.Unlock()
	}
}

// RateLimitMiddleware limits requests per IP address.
func RateLimitMiddleware(limiter *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			ip := r.RemoteAddr

			// Get bucket for this IP and check if allowed
			bucket := limiter.getBucket(ip)
			if !bucket.allow() {
				w.Header().Set("Retry-After", "1")
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded, please slow down")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
