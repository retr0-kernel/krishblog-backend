package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"krishblog/internal/database"
)

// Repository handles all analytics database operations.
// Writes are called by the async processor; reads serve the admin API.
type Repository struct {
	db *database.Postgres
}

func NewRepository(db *database.Postgres) *Repository {
	return &Repository{db: db}
}

// ApplySchemaPatches adds analytics-specific columns that Ent doesn't manage.
// Idempotent — safe to call on every boot.
func (r *Repository) ApplySchemaPatches(ctx context.Context) error {
	_, err := r.db.DB.ExecContext(ctx, `
		ALTER TABLE analytics_events
		  ADD COLUMN IF NOT EXISTS ip_hash    TEXT         NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS browser    TEXT         NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS os         TEXT         NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

		CREATE INDEX IF NOT EXISTS idx_ae_ip_hash    ON analytics_events(ip_hash);
		CREATE INDEX IF NOT EXISTS idx_ae_browser    ON analytics_events(browser);
		CREATE INDEX IF NOT EXISTS idx_ae_post_time  ON analytics_events(post_id, recorded_at DESC);
		CREATE INDEX IF NOT EXISTS idx_ae_session_ts ON analytics_events(session_id, recorded_at DESC);
	`)
	if err != nil {
		return fmt.Errorf("analytics schema patch: %w", err)
	}
	return nil
}

// InsertEvent persists one enriched event to Postgres.
// Called by the async processor, never by the HTTP handler directly.
func (r *Repository) InsertEvent(ctx context.Context, e enrichedEvent) error {
	meta, _ := json.Marshal(e.Metadata)
	if meta == nil {
		meta = []byte("{}")
	}

	_, err := r.db.DB.ExecContext(ctx, `
		INSERT INTO analytics_events (
			event_type, session_id,
			post_id, section_id,
			path, referrer,
			scroll_pct, duration_ms,
			ip_hash, country, device, browser, os,
			metadata, recorded_at
		) VALUES (
			$1,  $2,
			NULLIF($3,  '')::uuid, NULLIF($4,  '')::uuid,
			$5,  $6,
			$7,  $8,
			$9,  $10, $11, $12, $13,
			$14, $15
		)`,
		e.Type, e.SessionID,
		e.PostID, e.SectionID,
		e.Path, e.Referrer,
		e.ScrollPct, e.DurationMs,
		e.IPHash, e.Country, e.Device, e.Browser, e.OS,
		string(meta), e.RecordedAt,
	)
	return err
}

// ── Admin read queries ────────────────────────────────────────────────────────

