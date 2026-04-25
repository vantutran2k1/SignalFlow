package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
	"github.com/vantutran2k1/SignalFlow/internal/testdb"
)

func TestExecutionRepository_RecoverStaleRunning(t *testing.T) {
	pool := testdb.NewPool(t)
	repo := NewExecutionRepository(pool)
	ctx := context.Background()

	user := seedUser(t, pool)
	jobID := seedJob(t, pool, user, -1*time.Minute)

	// Insert three running executions: two old, one fresh.
	old := time.Now().Add(-30 * time.Minute)
	fresh := time.Now().Add(-30 * time.Second)
	for _, started := range []time.Time{old, old, fresh} {
		_, err := pool.Exec(ctx,
			`INSERT INTO executions (job_id, status, started_at) VALUES ($1, 'running', $2)`,
			jobID, started)
		if err != nil {
			t.Fatal(err)
		}
	}

	cutoff := time.Now().Add(-5 * time.Minute)
	n, err := repo.RecoverStaleRunning(ctx, cutoff)
	if err != nil {
		t.Fatalf("RecoverStaleRunning: %v", err)
	}
	if n != 2 {
		t.Errorf("recovered = %d, want 2", n)
	}

	// The fresh row must remain in 'running'; the old rows must now be 'error'
	// with a non-null finished_at.
	rows, err := pool.Query(ctx,
		`SELECT status, finished_at FROM executions ORDER BY started_at`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var statuses []string
	for rows.Next() {
		var s string
		var fin *time.Time
		if err := rows.Scan(&s, &fin); err != nil {
			t.Fatal(err)
		}
		if s == string(domain.ExecStatusError) && fin == nil {
			t.Error("recovered execution must have finished_at set")
		}
		statuses = append(statuses, s)
	}
	if statuses[0] != "error" || statuses[1] != "error" || statuses[2] != "running" {
		t.Errorf("statuses = %v, want [error error running]", statuses)
	}
}

func TestExecutionRepository_DeleteOlderThan(t *testing.T) {
	pool := testdb.NewPool(t)
	repo := NewExecutionRepository(pool)
	ctx := context.Background()

	user := seedUser(t, pool)
	jobID := seedJob(t, pool, user, 0)

	for _, started := range []time.Time{
		time.Now().Add(-10 * 24 * time.Hour),
		time.Now().Add(-10 * 24 * time.Hour),
		time.Now().Add(-1 * time.Hour),
	} {
		_, err := pool.Exec(ctx,
			`INSERT INTO executions (job_id, status, started_at, finished_at)
			 VALUES ($1, 'success', $2, $2)`,
			jobID, started)
		if err != nil {
			t.Fatal(err)
		}
	}
	n, err := repo.DeleteOlderThan(ctx, time.Now().Add(-7*24*time.Hour))
	if err != nil {
		t.Fatalf("DeleteOlderThan: %v", err)
	}
	if n != 2 {
		t.Errorf("deleted = %d, want 2", n)
	}
}
