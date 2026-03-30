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

var ErrCacheMiss = errors.New("cache: miss")

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(addr, password string, db int) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,

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
func (c *RedisCache) Get(ctx context.Context, slug string) (string, error) {
	val, err := c.client.Get(ctx, keyPrefix+slug).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrCacheMiss
	}
	//treat redis cache erros as cache missed rather than failiures
	///so basically if its down we just hadnle gracefully instead of sending 500 to user

	if err != nil {

		return "", ErrCacheMiss
	}
	return val, nil
}

// /set stores the slug and long url plus ttl is capped at time reamaining so no link is served after alink expires
func (c *RedisCache) Set(ctx context.Context, slug, longURL string, expiresAt *time.Time) error {
	ttl := DefaultTTL
	if expiresAt != nil {
		remaining := time.Until(*expiresAt)
		if remaining <= 0 {
			// Already expired  don't cache it at all.
			return nil
		}
		if remaining < ttl {
			ttl = remaining
		}
	}

	if err := c.client.Set(ctx, keyPrefix+slug, longURL, ttl).Err(); err != nil {
		// Log but don't fail
		return fmt.Errorf("cache.Set: %w", err)
	}
	return nil
}

// /// Delete removes a slug from cace on soft delete
func (c *RedisCache) Delete(ctx context.Context, slug string) error {
	if err := c.client.Del(ctx, keyPrefix+slug).Err(); err != nil {
		return fmt.Errorf("cache.Delete: %w", err)
	}
	return nil
}

func (c *RedisCache) Client() *redis.Client {
	return c.client
}

// Close shuts down the Redis connection pool gracefully.
func (c *RedisCache) Close() error {
	return c.client.Close()
}
