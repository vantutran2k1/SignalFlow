package domain

import (
	"context"
	"time"
)

type NotificationStatus string

const (
	NotifStatusPending NotificationStatus = "pending"
	NotifStatusSent    NotificationStatus = "sent"
	NotifStatusFailed  NotificationStatus = "failed"
)

type Notification struct {
	ID          string             `json:"id"`
	ExecutionID string             `json:"execution_id"`
	ChannelID   string             `json:"channel_id"`
	Status      NotificationStatus `json:"status"`
	Payload     string             `json:"payload"`
	Error       string             `json:"error"`
	SentAt      *time.Time         `json:"sent_at"`
	CreatedAt   time.Time          `json:"created_at"`
}

type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) error
	ListByExecution(ctx context.Context, executionID string) ([]Notification, error)
	ListRecent(ctx context.Context, limit int) ([]Notification, error)
	UpdateStatus(ctx context.Context, id string, status NotificationStatus, errMsg string) error
}
