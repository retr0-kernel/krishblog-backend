package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// Post holds the schema definition for the Post entity.
type Post struct {
	ent.Schema
}

// Fields of the Post.
func (Post) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).
			Default(uuid.New).
			Immutable(),

		// Foreign keys — stored as UUID columns
		field.UUID("section_id", uuid.UUID{}).
			Comment("Section this post belongs to"),

		field.UUID("author_id", uuid.UUID{}).
			Comment("User who created this post"),

		field.String("title").
			NotEmpty().
			MaxLen(300).
			Comment("Post headline"),

		field.String("slug").
			Unique().
			NotEmpty().
			MaxLen(300).
			Match(slugRegexp).
			Comment("URL-safe identifier"),

		field.String("summary").
			Optional().
			MaxLen(500).
			Comment("Short excerpt shown in listing views"),

		field.Text("content").
			Optional().
			Comment("Full markdown body of the post"),

		field.String("cover_image").
			Optional().
			Nillable().
			MaxLen(2048).
			Comment("R2 URL for hero image"),

		field.String("cover_image_alt").
			Optional().
			MaxLen(300).
			Comment("Alt text for cover image"),

		field.Enum("status").
			Values("draft", "scheduled", "published", "archived").
			Default("draft").
			Comment("Publication lifecycle status"),

		field.Bool("published").
			Default(false).
			Comment("Convenience flag — true when status=published"),

		field.Int("read_time").
			Default(0).
			NonNegative().
			Comment("Estimated reading time in minutes"),

		field.Int("word_count").
			Default(0).
			NonNegative(),

		field.Bool("is_featured").
			Default(false).
			Comment("Pin to featured slots on the homepage"),

		field.String("meta_title").
			Optional().
			MaxLen(70),

		field.String("meta_desc").
			Optional().
			MaxLen(160),

		field.String("og_image").
			Optional().
			Nillable().
			MaxLen(2048).
			Comment("Open Graph image override"),

		field.Time("published_at").
			Optional().
			Nillable().
			Comment("Set when status transitions to published"),

		field.Time("scheduled_at").
			Optional().
			Nillable().
			Comment("Future publish datetime for scheduled posts"),

		field.Time("created_at").
			Default(time.Now).
			Immutable(),

		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

// Edges of the Post.
func (Post) Edges() []ent.Edge {
	return []ent.Edge{
		// Many posts → one section
		edge.From("section", Section.Type).
			Ref("posts").
			Field("section_id").
			Required().
			Unique().
			Comment("Section this post belongs to"),

		// Many posts → one author
		edge.From("author", User.Type).
			Ref("posts").
			Field("author_id").
			Required().
			Unique().
			Comment("Authoring user"),

		// One post → many blocks (ordered content)
		edge.To("blocks", PostBlock.Type).
			Comment("Ordered rich-content blocks"),

		// One post → many analytics events
		edge.To("analytics_events", AnalyticsEvent.Type).
			Comment("All analytics events recorded for this post"),
	}
}

// Indexes of the Post.
func (Post) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("slug").Unique(),
		index.Fields("section_id"),
		index.Fields("author_id"),
		index.Fields("status"),
		index.Fields("published"),
		index.Fields("is_featured"),
		index.Fields("published_at"),
		index.Fields("scheduled_at"),
		// Composite: published posts ordered by date (used by reader listing)
		index.Fields("status", "published_at"),
		// Composite: section + status (used by admin filtering)
		index.Fields("section_id", "status"),
	}
}
