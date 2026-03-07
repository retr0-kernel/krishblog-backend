package subscribers

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"krishblog/internal/database"
)

type Subscriber struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	Name        string     `json:"name,omitempty"`
	Confirmed   bool       `json:"confirmed"`
	Token       string     `json:"-"`
	ConfirmedAt *time.Time `json:"confirmed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Repository struct {
	db *database.Postgres
}

func NewRepository(db *database.Postgres) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, email, name, token string) (*Subscriber, error) {
	const q = `
		INSERT INTO subscribers (email, name, token, confirmed, created_at, updated_at)
		VALUES ($1, $2, $3, false, NOW(), NOW())
		ON CONFLICT (email) DO UPDATE
		  SET name=EXCLUDED.name, token=EXCLUDED.token, updated_at=NOW()
		RETURNING id, email, name, confirmed, token, confirmed_at, created_at`

	s := &Subscriber{}
	err := r.db.DB.QueryRowContext(ctx, q, email, name, token).Scan(
		&s.ID, &s.Email, &s.Name, &s.Confirmed, &s.Token, &s.ConfirmedAt, &s.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create subscriber: %w", err)
	}
	return s, nil
}

func (r *Repository) Confirm(ctx context.Context, token string) (*Subscriber, error) {
	const q = `
		UPDATE subscribers
		SET confirmed=true, confirmed_at=NOW(), updated_at=NOW()
		WHERE token=$1 AND confirmed=false
		RETURNING id, email, name, confirmed, token, confirmed_at, created_at`

	s := &Subscriber{}
	err := r.db.DB.QueryRowContext(ctx, q, token).Scan(
		&s.ID, &s.Email, &s.Name, &s.Confirmed, &s.Token, &s.ConfirmedAt, &s.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("confirm subscriber: %w", err)
	}
	return s, nil
}

func (r *Repository) Unsubscribe(ctx context.Context, token string) error {
	res, err := r.db.DB.ExecContext(ctx, `DELETE FROM subscribers WHERE token=$1`, token)
	if err != nil {
		return fmt.Errorf("unsubscribe: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ListConfirmed(ctx context.Context) ([]Subscriber, error) {
	const q = `
		SELECT id, email, name, confirmed, token, confirmed_at, created_at
		FROM subscribers WHERE confirmed=true ORDER BY created_at DESC`

	rows, err := r.db.DB.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list confirmed: %w", err)
	}
	defer rows.Close()

	var subs []Subscriber
	for rows.Next() {
		var s Subscriber
		if err := rows.Scan(
			&s.ID, &s.Email, &s.Name, &s.Confirmed,
			&s.Token, &s.ConfirmedAt, &s.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan subscriber: %w", err)
		}
		subs = append(subs, s)
	}
	if subs == nil {
		subs = []Subscriber{}
	}
	return subs, rows.Err()
}

func (r *Repository) Count(ctx context.Context) (total int, confirmed int, err error) {
	const q = `SELECT COUNT(*), COUNT(*) FILTER (WHERE confirmed=true) FROM subscribers`
	err = r.db.DB.QueryRowContext(ctx, q).Scan(&total, &confirmed)
	return
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*Subscriber, error) {
	const q = `
		SELECT id, email, name, confirmed, token, confirmed_at, created_at
		FROM subscribers WHERE email=$1`

	s := &Subscriber{}
	err := r.db.DB.QueryRowContext(ctx, q, email).Scan(
		&s.ID, &s.Email, &s.Name, &s.Confirmed,
		&s.Token, &s.ConfirmedAt, &s.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get by email: %w", err)
	}
	return s, nil
}
