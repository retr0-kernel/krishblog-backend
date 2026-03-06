package analytics

import "time"

// ── Event types ───────────────────────────────────────────────────────────────

type EventType string

const (
	EventPageView     EventType = "page_view"
	EventPostView     EventType = "post_view"
	EventScrollDepth  EventType = "scroll_depth"
	EventReadComplete EventType = "read_complete"
	EventClick        EventType = "click"
	EventSearch       EventType = "search"
	EventSessionStart EventType = "session_start"
	EventSessionEnd   EventType = "session_end"
)

// ── Inbound ───────────────────────────────────────────────────────────────────

// EventRequest is the payload POSTed by the frontend JS tracker.
type EventRequest struct {
	Type       EventType              `json:"type"        validate:"required,oneof=page_view post_view scroll_depth read_complete click search session_start session_end"`
	SessionID  string                 `json:"session_id"  validate:"required,min=16,max=128"`
	PostID     string                 `json:"post_id"     validate:"omitempty,uuid4"`
	SectionID  string                 `json:"section_id"  validate:"omitempty,uuid4"`
	Path       string                 `json:"path"        validate:"required,max=2048"`
	Referrer   string                 `json:"referrer"    validate:"omitempty,max=2048"`
	ScrollPct  *int                   `json:"scroll_pct"  validate:"omitempty,min=0,max=100"`
	DurationMs *int                   `json:"duration_ms" validate:"omitempty,min=0"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// enrichedEvent is what we enqueue in Redis and persist to Postgres.
type enrichedEvent struct {
	Type       string                 `json:"type"`
	SessionID  string                 `json:"session_id"`
	PostID     string                 `json:"post_id"`
	SectionID  string                 `json:"section_id"`
	Path       string                 `json:"path"`
	Referrer   string                 `json:"referrer"`
	ScrollPct  *int                   `json:"scroll_pct,omitempty"`
	DurationMs *int                   `json:"duration_ms,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	IPHash     string                 `json:"ip_hash"`
	Country    string                 `json:"country"`
	Device     string                 `json:"device"`
	Browser    string                 `json:"browser"`
	OS         string                 `json:"os"`
	RecordedAt time.Time              `json:"recorded_at"`
}

// ── Outbound ──────────────────────────────────────────────────────────────────

type OverviewResponse struct {
	Period           string         `json:"period"`
	TotalPageViews   int64          `json:"total_page_views"`
	UniqueVisitors   int64          `json:"unique_visitors"`
	AvgScrollPct     float64        `json:"avg_scroll_pct"`
	AvgReadTimeSec   float64        `json:"avg_read_time_sec"`
	TopPosts         []PostStat     `json:"top_posts"`
	TopReferrers     []ReferrerStat `json:"top_referrers"`
	DeviceBreakdown  []DeviceStat   `json:"device_breakdown"`
	CountryBreakdown []CountryStat  `json:"country_breakdown"`
	DailyViews       []DailyStat    `json:"daily_views"`
}

type PostStat struct {
	PostID         string  `json:"post_id"`
	PostTitle      string  `json:"post_title"`
	PostSlug       string  `json:"post_slug"`
	Views          int64   `json:"views"`
	UniqueVisitors int64   `json:"unique_visitors"`
	AvgScrollPct   float64 `json:"avg_scroll_pct"`
}

type PostDetailResponse struct {
	PostID          string       `json:"post_id"`
	PostTitle       string       `json:"post_title"`
	TotalViews      int64        `json:"total_views"`
	UniqueVisitors  int64        `json:"unique_visitors"`
	AvgScrollPct    float64      `json:"avg_scroll_pct"`
	ReadCompletions int64        `json:"read_completions"`
	CompletionRate  float64      `json:"completion_rate"`
	DailyViews      []DailyStat  `json:"daily_views"`
	DeviceBreakdown []DeviceStat `json:"device_breakdown"`
}

type ReferrerStat struct {
	Referrer string `json:"referrer"`
	Count    int64  `json:"count"`
}

type DeviceStat struct {
	Device string  `json:"device"`
	Count  int64   `json:"count"`
	Pct    float64 `json:"pct"`
}

type CountryStat struct {
	Country string  `json:"country"`
	Count   int64   `json:"count"`
	Pct     float64 `json:"pct"`
}

type DailyStat struct {
	Date           string `json:"date"`
	PageViews      int64  `json:"page_views"`
	UniqueVisitors int64  `json:"unique_visitors"`
}
