package posts

import (
	"context"

	"krishblog/internal/database"
	"krishblog/pkg/pagination"
)

type Service struct {
	repo  *Repository
	redis *database.Redis
}

func NewService(db *database.Postgres, redis *database.Redis) *Service {
	return &Service{repo: NewRepository(db), redis: redis}
}

func (s *Service) ListPublished(ctx context.Context, section, tag, query string, p pagination.Params) ([]PostResponse, int64, error) {
	return s.repo.ListPublished(ctx, section, tag, query, p)
}

func (s *Service) GetBySlug(ctx context.Context, slug string) (*PostResponse, error) {
	return s.repo.GetBySlug(ctx, slug)
}

func (s *Service) AdminList(ctx context.Context, status, section string, p pagination.Params) ([]PostResponse, int64, error) {
	return s.repo.AdminList(ctx, status, section, p)
}

func (s *Service) Create(ctx context.Context, authorID string, req CreateRequest) (*PostResponse, error) {
	return s.repo.Create(ctx, authorID, req)
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (*PostResponse, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *Service) UpdateStatus(ctx context.Context, id string, status PostStatus) (*PostResponse, error) {
	return s.repo.UpdateStatus(ctx, id, status)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) AdminGetBySlug(ctx context.Context, slug string) (*PostResponse, error) {
	return s.repo.AdminGetBySlug(ctx, slug)
}
