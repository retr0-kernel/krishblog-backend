package database

import (
	"context"
	"fmt"
)

// RunCustomMigrations creates tables not managed by Ent.
// Idempotent — uses IF NOT EXISTS.
func RunCustomMigrations(ctx context.Context, pg *Postgres) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS subscribers (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL DEFAULT '',
			token TEXT NOT NULL UNIQUE,
			confirmed BOOLEAN NOT NULL DEFAULT FALSE,
			confirmed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_subscribers_token ON subscribers(token)`,
		`CREATE INDEX IF NOT EXISTS idx_subscribers_confirmed ON subscribers(confirmed)`,
		`CREATE TABLE IF NOT EXISTS comments (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
			parent_id UUID REFERENCES comments(id) ON DELETE CASCADE,
			author_name TEXT NOT NULL,
			author_email TEXT NOT NULL DEFAULT '',
			content TEXT NOT NULL,
			is_approved BOOLEAN NOT NULL DEFAULT FALSE,
			is_admin_reply BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments(post_id)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_parent_id ON comments(parent_id)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_approved ON comments(is_approved)`,
		`CREATE TABLE IF NOT EXISTS claps (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
			session_id TEXT NOT NULL,
			count INT NOT NULL DEFAULT 1,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE(post_id, session_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_claps_post_id ON claps(post_id)`,
	}

	for _, stmt := range statements {
		if _, err := pg.DB.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("custom migration: %w", err)
		}
	}
	return nil
}
