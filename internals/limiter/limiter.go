package limiter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// config defines the token bucket parameter for a rate limit tier
// capacity is the maximum number of tokens a bucket can hold
// Rate is the number of tokens added per second
// KeyPrefix namespaces Redis keys for this limit tier
type Config struct {
	Capacity  float64
	Rate      float64
	KeyPrefix string
}

// Result is returned by allow and contains enough information for the caller to to set rate limit response headers
type Result struct {
	Allowed    bool
	Remaining  float64       //tokens left after a request
	RetryAfter time.Duration //how long untill a token is available that is if denied
	Limit      float64       //the bucket capacity
}

// Limiter is a distributed token bucket limiter backed by Redis.
// Falls back to an in-process map limiter if Redis is nil.
type Limiter struct {
	redis  *redis.Client
	script *redis.Script
	local  *localLimiter //fallback if redis is unavailable
}

// luaScript is the atomic token bucket operation.
// Arguments: KEYS[1] = hash key, ARGV[1] = capacity, ARGV[2] = rate/s,
//
//	ARGV[3] = now (unix nanoseconds as string), ARGV[4] = TTL seconds.
//
// Returns: table { allowed (0|1), tokens_remaining, retry_after_ms }
var luaScript = redis.NewScript(`
local key      = KEYS[1]
local capacity = tonumber(ARGV[1])
local rate     = tonumber(ARGV[2])
local now      = tonumber(ARGV[3])
local ttl      = tonumber(ARGV[4])
 
local data = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens     = tonumber(data[1]) or capacity
local last_refill = tonumber(data[2]) or now
 
local elapsed = math.max(0, (now - last_refill) / 1e9)
tokens = math.min(capacity, tokens + elapsed * rate)
 
local allowed = 0
local retry_ms = 0
 
if tokens >= 1.0 then
    tokens = tokens - 1.0
    allowed = 1
else
    retry_ms = math.ceil((1.0 - tokens) / rate * 1000)
end
 
redis.call('HSET', key, 'tokens', tokens, 'last_refill', now)
redis.call('EXPIRE', key, ttl)
 
return {allowed, tokens * 1000, retry_ms}
`)

// NewLimiter creates a Limiter. redisClient may be nil — in that case the
// limiter operates in-process only (not distributed, but still functional).
func NewLimiter(redisClient *redis.Client) *Limiter {
	return &Limiter{
		redis:  redisClient,
		script: luaScript,
		local:  newLocalLimiter(),
	}
}

// allow checks wetheer the given key is within its rate limit under config
func (l *Limiter) Allow(ctx context.Context, key string, cfg Config) (Result, error) {

	if l.redis != nil {
		return l.allowRedis(ctx, key, cfg)
	}
	return l.local.allow(key, cfg), nil
}

func (l *Limiter) allowRedis(ctx context.Context, key string, cfg Config) (Result, error) {
	now := time.Now().UnixNano()
	// TTL: keep the key alive for at least capacity / rate seconds, the
	// time it takes a completely empty bucket to refill. Add a buffer.
	ttl := int64(cfg.Capacity/cfg.Rate) + 60

	vals, err := l.script.Run(ctx, l.redis,
		[]string{cfg.KeyPrefix + key},
		cfg.Capacity,
		cfg.Rate,
		now,
		ttl,
	).Int64Slice()

	if err != nil {
		// Redis error fail open, allow the request rather than denying
		// all traffic during a Redis outage. Log and fall back to local.
		slog.Warn("ratelimit: redis script failed, falling back to local",
			"err", err, "key", key)
		result := l.local.allow(key, cfg)
		return result, nil
	}

	allowed := vals[0] == 1
	remaining := float64(vals[1]) / 1000.0
	retryMs := vals[2]

	var retryAfter time.Duration
	if !allowed {
		retryAfter = time.Duration(retryMs) * time.Millisecond
	}

	return Result{
		Allowed:    allowed,
		Remaining:  remaining,
		RetryAfter: retryAfter,
		Limit:      cfg.Capacity,
	}, nil
}

