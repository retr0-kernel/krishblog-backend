package comments

import (
	"context"

	"krishblog/internal/database"
)

type Service struct {
	repo *Repository
}

func NewService(db *database.Postgres) *Service {
	return &Service{repo: NewRepository(db)}
}

func (s *Service) ListPublic(ctx context.Context, postID string) ([]PublicCommentResponse, error) {
	return s.repo.ListPublicByPost(ctx, postID)
}

func (s *Service) ListByPost(ctx context.Context, postID string) ([]CommentResponse, error) {
	return s.repo.ListAllByPost(ctx, postID)
}

func (s *Service) ListAll(ctx context.Context, approved *bool) ([]CommentResponse, error) {
	return s.repo.ListAll(ctx, approved)
}

func (s *Service) Create(ctx context.Context, postID string, req CreateRequest) (*CommentResponse, error) {
	return s.repo.Create(ctx, postID, req)
}

func (s *Service) Reply(ctx context.Context, postID, parentID, content string) (*CommentResponse, error) {
	return s.repo.Reply(ctx, postID, parentID, content)
}

func (s *Service) SetApproval(ctx context.Context, id string, approved bool) (*CommentResponse, error) {
	return s.repo.SetApproval(ctx, id, approved)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
