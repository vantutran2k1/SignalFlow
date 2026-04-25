package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	cronlib "github.com/robfig/cron/v3"
	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

type JobService struct {
	repo   domain.JobRepository
	parser cronlib.Parser
}

func NewJobService(repo domain.JobRepository) *JobService {
	return &JobService{
		repo:   repo,
		parser: cronlib.NewParser(cronlib.Minute | cronlib.Hour | cronlib.Dom | cronlib.Month | cronlib.Dow),
	}
}

type CreateJobInput struct {
	UserID         string
	Name           string
	Type           domain.JobType
	Schedule       string
	Config         json.RawMessage
	NotifyChannels []string
	Condition      json.RawMessage
	TimeoutSeconds int
}

type UpdateJobInput struct {
	Name           *string
	Schedule       *string
	Config         *json.RawMessage
	NotifyChannels *[]string
	Condition      *json.RawMessage
	TimeoutSeconds *int
}

func (s *JobService) Create(ctx context.Context, in CreateJobInput) (*domain.Job, error) {
	sched, err := s.parser.Parse(in.Schedule)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid cron: %v", ErrInvalidInput, err)
	}
	nextRun := sched.Next(time.Now())

	condition := in.Condition
	if condition == nil {
		condition = json.RawMessage(`{"on":"failure"}`)
	}

	timeout := in.TimeoutSeconds
	if timeout <= 0 {
		timeout = 30
	}

	job := &domain.Job{
		ID:             uuid.NewString(),
		UserID:         in.UserID,
		Name:           in.Name,
		Type:           in.Type,
		Schedule:       in.Schedule,
		Config:         in.Config,
		Status:         domain.JobStatusActive,
		NotifyChannels: in.NotifyChannels,
		Condition:      condition,
		TimeoutSeconds: timeout,
		NextRunAt:      &nextRun,
	}

	if err := s.repo.Create(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *JobService) GetByID(ctx context.Context, id string) (*domain.Job, error) {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	return job, nil
}

func (s *JobService) List(ctx context.Context, userID string, offset, limit int) ([]domain.Job, int, error) {
	if limit <= 0 {
		limit = 20
	}
	return s.repo.List(ctx, userID, offset, limit)
}

func (s *JobService) Update(ctx context.Context, id string, in UpdateJobInput) (*domain.Job, error) {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotFound, err)
	}

	if in.Name != nil {
		job.Name = *in.Name
	}
	if in.Schedule != nil {
		sched, err := s.parser.Parse(*in.Schedule)
		if err != nil {
			return nil, fmt.Errorf("%w: invalid cron: %v", ErrInvalidInput, err)
		}
		job.Schedule = *in.Schedule
		next := sched.Next(time.Now())
		job.NextRunAt = &next
	}
	if in.Config != nil {
		job.Config = *in.Config
	}
	if in.NotifyChannels != nil {
		job.NotifyChannels = *in.NotifyChannels
	}
	if in.Condition != nil {
		job.Condition = *in.Condition
	}
	if in.TimeoutSeconds != nil && *in.TimeoutSeconds > 0 {
		job.TimeoutSeconds = *in.TimeoutSeconds
	}

	if err := s.repo.Update(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *JobService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *JobService) TriggerRun(ctx context.Context, id string) (*domain.Job, error) {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	now := time.Now()
	job.NextRunAt = &now
	if err := s.repo.Update(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *JobService) Pause(ctx context.Context, id string) (*domain.Job, error) {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	job.Status = domain.JobStatusPaused
	job.NextRunAt = nil
	if err := s.repo.Update(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *JobService) Resume(ctx context.Context, id string) (*domain.Job, error) {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	sched, err := s.parser.Parse(job.Schedule)
	if err != nil {
		return nil, fmt.Errorf("stored schedule invalid: %w", err)
	}
	job.Status = domain.JobStatusActive
	next := sched.Next(time.Now())
	job.NextRunAt = &next
	if err := s.repo.Update(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}
