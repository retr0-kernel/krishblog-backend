package claps

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

func (s *Service) Clap(ctx context.Context, postID, sessionID string, count int) (*ClapResponse, error) {
	return s.repo.AddClap(ctx, postID, sessionID, count)
}

func (s *Service) GetStats(ctx context.Context, postID, sessionID string) (*ClapResponse, error) {
	total, err := s.repo.TotalClaps(ctx, postID)
	if err != nil {
		return nil, err
	}
	userClaps, err := s.repo.SessionClaps(ctx, postID, sessionID)
	if err != nil {
		return nil, err
	}
	return &ClapResponse{PostID: postID, TotalClaps: total, UserClaps: userClaps}, nil
}
