package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{pool: pool}
}

func (r *NotificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO notifications (id, execution_id, channel_id, status, payload, error, sent_at, created_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, now())`,
		n.ID, n.ExecutionID, n.ChannelID, n.Status, n.Payload, n.Error, n.SentAt,
	)
	return err
}

func (r *NotificationRepository) ListByExecution(ctx context.Context, executionID string) ([]domain.Notification, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, execution_id, channel_id, status, payload, error, sent_at, created_at
         FROM notifications WHERE execution_id = $1 ORDER BY created_at DESC`, executionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []domain.Notification
	for rows.Next() {
		var n domain.Notification
		if err := rows.Scan(
			&n.ID, &n.ExecutionID, &n.ChannelID, &n.Status, &n.Payload,
			&n.Error, &n.SentAt, &n.CreatedAt,
		); err != nil {
			return nil, err
		}
		notifs = append(notifs, n)
	}
	return notifs, nil
}

func (r *NotificationRepository) ListRecent(ctx context.Context, limit int) ([]domain.Notification, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, execution_id, channel_id, status, payload, error, sent_at, created_at
         FROM notifications ORDER BY created_at DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifs []domain.Notification
	for rows.Next() {
		var n domain.Notification
		if err := rows.Scan(
			&n.ID, &n.ExecutionID, &n.ChannelID, &n.Status, &n.Payload,
			&n.Error, &n.SentAt, &n.CreatedAt,
		); err != nil {
			return nil, err
		}
		notifs = append(notifs, n)
	}
	return notifs, nil
}

func (r *NotificationRepository) UpdateStatus(ctx context.Context, id string, status domain.NotificationStatus, errMsg string) error {
	var sentAt *time.Time
	if status == domain.NotifStatusSent {
		now := time.Now()
		sentAt = &now
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE notifications SET status=$1, error=$2, sent_at=$3 WHERE id=$4`,
		status, errMsg, sentAt, id,
	)
	return err
}