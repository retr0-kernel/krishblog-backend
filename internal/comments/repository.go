package comments

import (
	"context"
	"database/sql"
	"fmt"

	"krishblog/internal/database"
)

type Repository struct {
	db *database.Postgres
}

func NewRepository(db *database.Postgres) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ListPublicByPost(ctx context.Context, postID string) ([]PublicCommentResponse, error) {
	const q = `
		SELECT id, post_id, parent_id, author_name, content, is_admin_reply, created_at
		FROM comments
		WHERE post_id = $1 AND is_approved = TRUE
		ORDER BY created_at ASC`

	rows, err := r.db.DB.QueryContext(ctx, q, postID)
	if err != nil {
		return nil, fmt.Errorf("list public comments: %w", err)
	}
	defer rows.Close()

	byID := map[string]*PublicCommentResponse{}
	var topLevel []PublicCommentResponse

	for rows.Next() {
		var c PublicCommentResponse
		var parentID sql.NullString
		if err := rows.Scan(&c.ID, &c.PostID, &parentID, &c.AuthorName, &c.Content, &c.IsAdminReply, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		if parentID.Valid {
			c.ParentID = &parentID.String
		}
		byID[c.ID] = &c
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Build tree
	for _, c := range byID {
		if c.ParentID != nil {
			parent, ok := byID[*c.ParentID]
			if ok {
				parent.Replies = append(parent.Replies, *c)
			}
		} else {
			topLevel = append(topLevel, *c)
		}
	}

	for i := range topLevel {
		if updated, ok := byID[topLevel[i].ID]; ok {
			topLevel[i].Replies = updated.Replies
		}
	}

	if topLevel == nil {
		topLevel = []PublicCommentResponse{}
	}
	return topLevel, nil
}

func (r *Repository) ListAllByPost(ctx context.Context, postID string) ([]CommentResponse, error) {
	const q = `
		SELECT id, post_id, parent_id, author_name, author_email, content, is_approved, is_admin_reply, created_at, updated_at
		FROM comments
		WHERE post_id = $1
		ORDER BY created_at ASC`

	return r.scanList(ctx, q, postID)
}

func (r *Repository) ListAll(ctx context.Context, approved *bool) ([]CommentResponse, error) {
	q := `
		SELECT id, post_id, parent_id, author_name, author_email, content, is_approved, is_admin_reply, created_at, updated_at
		FROM comments`
	args := []interface{}{}
	if approved != nil {
		q += " WHERE is_approved = $1"
		args = append(args, *approved)
	}
	q += " ORDER BY created_at DESC"
	return r.scanList(ctx, q, args...)
}

func (r *Repository) Create(ctx context.Context, postID string, req CreateRequest) (*CommentResponse, error) {
	const q = `
		INSERT INTO comments (post_id, author_name, author_email, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, post_id, parent_id, author_name, author_email, content, is_approved, is_admin_reply, created_at, updated_at`

	return r.scanOne(ctx, q, postID, req.AuthorName, req.AuthorEmail, req.Content)
}

func (r *Repository) Reply(ctx context.Context, postID, parentID, content string) (*CommentResponse, error) {
	const q = `
		INSERT INTO comments (post_id, parent_id, author_name, author_email, content, is_approved, is_admin_reply)
		VALUES ($1, $2, 'Admin', '', $3, TRUE, TRUE)
		RETURNING id, post_id, parent_id, author_name, author_email, content, is_approved, is_admin_reply, created_at, updated_at`

	return r.scanOne(ctx, q, postID, parentID, content)
}

func (r *Repository) SetApproval(ctx context.Context, id string, approved bool) (*CommentResponse, error) {
	const q = `
		UPDATE comments SET is_approved=$1, updated_at=NOW()
		WHERE id=$2
		RETURNING id, post_id, parent_id, author_name, author_email, content, is_approved, is_admin_reply, created_at, updated_at`

	return r.scanOne(ctx, q, approved, id)
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.DB.ExecContext(ctx, `DELETE FROM comments WHERE id=$1`, id)
	return err
}

func (r *Repository) scanOne(ctx context.Context, q string, args ...interface{}) (*CommentResponse, error) {
	var c CommentResponse
	var parentID sql.NullString
	err := r.db.DB.QueryRowContext(ctx, q, args...).Scan(
		&c.ID, &c.PostID, &parentID, &c.AuthorName, &c.AuthorEmail,
		&c.Content, &c.IsApproved, &c.IsAdminReply, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("comment not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scan comment: %w", err)
	}
	if parentID.Valid {
		c.ParentID = &parentID.String
	}
	return &c, nil
}

func (r *Repository) scanList(ctx context.Context, q string, args ...interface{}) ([]CommentResponse, error) {
	rows, err := r.db.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query comments: %w", err)
	}
	defer rows.Close()

	byID := map[string]*CommentResponse{}
	var topLevel []CommentResponse

	for rows.Next() {
		var c CommentResponse
		var parentID sql.NullString
		if err := rows.Scan(
			&c.ID, &c.PostID, &parentID, &c.AuthorName, &c.AuthorEmail,
			&c.Content, &c.IsApproved, &c.IsAdminReply, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}
		if parentID.Valid {
			c.ParentID = &parentID.String
		}
		byID[c.ID] = &c
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Build tree
	for _, c := range byID {
		if c.ParentID != nil {
			if parent, ok := byID[*c.ParentID]; ok {
				parent.Replies = append(parent.Replies, *c)
			} else {
				topLevel = append(topLevel, *c)
			}
		} else {
			topLevel = append(topLevel, *c)
		}
	}

	// Sync replies back into top-level array
	for i := range topLevel {
		if updated, ok := byID[topLevel[i].ID]; ok {
			topLevel[i].Replies = updated.Replies
		}
	}

	if topLevel == nil {
		topLevel = []CommentResponse{}
	}
	return topLevel, nil
}
