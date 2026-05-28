package comments

import "time"

type CreateRequest struct {
	AuthorName  string `json:"author_name"  validate:"required,min=2,max=100"`
	AuthorEmail string `json:"author_email" validate:"required,email"`
	Content     string `json:"content"      validate:"required,min=2,max=2000"`
}

type ReplyRequest struct {
	Content string `json:"content" validate:"required,min=2,max=2000"`
}

type ApproveRequest struct {
	IsApproved bool `json:"is_approved"`
}

type CommentResponse struct {
	ID           string            `json:"id"`
	PostID       string            `json:"post_id"`
	ParentID     *string           `json:"parent_id,omitempty"`
	AuthorName   string            `json:"author_name"`
	AuthorEmail  string            `json:"author_email,omitempty"` // omit in public responses
	Content      string            `json:"content"`
	IsApproved   bool              `json:"is_approved"`
	IsAdminReply bool              `json:"is_admin_reply"`
	Replies      []CommentResponse `json:"replies,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

type PublicCommentResponse struct {
	ID           string                  `json:"id"`
	PostID       string                  `json:"post_id"`
	ParentID     *string                 `json:"parent_id,omitempty"`
	AuthorName   string                  `json:"author_name"`
	Content      string                  `json:"content"`
	IsAdminReply bool                    `json:"is_admin_reply"`
	Replies      []PublicCommentResponse `json:"replies,omitempty"`
	CreatedAt    time.Time               `json:"created_at"`
}
