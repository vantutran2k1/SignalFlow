package service

import (
	"context"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

type NotificationService struct {
	repo domain.NotificationRepository
}

func NewNotificationService(repo domain.NotificationRepository) *NotificationService {
	return &NotificationService{repo: repo}
}

func (s *NotificationService) ListByExecution(ctx context.Context, executionID string) ([]domain.Notification, error) {
	return s.repo.ListByExecution(ctx, executionID)
}

func (s *NotificationService) ListRecent(ctx context.Context, limit int) ([]domain.Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.ListRecent(ctx, limit)
}
