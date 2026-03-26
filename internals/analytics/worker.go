package analytics

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"sync/atomic"
	"time"
	"url-shortener/internals/models"
	"url-shortener/internals/repository"
)

const (
	maxReferrerLen = 512
	maxUALen       = 512
)

type Worker struct {
	repo    *repository.ClickRepository
	events  chan models.Click
	dropped atomic.Int64
}

func NewWorker(repo *repository.ClickRepository, bufferSize int) *Worker {
	return &Worker{
		repo:   repo,
		events: make(chan models.Click, bufferSize),
	}
}

func (w *Worker) Track(r *http.Request, slug string) {
	click := models.Click{
		Slug:      slug,
		ClickedAt: time.Now(),
		IPHash:    HashIP(RealIP(r)),
		Referrer:  Truncate(r.Referer(), maxReferrerLen),
		UserAgent: Truncate(r.UserAgent(), maxUALen),
	}

	// Non-blocking send. If the channel is full, drop and count.
	select {
	case w.events <- click:
	default:
		w.dropped.Add(1)
	}
}

func (w *Worker) insertBatch(ctx context.Context, clicks []models.Click) error {
	if len(clicks) == 0 {
		return nil
	}

	if err := w.repo.InsertMany(ctx, clicks); err != nil {
		return fmt.Errorf("analytics.insertBatch: %w", err)
	}

	return nil
}

func (w *Worker) Run(ctx context.Context) {
	batch := make([]models.Click, 0, 100)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// flush remaining events before exiting
			if len(batch) > 0 {
				_ = w.insertBatch(ctx, batch)
			}
			return
		case click := <-w.events:
			batch = append(batch, click)
			if len(batch) >= 100 {
				_ = w.insertBatch(ctx, batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				_ = w.insertBatch(ctx, batch)
				batch = batch[:0]
			}
		}
	}
}

// hashIP returns the SHA-256 hex digest of an IP address string which makes it anonymous
func HashIP(ip string) string {
	sum := sha256.Sum256([]byte(ip))
	// return fmt.Sprintf("%x", sum)
	return hex.EncodeToString(sum[:])
}

// / realIP extracts the client's real IP address from the request
func RealIP(r *http.Request) string {
	// Try X-Forwarded-For first (for proxies)
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	if r.RemoteAddr != "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			return host
		}
		return r.RemoteAddr
	}
	return ""
}

func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
