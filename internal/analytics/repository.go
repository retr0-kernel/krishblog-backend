package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"krishblog/internal/database"
)

const (
	eventTimeExpr  = "COALESCE(recorded_at, timestamp)"
	viewEventsExpr = "event_type IN ('page_view', 'post_view')"
)

// Repository handles all analytics database operations.
// Writes are called by the async processor; reads serve the admin API.
type Repository struct {
	db *database.Postgres
}

func NewRepository(db *database.Postgres) *Repository {
	return &Repository{db: db}
}

func safeLimit(n int) int {
	if n <= 0 {
		return 10
	}
	if n > 100 {
		return 100
	}
	return n
}

// ApplySchemaPatches adds analytics-specific columns that Ent doesn't manage.
// Idempotent — safe to call on every boot.
func (r *Repository) ApplySchemaPatches(ctx context.Context) error {
	patches := []string{
		`ALTER TABLE analytics_events
		  ADD COLUMN IF NOT EXISTS ip_hash     TEXT         NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS browser     TEXT         NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS os          TEXT         NOT NULL DEFAULT '',
		  ADD COLUMN IF NOT EXISTS recorded_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()`,

		`CREATE INDEX IF NOT EXISTS idx_ae_ip_hash    ON analytics_events(ip_hash)`,
		`CREATE INDEX IF NOT EXISTS idx_ae_browser    ON analytics_events(browser)`,
		`CREATE INDEX IF NOT EXISTS idx_ae_post_time  ON analytics_events(post_id, recorded_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_ae_session_ts ON analytics_events(session_id, recorded_at DESC)`,
	}

	for _, stmt := range patches {
		if _, err := r.db.DB.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("analytics schema patch: %w", err)
		}
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

	recordedAt := e.RecordedAt
	if recordedAt.IsZero() {
		recordedAt = time.Now().UTC()
	}

	device := e.Device
	if device == "" {
		device = "unknown"
	}

	_, err := r.db.DB.ExecContext(ctx, `
		INSERT INTO analytics_events (
			id, event_type, session_id, post_id, path, referrer,
			scroll_pct, duration_ms, ip_hash, country, device, browser, os,
			metadata, timestamp, recorded_at
		) VALUES (
			gen_random_uuid(),
			$1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7,
			$8, $9, $10, $11, $12, $13, $14, $14
		)`,
		e.Type, e.SessionID, e.PostID,
		e.Path, e.Referrer,
		e.ScrollPct, e.DurationMs,
		e.IPHash, e.Country, device, e.Browser, e.OS,
		string(meta), recordedAt,
	)
	return err
}

// Overview returns aggregated site-wide analytics for the last N days.
func (r *Repository) Overview(ctx context.Context, days int) (*OverviewResponse, error) {
	since := time.Now().UTC().AddDate(0, 0, -days)
	resp := &OverviewResponse{
		Period:           fmt.Sprintf("last_%d_days", days),
		TopPosts:         []PostStat{},
		TopReferrers:     []ReferrerStat{},
		DeviceBreakdown:  []DeviceStat{},
		CountryBreakdown: []CountryStat{},
		DailyViews:       []DailyStat{},
	}

	row := r.db.DB.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT
			COUNT(CASE WHEN %s THEN 1 END),
			COUNT(DISTINCT CASE WHEN %s THEN NULLIF(ip_hash, '') END),
			COALESCE(AVG(CASE WHEN event_type = 'scroll_depth' THEN scroll_pct::float END), 0),
			COALESCE(AVG(CASE WHEN event_type = 'session_end'  THEN duration_ms::float END) / 1000.0, 0)
		FROM analytics_events
		WHERE %s >= $1
	`, viewEventsExpr, viewEventsExpr, eventTimeExpr), since)
	if err := row.Scan(
		&resp.TotalPageViews,
		&resp.UniqueVisitors,
		&resp.AvgScrollPct,
		&resp.AvgReadTimeSec,
	); err != nil {
		return nil, fmt.Errorf("overview aggregate: %w", err)
	}

	var err error
	if resp.TopPosts, err = r.topPosts(ctx, since, 10); err != nil {
		return nil, err
	}
	if resp.TopReferrers, err = r.topReferrers(ctx, since, 10); err != nil {
		return nil, err
	}
	if resp.DeviceBreakdown, err = r.deviceBreakdown(ctx, since, ""); err != nil {
		return nil, err
	}
	if resp.CountryBreakdown, err = r.countryBreakdown(ctx, since, 10); err != nil {
		return nil, err
	}
	if resp.DailyViews, err = r.dailyViews(ctx, since, days, ""); err != nil {
		return nil, err
	}
	return resp, nil
}

// PostStats returns per-post analytics for the last N days.
func (r *Repository) PostStats(ctx context.Context, postID string, days int) (*PostDetailResponse, error) {
	since := time.Now().UTC().AddDate(0, 0, -days)
	resp := &PostDetailResponse{
		PostID:          postID,
		DailyViews:      []DailyStat{},
		DeviceBreakdown: []DeviceStat{},
	}

	row := r.db.DB.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT
			COUNT(CASE WHEN event_type = 'post_view'     THEN 1 END),
			COUNT(DISTINCT CASE WHEN event_type = 'post_view' THEN NULLIF(ip_hash, '') END),
			COALESCE(AVG(CASE WHEN event_type = 'scroll_depth' THEN scroll_pct::float END), 0),
			COUNT(CASE WHEN event_type = 'read_complete' THEN 1 END)
		FROM analytics_events
		WHERE post_id::text = $1 AND %s >= $2
	`, eventTimeExpr), postID, since)
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
	if resp.DailyViews, err = r.dailyViews(ctx, since, days, postID); err != nil {
		return nil, err
	}
	if resp.DeviceBreakdown, err = r.deviceBreakdown(ctx, since, postID); err != nil {
		return nil, err
	}
	return resp, nil
}

