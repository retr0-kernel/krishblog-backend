package database

import (
	"context"
	"fmt"

	"krishblog/ent"
	"krishblog/ent/migrate"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
)

// NewEntClient creates an Ent client that shares the existing *sql.DB pool.
func NewEntClient(pg *Postgres) (*ent.Client, error) {
	drv := entsql.OpenDB(dialect.Postgres, pg.DB)
	client := ent.NewClient(ent.Driver(drv))
	return client, nil
}

// RunMigrations applies all pending auto-migrations.
// Idempotent — safe to call on every boot in development.
func RunMigrations(ctx context.Context, client *ent.Client) error {
	if err := client.Schema.Create(
		ctx,
		migrate.WithDropIndex(true),
		migrate.WithDropColumn(false),
	); err != nil {
		return fmt.Errorf("ent migrate: %w", err)
	}
	return nil
}
