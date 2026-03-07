package posts

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"krishblog/internal/database"
	"krishblog/pkg/pagination"
)

type Repository struct {
	db *database.Postgres
}

func NewRepository(db *database.Postgres) *Repository {
	return &Repository{db: db}
}

// ─── public ───────────────────────────────────────────────────────────────────

func (r *Repository) ListPublished(ctx context.Context, section, tag, query string, p pagination.Params) ([]PostResponse, int64, error) {
	args := []interface{}{}
	where := []string{"p.status = 'published'"}
	i := 1

	if section != "" {
		where = append(where, fmt.Sprintf("s.slug = $%d", i))
		args = append(args, section)
		i++
	}
	if tag != "" {
		where = append(where, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM post_tags pt JOIN tags t ON t.id=pt.tag_id WHERE pt.post_id=p.id AND t.slug=$%d)", i,
		))
		args = append(args, tag)
		i++
	}
	if query != "" {
		where = append(where, fmt.Sprintf(
			"(p.search_vector @@ plainto_tsquery('english',$%d) OR p.title ILIKE $%d OR p.excerpt ILIKE $%d)",
			i, i+1, i+2,
		))
		like := "%" + query + "%"
		args = append(args, query, like, like)
		i += 3
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	var total int64
	countQ := fmt.Sprintf(
		`SELECT COUNT(*) FROM posts p LEFT JOIN sections s ON s.id=p.section_id %s`,
		whereClause,
	)
	if err := r.db.DB.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count posts: %w", err)
	}

	orderBy := "p.published_at DESC"
	if query != "" {
		orderBy = fmt.Sprintf(
			"ts_rank(p.search_vector, plainto_tsquery('english',$%d)) DESC, p.published_at DESC", i,
		)
		args = append(args, query)
		i++
	}

	args = append(args, p.Limit, p.Offset)
	listQ := fmt.Sprintf(`
		SELECT p.id, p.section_id, COALESCE(s.slug,''), p.author_id, p.title, p.slug,
		       COALESCE(p.excerpt,''), COALESCE(p.cover_image,''), COALESCE(p.cover_image_alt,''),
		       p.status, p.is_featured, p.reading_time_min, p.word_count,
		       p.published_at, p.created_at, p.updated_at
		FROM posts p
		LEFT JOIN sections s ON s.id = p.section_id
		%s
		ORDER BY %s
		LIMIT $%d OFFSET $%d`, whereClause, orderBy, i, i+1)

	posts, err := r.scanList(ctx, listQ, args...)
	return posts, total, err
}

func (r *Repository) GetBySlug(ctx context.Context, slugStr string) (*PostResponse, error) {
	const q = `
		SELECT p.id, p.section_id, COALESCE(s.slug,''), p.author_id, p.title, p.slug,
		       COALESCE(p.excerpt,''), COALESCE(p.cover_image,''), COALESCE(p.cover_image_alt,''),
		       p.status, p.is_featured, p.reading_time_min, p.word_count,
		       p.published_at, p.created_at, p.updated_at,
		       COALESCE(p.content,''), COALESCE(p.meta_title,''), COALESCE(p.meta_desc,'')
		FROM posts p
		LEFT JOIN sections s ON s.id = p.section_id
		WHERE p.slug = $1 AND p.status = 'published'`
	return r.scanOne(ctx, q, slugStr)
}

// ─── admin ────────────────────────────────────────────────────────────────────