// ── sub-queries ───────────────────────────────────────────────────────────────

func (r *Repository) topPosts(ctx context.Context, since time.Time, limit int) ([]PostStat, error) {
	rows, err := r.db.DB.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			ae.post_id::text,
			COALESCE(p.title, 'Unknown'),
			COALESCE(p.slug,  ''),
			COUNT(*) AS views,
			COUNT(DISTINCT NULLIF(ae.ip_hash, '')) AS unique_visitors,
			COALESCE(AVG(CASE WHEN ae.event_type = 'scroll_depth' THEN ae.scroll_pct::float END), 0) AS avg_scroll
		FROM analytics_events ae
		LEFT JOIN posts p ON p.id = ae.post_id
		WHERE ae.event_type = 'post_view'
		  AND ae.post_id IS NOT NULL
		  AND %s >= $1
		GROUP BY ae.post_id, p.title, p.slug
		ORDER BY views DESC
		LIMIT %d
	`, eventTimeExpr, safeLimit(limit)), since)
	if err != nil {
		return nil, fmt.Errorf("top posts: %w", err)
	}
	defer rows.Close()

	out := []PostStat{}
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
	rows, err := r.db.DB.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			CASE WHEN referrer = '' OR referrer IS NULL THEN 'direct' ELSE referrer END AS ref,
			COUNT(*) AS cnt
		FROM analytics_events
		WHERE %s AND %s >= $1
		GROUP BY 1
		ORDER BY cnt DESC
		LIMIT %d
	`, viewEventsExpr, eventTimeExpr, safeLimit(limit)), since)
	if err != nil {
		return nil, fmt.Errorf("top referrers: %w", err)
	}
	defer rows.Close()

	out := []ReferrerStat{}
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
	var (
		rows interface {
			Next() bool
			Scan(...interface{}) error
			Close() error
			Err() error
		}
		err error
	)

	if postID != "" {
		rows, err = r.db.DB.QueryContext(ctx, fmt.Sprintf(`
			SELECT device, COUNT(*) AS cnt
			FROM analytics_events
			WHERE event_type = 'post_view'
			  AND post_id::text = $1
			  AND %s >= $2
			GROUP BY device ORDER BY cnt DESC
		`, eventTimeExpr), postID, since)
	} else {
		rows, err = r.db.DB.QueryContext(ctx, fmt.Sprintf(`
			SELECT device, COUNT(*) AS cnt
			FROM analytics_events
			WHERE %s AND %s >= $1
			GROUP BY device ORDER BY cnt DESC
		`, viewEventsExpr, eventTimeExpr), since)
	}
	if err != nil {
		return nil, fmt.Errorf("device breakdown: %w", err)
	}
	defer rows.Close()

	out := []DeviceStat{}
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
	rows, err := r.db.DB.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			CASE WHEN country = '' OR country IS NULL THEN 'Unknown' ELSE UPPER(country) END AS co,
			COUNT(*) AS cnt
		FROM analytics_events
		WHERE %s AND %s >= $1
		GROUP BY 1 ORDER BY cnt DESC LIMIT %d
	`, viewEventsExpr, eventTimeExpr, safeLimit(limit)), since)
	if err != nil {
		return nil, fmt.Errorf("country breakdown: %w", err)
	}
	defer rows.Close()

	out := []CountryStat{}
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
	limit := safeLimit(days)
	var query string
	var args []interface{}

	if postID != "" {
		query = fmt.Sprintf(`
			SELECT
				TO_CHAR(DATE_TRUNC('day', %s AT TIME ZONE 'UTC'), 'YYYY-MM-DD') AS day,
				COUNT(CASE WHEN event_type = 'post_view' THEN 1 END),
				COUNT(DISTINCT CASE WHEN event_type = 'post_view' THEN NULLIF(ip_hash, '') END)
			FROM analytics_events
			WHERE post_id::text = $1 AND %s >= $2
			GROUP BY 1 ORDER BY day ASC LIMIT %d
		`, eventTimeExpr, eventTimeExpr, limit)
		args = []interface{}{postID, since}
	} else {
		query = fmt.Sprintf(`
			SELECT
				TO_CHAR(DATE_TRUNC('day', %s AT TIME ZONE 'UTC'), 'YYYY-MM-DD') AS day,
				COUNT(CASE WHEN %s THEN 1 END),
				COUNT(DISTINCT CASE WHEN %s THEN NULLIF(ip_hash, '') END)
			FROM analytics_events
			WHERE %s >= $1
			GROUP BY 1 ORDER BY day ASC LIMIT %d
		`, eventTimeExpr, viewEventsExpr, viewEventsExpr, eventTimeExpr, limit)
		args = []interface{}{since}
	}

	rows, err := r.db.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("daily views: %w", err)
	}
	defer rows.Close()

	out := []DailyStat{}
	for rows.Next() {
		var s DailyStat
		if err := rows.Scan(&s.Date, &s.PageViews, &s.UniqueVisitors); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
