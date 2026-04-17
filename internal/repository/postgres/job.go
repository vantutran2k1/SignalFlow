package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

type JobRepository struct {
	pool *pgxpool.Pool
}

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{pool: pool}
}

func (r *JobRepository) Create(ctx context.Context, job *domain.Job) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO jobs (id, user_id, name, type, schedule, config, status, notify_channels, condition, next_run_at, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), now())`,
		job.ID, job.UserID, job.Name, job.Type, job.Schedule,
		job.Config, job.Status, job.NotifyChannels, job.Condition, job.NextRunAt,
	)
	return err
}

func (r *JobRepository) GetByID(ctx context.Context, id string) (*domain.Job, error) {
	job := &domain.Job{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, type, schedule, config, status, notify_channels,
                condition, last_run_at, next_run_at, created_at, updated_at
         FROM jobs WHERE id = $1`, id,
	).Scan(
		&job.ID, &job.UserID, &job.Name, &job.Type, &job.Schedule,
		&job.Config, &job.Status, &job.NotifyChannels, &job.Condition,
		&job.LastRunAt, &job.NextRunAt, &job.CreatedAt, &job.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (r *JobRepository) List(ctx context.Context, userID string, offset, limit int) ([]domain.Job, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM jobs WHERE user_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, name, type, schedule, config, status, notify_channels,
                condition, last_run_at, next_run_at, created_at, updated_at
         FROM jobs WHERE user_id = $1
         ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		userID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		var j domain.Job
		if err := rows.Scan(
			&j.ID, &j.UserID, &j.Name, &j.Type, &j.Schedule,
			&j.Config, &j.Status, &j.NotifyChannels, &j.Condition,
			&j.LastRunAt, &j.NextRunAt, &j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, j)
	}
	return jobs, total, nil
}

func (r *JobRepository) ListActive(ctx context.Context) ([]domain.Job, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, name, type, schedule, config, status, notify_channels,
                condition, last_run_at, next_run_at, created_at, updated_at
         FROM jobs WHERE status = 'active'`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		var j domain.Job
		if err := rows.Scan(
			&j.ID, &j.UserID, &j.Name, &j.Type, &j.Schedule,
			&j.Config, &j.Status, &j.NotifyChannels, &j.Condition,
			&j.LastRunAt, &j.NextRunAt, &j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (r *JobRepository) ListDue(ctx context.Context) ([]domain.Job, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, name, type, schedule, config, status, notify_channels,
                condition, last_run_at, next_run_at, created_at, updated_at
         FROM jobs WHERE status = 'active' AND next_run_at <= now()`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		var j domain.Job
		if err := rows.Scan(
			&j.ID, &j.UserID, &j.Name, &j.Type, &j.Schedule,
			&j.Config, &j.Status, &j.NotifyChannels, &j.Condition,
			&j.LastRunAt, &j.NextRunAt, &j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (r *JobRepository) Update(ctx context.Context, job *domain.Job) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs SET name=$1, type=$2, schedule=$3, config=$4, status=$5,
         notify_channels=$6, condition=$7, last_run_at=$8, next_run_at=$9, updated_at=now()
         WHERE id=$10`,
		job.Name, job.Type, job.Schedule, job.Config, job.Status,
		job.NotifyChannels, job.Condition, job.LastRunAt, job.NextRunAt, job.ID,
	)
	return err
}

func (r *JobRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM jobs WHERE id = $1`, id)
	return err
}
