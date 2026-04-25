package service

import (
	"context"
	"fmt"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

type ExecutionService struct {
	repo domain.ExecutionRepository
}

func NewExecutionService(repo domain.ExecutionRepository) *ExecutionService {
	return &ExecutionService{repo: repo}
}

func (s *ExecutionService) GetByID(ctx context.Context, id string) (*domain.Execution, error) {
	exec, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	return exec, nil
}

func (s *ExecutionService) ListByJob(ctx context.Context, jobID string, offset, limit int) ([]domain.Execution, int, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.ListByJob(ctx, jobID, offset, limit)
}

func (s *ExecutionService) ListRecent(ctx context.Context, limit int) ([]domain.Execution, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListRecent(ctx, limit)
}
