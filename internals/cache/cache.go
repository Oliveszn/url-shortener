package cache

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	DefaultTTL = 24 * time.Hour

	keyPrefix = "url:"
)

// ErrCacheMiss is returned when a slug is not in the cache.
// The caller should fall through to Postgres and then populate the cache.
var ErrCacheMiss = errors.New("cache: miss")

// RedisCache wraps a go-redis client with typed URL caching operations.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache connects to Redis and returns a RedisCache.
// The caller is responsible for calling Close() when done.
func NewRedisCache(addr, password string, db int) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,

		// Connection pool settings.
		// PoolSize should roughly match the number of concurrent redirect
		// handlers. 10 is safe for a small deployment.
		PoolSize:     10,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("cache.NewRedisCache: ping failed: %w", err)
	}

	return &RedisCache{client: client}, nil
}

// Get retrieves the long URL for a slug.
// Returns ErrCacheMiss if the key does not exist in Redis.
func (c *RedisCache) Get(ctx context.Context, slug string) (string, error) {
	val, err := c.client.Get(ctx, keyPrefix+slug).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrCacheMiss
	}
	if err != nil {
		// Treat Redis errors as cache misses rather than hard failures.
		// Rationale: Redis is a performance optimisation. If it's down, the
		// system should degrade gracefully by falling through to Postgres,
		// not by returning 500 to the user.
		// In production, emit a metric here (e.g. Prometheus counter) so you
		// know when Redis is unhealthy.
		return "", ErrCacheMiss
	}
	return val, nil
}

// Set stores a slug → long URL mapping with an appropriate TTL.
//
// expiresAt is the link's absolute expiry time (nil = no expiry).
// The cache TTL is capped at the time remaining until expiry so we never
// serve a cached redirect past the link's expiry.
func (c *RedisCache) Set(ctx context.Context, slug, longURL string, expiresAt *time.Time) error {
	ttl := DefaultTTL
	if expiresAt != nil {
		remaining := time.Until(*expiresAt)
		if remaining <= 0 {
			// Already expired — don't cache it at all.
			return nil
		}
		if remaining < ttl {
			ttl = remaining
		}
	}

	if err := c.client.Set(ctx, keyPrefix+slug, longURL, ttl).Err(); err != nil {
		// Log but don't fail — cache write failure is non-fatal.
		return fmt.Errorf("cache.Set: %w", err)
	}
	return nil
}

// Delete removes a slug from the cache immediately.
// Called when a link is deactivated (soft-deleted). We must invalidate
// synchronously here — a stale cache entry would keep the link redirecting
// until the TTL expires, which is unacceptable for a deleted link.
func (c *RedisCache) Delete(ctx context.Context, slug string) error {
	if err := c.client.Del(ctx, keyPrefix+slug).Err(); err != nil {
		return fmt.Errorf("cache.Delete: %w", err)
	}
	return nil
}

// Close shuts down the Redis connection pool gracefully.
func (c *RedisCache) Close() error {
	return c.client.Close()
}
