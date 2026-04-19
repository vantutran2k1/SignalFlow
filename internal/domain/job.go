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
	Schedule       string          `json:"schedule"`
	Config         json.RawMessage `json:"config"`
	Status         JobStatus       `json:"status"`
	NotifyChannels []string        `json:"notify_channels"`
	Condition      json.RawMessage `json:"condition"`
	TimeoutSeconds int             `json:"timeout_seconds"`
	LastRunAt      *time.Time      `json:"last_run_at"`
	NextRunAt      *time.Time      `json:"next_run_at"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// JobClaim is a job that has been atomically claimed for execution, along with
// the pre-created execution row (in 'running' state) that the worker must
// transition to a terminal state when it finishes.
type JobClaim struct {
	Job         Job
	ExecutionID string
}

// NextRunFunc computes the next run time for a cron schedule expression.
// Called inside the ClaimDue transaction to advance next_run_at.
type NextRunFunc func(schedule string) (time.Time, error)

type JobRepository interface {
	Create(ctx context.Context, job *Job) error
	GetByID(ctx context.Context, id string) (*Job, error)
	List(ctx context.Context, userID string, offset, limit int) ([]Job, int, error)
	ListActive(ctx context.Context) ([]Job, error)
	// ClaimDue atomically selects up to limit due jobs (FOR UPDATE SKIP LOCKED),
	// advances each job's next_run_at via next(), and inserts a 'running' execution
	// row for each claimed job. Concurrent schedulers skip rows locked by peers.
	ClaimDue(ctx context.Context, limit int, next NextRunFunc) ([]JobClaim, error)
	Update(ctx context.Context, job *Job) error
	Delete(ctx context.Context, id string) error
}
