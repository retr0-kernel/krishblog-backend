package claps

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

// AddClap upserts clap count for a session, capped at 50 per session.
func (r *Repository) AddClap(ctx context.Context, postID, sessionID string, count int) (*ClapResponse, error) {
	const q = `
		INSERT INTO claps (post_id, session_id, count)
		VALUES ($1, $2, LEAST($3, 50))
		ON CONFLICT (post_id, session_id)
		DO UPDATE SET count = LEAST(claps.count + $3, 50), updated_at = NOW()
		RETURNING count`

	var userClaps int
	err := r.db.DB.QueryRowContext(ctx, q, postID, sessionID, count).Scan(&userClaps)
	if err != nil {
		return nil, fmt.Errorf("upsert clap: %w", err)
	}

	total, err := r.TotalClaps(ctx, postID)
	if err != nil {
		return nil, err
	}

	return &ClapResponse{PostID: postID, TotalClaps: total, UserClaps: userClaps}, nil
}

// TotalClaps returns total clap count for a post.
func (r *Repository) TotalClaps(ctx context.Context, postID string) (int64, error) {
	var total sql.NullInt64
	err := r.db.DB.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(count),0) FROM claps WHERE post_id=$1`, postID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("count claps: %w", err)
	}
	return total.Int64, nil
}

// SessionClaps returns the session's clap count for a post.
func (r *Repository) SessionClaps(ctx context.Context, postID, sessionID string) (int, error) {
	var count int
	err := r.db.DB.QueryRowContext(ctx,
		`SELECT COALESCE(count,0) FROM claps WHERE post_id=$1 AND session_id=$2`, postID, sessionID,
	).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("session claps: %w", err)
	}
	return count, nil
}