// THIS THE IN-PROCESS FALLBAK LIMITER
// localLimiter is a non-distributed token bucket for when Redis is down.
// It uses a sync.Map of buckets, each protected by its own mutex.
type localLimiter struct {
	buckets sync.Map // key → *localBucket
}

type localBucket struct {
	mu         sync.Mutex
	tokens     float64
	lastRefill time.Time
}

func newLocalLimiter() *localLimiter { return &localLimiter{} }

func (l *localLimiter) allow(key string, cfg Config) Result {
	v, _ := l.buckets.LoadOrStore(key, &localBucket{
		tokens:     cfg.Capacity,
		lastRefill: time.Now(),
	})
	b := v.(*localBucket)

	b.mu.Lock()
	defer b.mu.Unlock()

	elapsed := time.Since(b.lastRefill).Seconds()
	b.tokens = min(cfg.Capacity, b.tokens+elapsed*cfg.Rate)
	b.lastRefill = time.Now()

	if b.tokens >= 1.0 {
		b.tokens--
		return Result{Allowed: true, Remaining: b.tokens, Limit: cfg.Capacity}
	}

	retryAfter := time.Duration((1.0-b.tokens)/cfg.Rate*1000) * time.Millisecond
	return Result{Allowed: false, Remaining: 0, RetryAfter: retryAfter, Limit: cfg.Capacity}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

var (
	// ??Redirectaanon limits unauthenticated requests by IP, 60 burst, 10/s sustained
	RedirectAnon = Config{Capacity: 60, Rate: 10, KeyPrefix: "rl:rd:a:"}

	//RedirectAuthed limits authenticated redirect requests byuser id, auth users have more trust so set higher
	RedirectAuthed = Config{Capacity: 300, Rate: 50, KeyPrefix: "rl:rd:u:"}

	//shortenanon limits anonymous url creation by IP 10 burst 1/s prevents bulk anonnymous spam
	ShortenAnon = Config{Capacity: 10, Rate: 1, KeyPrefix: "rl:sh:a:"}

	// ShortenAuthed limits authenticated URL creation by user ID.
	ShortenAuthed = Config{Capacity: 100, Rate: 5, KeyPrefix: "rl:sh:u:"}

	//Authstrict is for login and register, the tightest limit, 5 burst, 0.1/s (one attempt per 10 seconds sustained).
	AuthStrict = Config{Capacity: 5, Rate: 0.1, KeyPrefix: "rl:auth:"}
)

//KEY EXTRACTION HELPERS

// Ipkey returns a clients real ip for use as a rate limit key
func IPKey(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP — the original client before any proxies.
		for i, c := range xff {
			if c == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// RemoteAddr includes the port, strip it.
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}

// UserKey returns a stable string key for a user ID.
func UserKey(userID string) string {
	return "user:" + userID
}

//RESPONSE HELPERS

// SetHeaders writes the standard rate limit headers to w.
// Clients that respect these headers can self-throttle gracefully.
func SetHeaders(w http.ResponseWriter, r Result) {
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", r.Limit))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%.0f", r.Remaining))
	if !r.Allowed {
		w.Header().Set("Retry-After", fmt.Sprintf("%.0f", r.RetryAfter.Seconds()))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d",
			time.Now().Add(r.RetryAfter).Unix()))
	}
}

// Deny writes a 429 Too Many Requests response with a JSON body and headers.
func Deny(w http.ResponseWriter, r Result) {
	SetHeaders(w, r)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	fmt.Fprintf(w, `{"error":"rate limit exceeded","retry_after_seconds":%.1f}`,
		r.RetryAfter.Seconds())
}

// we return this when redis is not configured
var ErrRedisUnavailable = errors.New("ratelimit: redis not available")
