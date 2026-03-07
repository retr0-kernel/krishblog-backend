package posts

import "time"

type PostStatus string

const (
	StatusDraft     PostStatus = "draft"
	StatusScheduled PostStatus = "scheduled"
	StatusPublished PostStatus = "published"
	StatusArchived  PostStatus = "archived"
)

type CreateRequest struct {
	SectionID     string     `json:"section_id"      validate:"required,uuid4"`
	Title         string     `json:"title"           validate:"required,min=3,max=300"`
	Slug          string     `json:"slug"            validate:"omitempty,min=3,max=300"`
	Excerpt       string     `json:"excerpt"         validate:"omitempty,max=500"`
	Content       string     `json:"content"         validate:"required"`
	CoverImage    string     `json:"cover_image"     validate:"omitempty,url"`
	CoverImageAlt string     `json:"cover_image_alt"`
	Status        PostStatus `json:"status"          validate:"omitempty,oneof=draft scheduled published archived"`
	IsFeatured    bool       `json:"is_featured"`
	MetaTitle     string     `json:"meta_title"      validate:"omitempty,max=70"`
	MetaDesc      string     `json:"meta_desc"       validate:"omitempty,max=160"`
	ScheduledAt   *time.Time `json:"scheduled_at"`
}

type UpdateRequest struct {
	SectionID     string     `json:"section_id"      validate:"omitempty,uuid4"`
	Title         string     `json:"title"           validate:"omitempty,min=3,max=300"`
	Slug          string     `json:"slug"            validate:"omitempty,min=3,max=300"`
	Excerpt       string     `json:"excerpt"         validate:"omitempty,max=500"`
	Content       string     `json:"content"`
	CoverImage    string     `json:"cover_image"     validate:"omitempty,url"`
	CoverImageAlt string     `json:"cover_image_alt"`
	IsFeatured    *bool      `json:"is_featured"`
	MetaTitle     string     `json:"meta_title"      validate:"omitempty,max=70"`
	MetaDesc      string     `json:"meta_desc"       validate:"omitempty,max=160"`
	ScheduledAt   *time.Time `json:"scheduled_at"`
}

type StatusRequest struct {
	Status PostStatus `json:"status" validate:"required,oneof=draft scheduled published archived"`
}

type PostResponse struct {
	ID             string     `json:"id"`
	SectionID      string     `json:"section_id"`
	SectionSlug    string     `json:"section_slug,omitempty"`
	AuthorID       string     `json:"author_id"`
	Title          string     `json:"title"`
	Slug           string     `json:"slug"`
	Excerpt        string     `json:"excerpt,omitempty"`
	Content        string     `json:"content,omitempty"`
	CoverImage     string     `json:"cover_image,omitempty"`
	CoverImageAlt  string     `json:"cover_image_alt,omitempty"`
	Status         PostStatus `json:"status"`
	IsFeatured     bool       `json:"is_featured"`
	ReadingTimeMin int        `json:"reading_time_min"`
	WordCount      int        `json:"word_count"`
	MetaTitle      string     `json:"meta_title,omitempty"`
	MetaDesc       string     `json:"meta_desc,omitempty"`
	PublishedAt    *time.Time `json:"published_at,omitempty"`
	ScheduledAt    *time.Time `json:"scheduled_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
