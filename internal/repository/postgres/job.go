package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

type JobRepository struct {
	pool *pgxpool.Pool
}

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{pool: pool}
}

const jobColumns = `id, user_id, name, type, schedule, config, status, notify_channels,
    condition, timeout_seconds, last_run_at, next_run_at, created_at, updated_at`

func scanJob(row pgx.Row, j *domain.Job) error {
	return row.Scan(
		&j.ID, &j.UserID, &j.Name, &j.Type, &j.Schedule,
		&j.Config, &j.Status, &j.NotifyChannels, &j.Condition, &j.TimeoutSeconds,
		&j.LastRunAt, &j.NextRunAt, &j.CreatedAt, &j.UpdatedAt,
	)
}

func (r *JobRepository) Create(ctx context.Context, job *domain.Job) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO jobs (id, user_id, name, type, schedule, config, status, notify_channels,
            condition, timeout_seconds, next_run_at, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now(), now())`,
		job.ID, job.UserID, job.Name, job.Type, job.Schedule,
		job.Config, job.Status, job.NotifyChannels, job.Condition, job.TimeoutSeconds, job.NextRunAt,
	)
	return err
}

func (r *JobRepository) GetByID(ctx context.Context, id string) (*domain.Job, error) {
	job := &domain.Job{}
	err := scanJob(r.pool.QueryRow(ctx,
		`SELECT `+jobColumns+` FROM jobs WHERE id = $1`, id), job)
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
		`SELECT `+jobColumns+` FROM jobs WHERE user_id = $1
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
		if err := scanJob(rows, &j); err != nil {
			return nil, 0, err
		}
		jobs = append(jobs, j)
	}
	return jobs, total, nil
}

func (r *JobRepository) ListActive(ctx context.Context) ([]domain.Job, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+jobColumns+` FROM jobs WHERE status = 'active'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []domain.Job
	for rows.Next() {
		var j domain.Job
		if err := scanJob(rows, &j); err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

func (r *JobRepository) ClaimDue(ctx context.Context, limit int, next domain.NextRunFunc) ([]domain.JobClaim, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx,
		`SELECT `+jobColumns+` FROM jobs
         WHERE status = 'active' AND next_run_at <= now()
         ORDER BY next_run_at ASC
         LIMIT $1
         FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, err
	}

	var jobs []domain.Job
	for rows.Next() {
		var j domain.Job
		if err := scanJob(rows, &j); err != nil {
			rows.Close()
			return nil, err
		}
		jobs = append(jobs, j)
	}
	rows.Close()

	now := time.Now()
	claims := make([]domain.JobClaim, 0, len(jobs))
	for i := range jobs {
		job := &jobs[i]

		nextAt, err := next(job.Schedule)
		if err != nil {
			// Leave next_run_at alone; job will be retried next tick and surface
			// the same error. Skip rather than fail the whole claim.
			continue
		}
		job.NextRunAt = &nextAt

		if _, err := tx.Exec(ctx,
			`UPDATE jobs SET next_run_at = $1, updated_at = now() WHERE id = $2`,
			nextAt, job.ID); err != nil {
			return nil, err
		}

		var execID string
		if err := tx.QueryRow(ctx,
			`INSERT INTO executions (job_id, status, started_at)
             VALUES ($1, 'running', $2) RETURNING id`,
			job.ID, now).Scan(&execID); err != nil {
			return nil, err
		}

		claims = append(claims, domain.JobClaim{Job: *job, ExecutionID: execID})
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return claims, nil
}

func (r *JobRepository) Update(ctx context.Context, job *domain.Job) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE jobs SET name=$1, type=$2, schedule=$3, config=$4, status=$5,
         notify_channels=$6, condition=$7, timeout_seconds=$8, last_run_at=$9, next_run_at=$10, updated_at=now()
         WHERE id=$11`,
		job.Name, job.Type, job.Schedule, job.Config, job.Status,
		job.NotifyChannels, job.Condition, job.TimeoutSeconds, job.LastRunAt, job.NextRunAt, job.ID,
	)
	return err
}

func (r *JobRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM jobs WHERE id = $1`, id)
	return err
}