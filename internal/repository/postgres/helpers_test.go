package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

// seedUser inserts a user directly via SQL and returns the id. Email is
// generated per-call so concurrent test packages can't collide on the unique
// constraint when `go test ./...` runs them in parallel.
func seedUser(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	id := uuid.NewString()
	email := id + "@test.local"
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, password_hash, name) VALUES ($1, $2, 'hash', '')`,
		id, email)
	if err != nil {
		t.Fatalf("seedUser: %v", err)
	}
	return id
}

// seedJob inserts an active job whose next_run_at is `dueIn` from now (negative
// = already due). Returns the job ID.
func seedJob(t *testing.T, pool *pgxpool.Pool, userID string, dueIn time.Duration) string {
	t.Helper()
	repo := NewJobRepository(pool)
	next := time.Now().Add(dueIn)
	job := &domain.Job{
		ID:             uuid.NewString(),
		UserID:         userID,
		Name:           "j-" + uuid.NewString()[:8],
		Type:           domain.JobTypeCommand,
		Schedule:       "* * * * *",
		Config:         json.RawMessage(`{"command":"echo hi"}`),
		Status:         domain.JobStatusActive,
		NotifyChannels: []string{},
		Condition:      json.RawMessage(`{"on":"failure"}`),
		TimeoutSeconds: 10,
		NextRunAt:      &next,
	}
	if err := repo.Create(context.Background(), job); err != nil {
		t.Fatalf("seedJob Create: %v", err)
	}
	return job.ID
}
