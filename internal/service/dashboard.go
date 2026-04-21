package service

import (
	"context"
	"time"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

type DashboardStats struct {
	TotalJobs         int
	ActiveJobs        int
	RecentFailures    int
	NotificationsSent int
}

type DashboardService struct {
	jobRepo   domain.JobRepository
	execRepo  domain.ExecutionRepository
	notifRepo domain.NotificationRepository
}

func NewDashboardService(
	jobRepo domain.JobRepository,
	execRepo domain.ExecutionRepository,
	notifRepo domain.NotificationRepository,
) *DashboardService {
	return &DashboardService{
		jobRepo:   jobRepo,
		execRepo:  execRepo,
		notifRepo: notifRepo,
	}
}

// Stats returns user-scoped aggregate counters for the dashboard landing page.
// Each sub-query runs independently; a failure of one does not fail the whole
// response — missing fields read as zero. The caller should log the error but
// can still render a partial dashboard.
func (s *DashboardService) Stats(ctx context.Context, userID string) (*DashboardStats, error) {
	stats := &DashboardStats{}
	since := time.Now().Add(-24 * time.Hour)

	if v, err := s.jobRepo.CountByUser(ctx, userID); err == nil {
		stats.TotalJobs = v
	} else {
		return stats, err
	}
	if v, err := s.jobRepo.CountActiveByUser(ctx, userID); err == nil {
		stats.ActiveJobs = v
	} else {
		return stats, err
	}
	if v, err := s.execRepo.CountFailuresByUserSince(ctx, userID, since); err == nil {
		stats.RecentFailures = v
	} else {
		return stats, err
	}
	if v, err := s.notifRepo.CountSentByUserSince(ctx, userID, since); err == nil {
		stats.NotificationsSent = v
	} else {
		return stats, err
	}
	return stats, nil
}

func (s *DashboardService) RecentExecutions(ctx context.Context, userID string, limit int) ([]domain.Execution, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.execRepo.ListRecentByUser(ctx, userID, limit)
}
