package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

type ExecutionRepository struct {
	pool *pgxpool.Pool
}

func NewExecutionRepository(pool *pgxpool.Pool) *ExecutionRepository {
	return &ExecutionRepository{pool: pool}
}

func (r *ExecutionRepository) Create(ctx context.Context, exec *domain.Execution) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO executions (id, job_id, status, output, error, duration_ms, started_at, finished_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		exec.ID, exec.JobID, exec.Status, exec.Output, exec.Error,
		exec.DurationMs, exec.StartedAt, exec.FinishedAt,
	)
	return err
}

func (r *ExecutionRepository) Update(ctx context.Context, exec *domain.Execution) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE executions SET status=$1, output=$2, error=$3, duration_ms=$4, finished_at=$5
         WHERE id=$6`,
		exec.Status, exec.Output, exec.Error, exec.DurationMs, exec.FinishedAt, exec.ID,
	)
	return err
}

func (r *ExecutionRepository) GetByID(ctx context.Context, id string) (*domain.Execution, error) {
	exec := &domain.Execution{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, job_id, status, output, error, duration_ms, started_at, finished_at
         FROM executions WHERE id = $1`, id,
	).Scan(
		&exec.ID, &exec.JobID, &exec.Status, &exec.Output, &exec.Error,
		&exec.DurationMs, &exec.StartedAt, &exec.FinishedAt,
	)
	if err != nil {
		return nil, err
	}
	return exec, nil
}

func (r *ExecutionRepository) ListByJob(ctx context.Context, jobID string, offset, limit int) ([]domain.Execution, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM executions WHERE job_id = $1`, jobID,
	).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, job_id, status, output, error, duration_ms, started_at, finished_at
         FROM executions WHERE job_id = $1
         ORDER BY started_at DESC LIMIT $2 OFFSET $3`,
		jobID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var execs []domain.Execution
	for rows.Next() {
		var e domain.Execution
		if err := rows.Scan(
			&e.ID, &e.JobID, &e.Status, &e.Output, &e.Error,
			&e.DurationMs, &e.StartedAt, &e.FinishedAt,
		); err != nil {
			return nil, 0, err
		}
		execs = append(execs, e)
	}
	return execs, total, nil
}

func (r *ExecutionRepository) ListRecent(ctx context.Context, limit int) ([]domain.Execution, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, job_id, status, output, error, duration_ms, started_at, finished_at
         FROM executions ORDER BY started_at DESC LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var execs []domain.Execution
	for rows.Next() {
		var e domain.Execution
		if err := rows.Scan(
			&e.ID, &e.JobID, &e.Status, &e.Output, &e.Error,
			&e.DurationMs, &e.StartedAt, &e.FinishedAt,
		); err != nil {
			return nil, err
		}
		execs = append(execs, e)
	}
	return execs, nil
}

func (r *ExecutionRepository) RecoverStaleRunning(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx,
		`UPDATE executions
         SET status = 'error', error = 'scheduler crashed or timed out', finished_at = now()
         WHERE status = 'running' AND started_at < $1`,
		cutoff,
	)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *ExecutionRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM executions WHERE started_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}