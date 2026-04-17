package domain

import (
	"context"
	"encoding/json"
	"time"
)

type JobType string

const (
	JobTypeHTTPCheck JobType = "http_check"
	JobTypeCommand   JobType = "command"
	JobTypeRSSWatch  JobType = "rss_watch"
)

type JobStatus string

const (
	JobStatusActive   JobStatus = "active"
	JobStatusPaused   JobStatus = "paused"
	JobStatusDisabled JobStatus = "disabled"
)

type Job struct {
	ID             string          `json:"id"`
	UserID         string          `json:"user_id"`
	Name           string          `json:"name"`
	Type           JobType         `json:"type"`
	Schedule       string          `json:"schedule"` // Cron expression: "*/5 * * * *"
	Config         json.RawMessage `json:"config"`   // Type-specific config
	Status         JobStatus       `json:"status"`
	NotifyChannels []string        `json:"notify_channels"` // Channel IDs
	Condition      json.RawMessage `json:"condition"`       // e.g. {"on":"failure"}
	LastRunAt      *time.Time      `json:"last_run_at"`
	NextRunAt      *time.Time      `json:"next_run_at"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type JobRepository interface {
	Create(ctx context.Context, job *Job) error
	GetByID(ctx context.Context, id string) (*Job, error)
	List(ctx context.Context, userID string, offset, limit int) ([]Job, int, error)
	ListActive(ctx context.Context) ([]Job, error)
	ListDue(ctx context.Context) ([]Job, error) // WHERE status='active' AND next_run_at <= now()
	Update(ctx context.Context, job *Job) error
	Delete(ctx context.Context, id string) error
}
