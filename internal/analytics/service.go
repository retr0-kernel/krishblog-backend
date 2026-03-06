package analytics

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"krishblog/internal/database"
)

const (
	redisQueueKey   = "analytics:queue"
	redisQueueLimit = 10_000 // max events buffered before we start dropping
)

// Service handles event enrichment and admin read logic.
type Service struct {
	repo  *Repository
	redis *database.Redis
	log   *slog.Logger
}

func NewService(db *database.Postgres, redis *database.Redis, log *slog.Logger) *Service {
	return &Service{
		repo:  NewRepository(db),
		redis: redis,
		log:   log,
	}
}

// Repository returns the underlying repo (used by the processor).
func (s *Service) Repository() *Repository { return s.repo }

// Enqueue validates, enriches, and pushes an event to the Redis queue.
// Returns immediately — the processor flushes async.
func (s *Service) Enqueue(ctx context.Context, req EventRequest, rawIP, ua, country string) error {
	// Bot filtering
	if IsBot(ua) {
		return nil // silently drop
	}

	// Session validation
	if !IsValidSession(req.SessionID) {
		return nil // silently drop invalid sessions
	}

	// Parse device/browser/OS from User-Agent
	parsed := ParseUserAgent(ua)

	// Hash IP for privacy — we never store raw IPs
	ipHash := hashIP(rawIP)

	event := enrichedEvent{
		Type:       string(req.Type),
		SessionID:  req.SessionID,
		PostID:     req.PostID,
		SectionID:  req.SectionID,
		Path:       req.Path,
		Referrer:   req.Referrer,
		ScrollPct:  req.ScrollPct,
		DurationMs: req.DurationMs,
		Metadata:   req.Metadata,
		IPHash:     ipHash,
		Country:    country,
		Device:     string(parsed.Device),
		Browser:    parsed.Browser,
		OS:         parsed.OS,
		RecordedAt: time.Now().UTC(),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	// Check queue depth — drop if backpressure is too high
	length, _ := s.redis.Client.LLen(ctx, redisQueueKey).Result()
	if length >= redisQueueLimit {
		s.log.Warn("analytics queue full, dropping event",
			slog.Int64("queue_depth", length),
		)
		return nil
	}

	return s.redis.RPush(ctx, redisQueueKey, string(payload))
}

// Overview returns site-wide analytics for the last N days.
func (s *Service) Overview(ctx context.Context, days int) (*OverviewResponse, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	result, err := s.repo.Overview(ctx, days)
	if err != nil {
		// Log the actual error so it's visible in server logs
		s.log.Error("analytics overview failed", slog.String("error", err.Error()))
		return nil, err
	}
	return result, nil
}

// PostStats returns per-post analytics for the last N days.
func (s *Service) PostStats(ctx context.Context, postID string, days int) (*PostDetailResponse, error) {
	if days <= 0 || days > 365 {
		days = 30
	}
	return s.repo.PostStats(ctx, postID, days)
}

// hashIP returns a salted SHA-256 hex digest of the raw IP address.
// The salt prevents rainbow-table reversal of common IP ranges.
func hashIP(ip string) string {
	// Fixed salt — in production rotate this periodically to further protect privacy
	const salt = "analytics-ip-salt-v1"
	h := sha256.New()
	h.Write([]byte(salt + ip))
	return fmt.Sprintf("%x", h.Sum(nil))[:16] // truncate to 16 hex chars
}

// parseDays extracts ?days= query param, defaulting to 30.
func parseDays(raw string) int {
	if raw == "" {
		return 30
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 || n > 365 {
		return 30
	}
	return n
}