func (r *Repository) AdminList(ctx context.Context, status, section string, p pagination.Params) ([]PostResponse, int64, error) {
	args := []interface{}{}
	where := []string{"1=1"}
	i := 1

	if status != "" {
		where = append(where, fmt.Sprintf("p.status = $%d", i))
		args = append(args, status)
		i++
	}
	if section != "" {
		where = append(where, fmt.Sprintf("s.slug = $%d", i))
		args = append(args, section)
		i++
	}

	whereClause := "WHERE " + strings.Join(where, " AND ")

	var total int64
	countQ := fmt.Sprintf(
		`SELECT COUNT(*) FROM posts p LEFT JOIN sections s ON s.id=p.section_id %s`,
		whereClause,
	)
	if err := r.db.DB.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count admin posts: %w", err)
	}

	args = append(args, p.Limit, p.Offset)
	listQ := fmt.Sprintf(`
		SELECT p.id, p.section_id, COALESCE(s.slug,''), p.author_id, p.title, p.slug,
		       COALESCE(p.excerpt,''), COALESCE(p.cover_image,''), COALESCE(p.cover_image_alt,''),
		       p.status, p.is_featured, p.reading_time_min, p.word_count,
		       p.published_at, p.created_at, p.updated_at
		FROM posts p
		LEFT JOIN sections s ON s.id = p.section_id
		%s
		ORDER BY p.updated_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, i, i+1)

	posts, err := r.scanList(ctx, listQ, args...)
	return posts, total, err
}

func (r *Repository) Create(ctx context.Context, authorID string, req CreateRequest) (*PostResponse, error) {
	sl := req.Slug
	if sl == "" {
		sl = makeSlug(req.Title)
	}
	status := req.Status
	if status == "" {
		status = StatusDraft
	}
	var publishedAt *time.Time
	if status == StatusPublished {
		now := time.Now()
		publishedAt = &now
	}
	wc := wordCount(req.Content)
	rtm := readingTime(wc)

	const q = `
		INSERT INTO posts (
			section_id, author_id, title, slug, excerpt, content,
			cover_image, cover_image_alt, status, is_featured,
			reading_time_min, word_count, meta_title, meta_desc,
			scheduled_at, published_at,
			search_vector, created_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,
			to_tsvector('english', $3||' '||coalesce($5,'')||' '||coalesce($6,'')),
			NOW(), NOW()
		)
		RETURNING id, section_id, ''::text, author_id, title, slug,
		          COALESCE(excerpt,''), COALESCE(cover_image,''), COALESCE(cover_image_alt,''),
		          status, is_featured, reading_time_min, word_count,
		          published_at, created_at, updated_at,
		          COALESCE(content,''), COALESCE(meta_title,''), COALESCE(meta_desc,'')`

	return r.scanOne(ctx, q,
		req.SectionID, authorID, req.Title, sl, req.Excerpt, req.Content,
		req.CoverImage, req.CoverImageAlt, status, req.IsFeatured,
		rtm, wc, req.MetaTitle, req.MetaDesc, req.ScheduledAt, publishedAt,
	)
}

func (r *Repository) Update(ctx context.Context, id string, req UpdateRequest) (*PostResponse, error) {
	curr, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	title := coalesceStr(req.Title, curr.Title)
	excerpt := coalesceStr(req.Excerpt, curr.Excerpt)
	content := coalesceStr(req.Content, curr.Content)
	sl := coalesceStr(req.Slug, curr.Slug)
	coverImage := coalesceStr(req.CoverImage, curr.CoverImage)
	coverImageAlt := coalesceStr(req.CoverImageAlt, curr.CoverImageAlt)
	sectionID := coalesceStr(req.SectionID, curr.SectionID)
	metaTitle := coalesceStr(req.MetaTitle, curr.MetaTitle)
	metaDesc := coalesceStr(req.MetaDesc, curr.MetaDesc)
	isFeatured := curr.IsFeatured
	if req.IsFeatured != nil {
		isFeatured = *req.IsFeatured
	}

	wc := wordCount(content)
	rtm := readingTime(wc)

	const q = `
		UPDATE posts SET
			section_id=$1, title=$2, slug=$3, excerpt=$4, content=$5,
			cover_image=$6, cover_image_alt=$7, is_featured=$8,
			reading_time_min=$9, word_count=$10,
			meta_title=$11, meta_desc=$12, scheduled_at=$13,
			updated_at=NOW(),
			search_vector=to_tsvector('english',$2||' '||coalesce($4,'')||' '||coalesce($5,''))
		WHERE id=$14
		RETURNING id, section_id, ''::text, author_id, title, slug,
		          COALESCE(excerpt,''), COALESCE(cover_image,''), COALESCE(cover_image_alt,''),
		          status, is_featured, reading_time_min, word_count,
		          published_at, created_at, updated_at,
		          COALESCE(content,''), COALESCE(meta_title,''), COALESCE(meta_desc,'')`

	return r.scanOne(ctx, q,
		sectionID, title, sl, excerpt, content,
		coverImage, coverImageAlt, isFeatured,
		rtm, wc, metaTitle, metaDesc, req.ScheduledAt, id,
	)
}

func (r *Repository) UpdateStatus(ctx context.Context, id string, status PostStatus) (*PostResponse, error) {
	publishedClause := ""
	if status == StatusPublished {
		publishedClause = ", published_at = COALESCE(published_at, NOW())"
	}
	q := fmt.Sprintf(`
		UPDATE posts SET status=$1, updated_at=NOW()%s
		WHERE id=$2
		RETURNING id, section_id, ''::text, author_id, title, slug,
		          COALESCE(excerpt,''), COALESCE(cover_image,''), COALESCE(cover_image_alt,''),
		          status, is_featured, reading_time_min, word_count,
		          published_at, created_at, updated_at,
		          COALESCE(content,''), COALESCE(meta_title,''), COALESCE(meta_desc,'')`,
		publishedClause)

	return r.scanOne(ctx, q, status, id)
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	_, err := r.db.DB.ExecContext(ctx, `DELETE FROM posts WHERE id=$1`, id)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id string) (*PostResponse, error) {
	const q = `
		SELECT p.id, p.section_id, COALESCE(s.slug,''), p.author_id, p.title, p.slug,
		       COALESCE(p.excerpt,''), COALESCE(p.cover_image,''), COALESCE(p.cover_image_alt,''),
		       p.status, p.is_featured, p.reading_time_min, p.word_count,
		       p.published_at, p.created_at, p.updated_at,
		       COALESCE(p.content,''), COALESCE(p.meta_title,''), COALESCE(p.meta_desc,'')
		FROM posts p
		LEFT JOIN sections s ON s.id = p.section_id
		WHERE p.id=$1`
	return r.scanOne(ctx, q, id)
}

// ─── scan helpers ─────────────────────────────────────────────────────────────

// scanList — 17 columns (no content/meta, has section_slug)
func (r *Repository) scanList(_ context.Context, q string, args ...interface{}) ([]PostResponse, error) {
	rows, err := r.db.DB.QueryContext(context.Background(), q, args...)
	if err != nil {
		return nil, fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	var posts []PostResponse
	for rows.Next() {
		var p PostResponse
		if err := rows.Scan(
			&p.ID, &p.SectionID, &p.SectionSlug, &p.AuthorID, &p.Title, &p.Slug,
			&p.Excerpt, &p.CoverImage, &p.CoverImageAlt,
			&p.Status, &p.IsFeatured, &p.ReadingTimeMin, &p.WordCount,
			&p.PublishedAt, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan post: %w", err)
		}
		posts = append(posts, p)
	}
	if posts == nil {
		posts = []PostResponse{}
	}
	return posts, rows.Err()
}

// scanOne — 19 columns (includes content + meta, has section_slug)
func (r *Repository) scanOne(_ context.Context, q string, args ...interface{}) (*PostResponse, error) {
	var p PostResponse
	err := r.db.DB.QueryRowContext(context.Background(), q, args...).Scan(
		&p.ID, &p.SectionID, &p.SectionSlug, &p.AuthorID, &p.Title, &p.Slug,
		&p.Excerpt, &p.CoverImage, &p.CoverImageAlt,
		&p.Status, &p.IsFeatured, &p.ReadingTimeMin, &p.WordCount,
		&p.PublishedAt, &p.CreatedAt, &p.UpdatedAt,
		&p.Content, &p.MetaTitle, &p.MetaDesc,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("post not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scan post: %w", err)
	}
	return &p, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func wordCount(s string) int { return len(strings.Fields(s)) }
func readingTime(wc int) int {
	if wc/200 < 1 {
		return 1
	}
	return wc / 200
}
func coalesceStr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func makeSlug(title string) string {
	s := strings.ToLower(title)
	s = strings.NewReplacer(" ", "-", "_", "-").Replace(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-")
}