// Overview returns aggregated site-wide analytics for the last N days.
// Overview returns aggregated site-wide analytics for the last N days.
func (r *Repository) Overview(ctx context.Context, days int) (*OverviewResponse, error) {
	since := time.Now().UTC().AddDate(0, 0, -days)
	resp := &OverviewResponse{Period: fmt.Sprintf("last_%d_days", days)}

	// Use CASE WHEN instead of FILTER for broader compatibility
	row := r.db.DB.QueryRowContext(ctx, `
		SELECT
			COUNT(CASE WHEN event_type = 'page_view' THEN 1 END),
			COUNT(DISTINCT CASE WHEN event_type = 'page_view' THEN ip_hash END),
			COALESCE(AVG(CASE WHEN event_type = 'scroll_depth' THEN scroll_pct::float END), 0),
			COALESCE(AVG(CASE WHEN event_type = 'session_end'  THEN duration_ms::float END) / 1000.0, 0)
		FROM analytics_events
		WHERE recorded_at >= $1
	`, since)
	if err := row.Scan(
		&resp.TotalPageViews,
		&resp.UniqueVisitors,
		&resp.AvgScrollPct,
		&resp.AvgReadTimeSec,
	); err != nil {
		return nil, fmt.Errorf("overview aggregate: %w", err)
	}

	var err error
	resp.TopPosts, err = r.topPosts(ctx, since, 10)
	if err != nil {
		return nil, err
	}
	resp.TopReferrers, err = r.topReferrers(ctx, since, 10)
	if err != nil {
		return nil, err
	}
	resp.DeviceBreakdown, err = r.deviceBreakdown(ctx, since, "")
	if err != nil {
		return nil, err
	}
	resp.CountryBreakdown, err = r.countryBreakdown(ctx, since, 10)
	if err != nil {
		return nil, err
	}
	resp.DailyViews, err = r.dailyViews(ctx, since, days, "")
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// PostStats returns per-post analytics for the last N days.
func (r *Repository) PostStats(ctx context.Context, postID string, days int) (*PostDetailResponse, error) {
	since := time.Now().UTC().AddDate(0, 0, -days)
	resp := &PostDetailResponse{PostID: postID}

	row := r.db.DB.QueryRowContext(ctx, `
		SELECT
			COUNT(CASE WHEN event_type = 'post_view'     THEN 1 END),
			COUNT(DISTINCT CASE WHEN event_type = 'post_view' THEN ip_hash END),
			COALESCE(AVG(CASE WHEN event_type = 'scroll_depth' THEN scroll_pct::float END), 0),
			COUNT(CASE WHEN event_type = 'read_complete' THEN 1 END)
		FROM analytics_events
		WHERE post_id::text = $1 AND recorded_at >= $2
	`, postID, since)
	if err := row.Scan(
		&resp.TotalViews,
		&resp.UniqueVisitors,
		&resp.AvgScrollPct,
		&resp.ReadCompletions,
	); err != nil {
		return nil, fmt.Errorf("post stats: %w", err)
	}

	if resp.TotalViews > 0 {
		resp.CompletionRate = float64(resp.ReadCompletions) / float64(resp.TotalViews) * 100
	}

	_ = r.db.DB.QueryRowContext(ctx, `SELECT title FROM posts WHERE id::text = $1`, postID).
		Scan(&resp.PostTitle)

	var err error
	resp.DailyViews, err = r.dailyViews(ctx, since, days, postID)
	if err != nil {
		return nil, err
	}
	resp.DeviceBreakdown, err = r.deviceBreakdown(ctx, since, postID)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ── sub-queries ───────────────────────────────────────────────────────────────

func (r *Repository) topPosts(ctx context.Context, since time.Time, limit int) ([]PostStat, error) {
	rows, err := r.db.DB.QueryContext(ctx, `
		SELECT
			ae.post_id::text,
			COALESCE(p.title, 'Unknown'),
			COALESCE(p.slug,  ''),
			COUNT(*)                                                          AS views,
			COUNT(DISTINCT ae.ip_hash)                                        AS unique_visitors,
			COALESCE(AVG(CASE WHEN ae.event_type = 'scroll_depth' THEN ae.scroll_pct::float END), 0) AS avg_scroll
		FROM analytics_events ae
		LEFT JOIN posts p ON p.id = ae.post_id
		WHERE ae.event_type = 'post_view'
		  AND ae.post_id IS NOT NULL
		  AND ae.recorded_at >= $1
		GROUP BY ae.post_id, p.title, p.slug
		ORDER BY views DESC
		LIMIT $2
	`, since, limit)
	if err != nil {
		return nil, fmt.Errorf("top posts: %w", err)
	}
	defer rows.Close()

	var out []PostStat
	for rows.Next() {
		var s PostStat
		if err := rows.Scan(&s.PostID, &s.PostTitle, &s.PostSlug, &s.Views, &s.UniqueVisitors, &s.AvgScrollPct); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) topReferrers(ctx context.Context, since time.Time, limit int) ([]ReferrerStat, error) {
	rows, err := r.db.DB.QueryContext(ctx, `
		SELECT
			CASE WHEN referrer = '' OR referrer IS NULL THEN 'direct' ELSE referrer END AS ref,
			COUNT(*) AS cnt
		FROM analytics_events
		WHERE event_type = 'page_view' AND recorded_at >= $1
		GROUP BY ref
		ORDER BY cnt DESC
		LIMIT $2
	`, since, limit)
	if err != nil {
		return nil, fmt.Errorf("top referrers: %w", err)
	}
	defer rows.Close()

	var out []ReferrerStat
	for rows.Next() {
		var s ReferrerStat
		if err := rows.Scan(&s.Referrer, &s.Count); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) deviceBreakdown(ctx context.Context, since time.Time, postID string) ([]DeviceStat, error) {
	var rows interface {
		Next() bool
		Scan(...interface{}) error
		Close() error
		Err() error
	}
	var err error

	if postID != "" {
		rows, err = r.db.DB.QueryContext(ctx, `
			SELECT device, COUNT(*) AS cnt
			FROM analytics_events
			WHERE event_type = 'post_view'
			  AND post_id::text = $1
			  AND recorded_at >= $2
			GROUP BY device ORDER BY cnt DESC
		`, postID, since)
	} else {
		rows, err = r.db.DB.QueryContext(ctx, `
			SELECT device, COUNT(*) AS cnt
			FROM analytics_events
			WHERE event_type = 'page_view' AND recorded_at >= $1
			GROUP BY device ORDER BY cnt DESC
		`, since)
	}
	if err != nil {
		return nil, fmt.Errorf("device breakdown: %w", err)
	}
	defer rows.Close()

	var out []DeviceStat
	var total int64
	for rows.Next() {
		var s DeviceStat
		if err := rows.Scan(&s.Device, &s.Count); err != nil {
			return nil, err
		}
		total += s.Count
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if total > 0 {
			out[i].Pct = float64(out[i].Count) / float64(total) * 100
		}
	}
	return out, nil
}

func (r *Repository) countryBreakdown(ctx context.Context, since time.Time, limit int) ([]CountryStat, error) {
	rows, err := r.db.DB.QueryContext(ctx, `
		SELECT
			CASE WHEN country = '' OR country IS NULL THEN 'Unknown' ELSE UPPER(country) END AS co,
			COUNT(*) AS cnt
		FROM analytics_events
		WHERE event_type = 'page_view' AND recorded_at >= $1
		GROUP BY co ORDER BY cnt DESC LIMIT $2
	`, since, limit)
	if err != nil {
		return nil, fmt.Errorf("country breakdown: %w", err)
	}
	defer rows.Close()

	var out []CountryStat
	var total int64
	for rows.Next() {
		var s CountryStat
		if err := rows.Scan(&s.Country, &s.Count); err != nil {
			return nil, err
		}
		total += s.Count
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		if total > 0 {
			out[i].Pct = float64(out[i].Count) / float64(total) * 100
		}
	}
	return out, nil
}

func (r *Repository) dailyViews(ctx context.Context, since time.Time, days int, postID string) ([]DailyStat, error) {
	var query string
	var args []interface{}

	if postID != "" {
		query = `
			SELECT
				TO_CHAR(DATE_TRUNC('day', recorded_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD') AS day,
				COUNT(CASE WHEN event_type = 'post_view' THEN 1 END),
				COUNT(DISTINCT CASE WHEN event_type = 'post_view' THEN ip_hash END)
			FROM analytics_events
			WHERE post_id::text = $1 AND recorded_at >= $2
			GROUP BY day ORDER BY day ASC LIMIT $3
		`
		args = []interface{}{postID, since, days}
	} else {
		query = `
			SELECT
				TO_CHAR(DATE_TRUNC('day', recorded_at AT TIME ZONE 'UTC'), 'YYYY-MM-DD') AS day,
				COUNT(CASE WHEN event_type = 'page_view' THEN 1 END),
				COUNT(DISTINCT CASE WHEN event_type = 'page_view' THEN ip_hash END)
			FROM analytics_events
			WHERE recorded_at >= $1
			GROUP BY day ORDER BY day ASC LIMIT $2
		`
		args = []interface{}{since, days}
	}

	rows, err := r.db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("daily views: %w", err)
	}
	defer rows.Close()

	var out []DailyStat
	for rows.Next() {
		var s DailyStat
		if err := rows.Scan(&s.Date, &s.PageViews, &s.UniqueVisitors); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
