package domain

import (
	"context"
	"time"
)

type ExecStatus string

const (
	ExecStatusSuccess ExecStatus = "success"
	ExecStatusFailure ExecStatus = "failure"
	ExecStatusError   ExecStatus = "error"
)

type Execution struct {
	ID         string     `json:"id"`
	JobID      string     `json:"job_id"`
	Status     ExecStatus `json:"status"`
	Output     string     `json:"output"`
	Error      string     `json:"error"`
	DurationMs int64      `json:"duration_ms"`
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt time.Time  `json:"finished_at"`
}

type ExecutionRepository interface {
	Create(ctx context.Context, exec *Execution) error
	GetByID(ctx context.Context, id string) (*Execution, error)
	ListByJob(ctx context.Context, jobID string, offset, limit int) ([]Execution, int, error)
	ListRecent(ctx context.Context, limit int) ([]Execution, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}
