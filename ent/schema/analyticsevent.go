package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// AnalyticsEvent holds the schema definition for the AnalyticsEvent entity.
type AnalyticsEvent struct {
	ent.Schema
}

// Fields of the AnalyticsEvent.
func (AnalyticsEvent) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),

		// post_id is optional — page_view events may not target a specific post
		field.UUID("post_id", uuid.UUID{}).
			Optional().
			Nillable().
			Comment("Post being viewed — nil for non-post page views"),

		field.Enum("event_type").
			Values(
				"page_view",
				"post_view",
				"scroll_depth",
				"read_complete",
				"click",
				"search",
				"session_start",
				"session_end",
			).
			Comment("Discriminator for event shape"),

		field.String("session_id").
			NotEmpty().
			MaxLen(100).
			Comment("Anonymous session UUID stored in a first-party cookie"),

		field.Enum("device").
			Values("desktop", "tablet", "mobile", "unknown").
			Default("unknown").
			Comment("Derived from User-Agent"),

		field.String("country").
			Optional().
			MaxLen(2).
			Comment("ISO 3166-1 alpha-2 country code from Cloudflare CF-IPCountry header"),

		field.String("path").
			NotEmpty().
			MaxLen(2048).
			Comment("URL path of the page that fired the event"),

		field.String("referrer").
			Optional().
			MaxLen(2048).
			Comment("HTTP Referer header value"),

		field.Int("scroll_pct").
			Optional().
			Nillable().
			Min(0).
			Max(100).
			Comment("Scroll depth percentage — populated for scroll_depth events"),

		field.Int("duration_ms").
			Optional().
			Nillable().
			NonNegative().
			Comment("Session or page duration in milliseconds"),

		field.JSON("metadata", map[string]interface{}{}).
			Optional().
			Comment("Event-specific extra payload"),

		field.Time("timestamp").
			Default(time.Now).
			Immutable().
			Comment("When the event was recorded"),
	}
}

// Edges of the AnalyticsEvent.
func (AnalyticsEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("post", Post.Type).
			Ref("analytics_events").
			Field("post_id").
			Unique().
			Comment("Post this event belongs to — optional"),
	}
}

// Indexes of the AnalyticsEvent.
func (AnalyticsEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("event_type"),
		index.Fields("session_id"),
		index.Fields("post_id"),
		index.Fields("timestamp"),
		index.Fields("country"),
		index.Fields("device"),
		// Composite: post analytics over time
		index.Fields("post_id", "timestamp"),
		// Composite: session reconstruction
		index.Fields("session_id", "timestamp"),
	}
}
