package scheduler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
	"github.com/vantutran2k1/SignalFlow/internal/executor"
	"github.com/vantutran2k1/SignalFlow/internal/repository/postgres"
	"github.com/vantutran2k1/SignalFlow/internal/testdb"
)

// ---------- harness ----------

type harness struct {
	pool      *pgxpool.Pool
	jobRepo   *postgres.JobRepository
	execRepo  *postgres.ExecutionRepository
	chanRepo  *postgres.ChannelRepository
	notifRepo *postgres.NotificationRepository
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	pool := testdb.NewPool(t)
	return &harness{
		pool:      pool,
		jobRepo:   postgres.NewJobRepository(pool),
		execRepo:  postgres.NewExecutionRepository(pool),
		chanRepo:  postgres.NewChannelRepository(pool),
		notifRepo: postgres.NewNotificationRepository(pool),
	}
}

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func (h *harness) seedUser(t *testing.T) string {
	t.Helper()
	id := uuid.NewString()
	email := id + "@test.local"
	_, err := h.pool.Exec(context.Background(),
		`INSERT INTO users (id, email, password_hash, name) VALUES ($1, $2, 'hash', '')`,
		id, email)
	if err != nil {
		t.Fatalf("seedUser: %v", err)
	}
	return id
}

func (h *harness) seedDueJob(t *testing.T, userID string) string {
	t.Helper()
	due := time.Now().Add(-1 * time.Minute)
	job := &domain.Job{
		ID:             uuid.NewString(),
		UserID:         userID,
		Name:           "j",
		Type:           domain.JobTypeCommand,
		Schedule:       "* * * * *",
		Config:         json.RawMessage(`{}`),
		Status:         domain.JobStatusActive,
		NotifyChannels: []string{},
		Condition:      json.RawMessage(`{"on":"failure"}`),
		TimeoutSeconds: 5,
		NextRunAt:      &due,
	}
	if err := h.jobRepo.Create(context.Background(), job); err != nil {
		t.Fatalf("seed job: %v", err)
	}
	return job.ID
}

// ---------- fake executor ----------

type fakeExecutor struct {
	calls atomic.Int32
	// behavior to return; defaults to success
	status domain.ExecStatus
	output string
	err    error
	// optional gate to delay execution (for drain tests)
	gate <-chan struct{}
}

func (f *fakeExecutor) Execute(ctx context.Context, _ json.RawMessage) (*executor.Result, error) {
	f.calls.Add(1)
	if f.gate != nil {
		select {
		case <-f.gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	status := f.status
	if status == "" {
		status = domain.ExecStatusSuccess
	}
	return &executor.Result{Status: status, Output: f.output}, nil
}

// ---------- tests ----------

func TestScheduler_TickClaimsAndExecutes(t *testing.T) {
	h := newHarness(t)
	user := h.seedUser(t)
	jobID := h.seedDueJob(t, user)

	exec := &fakeExecutor{status: domain.ExecStatusSuccess, output: "ok"}
	s := New(
		h.jobRepo, h.execRepo,
		map[domain.JobType]executor.Executor{domain.JobTypeCommand: exec},
		nil, // no dispatcher needed
		2, quietLogger(),
	)

	// Start workers manually so we can drive a single tick deterministically.
	execCtx, cancelExec := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.worker(execCtx)
		}()
	}

	s.tick(context.Background())

	// Wait for the job channel to drain by closing it and waiting on workers.
	close(s.jobChan)
	wg.Wait()
	cancelExec()

	if got := exec.calls.Load(); got != 1 {
		t.Fatalf("executor calls = %d, want 1", got)
	}

	// The execution row must now be in 'success' (no longer 'running').
	var status, output string
	err := h.pool.QueryRow(context.Background(),
		`SELECT status, output FROM executions WHERE job_id = $1`, jobID,
	).Scan(&status, &output)
	if err != nil {
		t.Fatal(err)
	}
	if status != "success" {
		t.Errorf("execution status = %q, want success", status)
	}
	if output != "ok" {
		t.Errorf("output = %q, want ok", output)
	}

	// last_run_at should be set on the job.
	var lastRun *time.Time
	if err := h.pool.QueryRow(context.Background(),
		`SELECT last_run_at FROM jobs WHERE id = $1`, jobID).Scan(&lastRun); err != nil {
		t.Fatal(err)
	}
	if lastRun == nil {
		t.Error("last_run_at not set")
	}
}

// recoverStale must mark long-running execution rows as 'error' on startup,
// so a crash during execution doesn't leave rows wedged forever.
func TestScheduler_RecoverStaleOnStart(t *testing.T) {
	h := newHarness(t)
	user := h.seedUser(t)
	jobID := h.seedDueJob(t, user)

	old := time.Now().Add(-30 * time.Minute)
	if _, err := h.pool.Exec(context.Background(),
		`INSERT INTO executions (job_id, status, started_at) VALUES ($1, 'running', $2)`,
		jobID, old); err != nil {
		t.Fatal(err)
	}

	s := New(h.jobRepo, h.execRepo, nil, nil, 1, quietLogger())
	s.recoverStale(context.Background())

	var status string
	if err := h.pool.QueryRow(context.Background(),
		`SELECT status FROM executions WHERE job_id = $1`, jobID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "error" {
		t.Errorf("status = %q, want error", status)
	}
}

// On context cancel, Run drains in-flight executions within the grace window —
// terminal state must still be persisted (we use context.Background() for the
// final write to make this guarantee hold).
func TestScheduler_RunDrainsInFlightOnCancel(t *testing.T) {
	h := newHarness(t)
	user := h.seedUser(t)
	jobID := h.seedDueJob(t, user)

	gate := make(chan struct{})
	exec := &fakeExecutor{status: domain.ExecStatusSuccess, gate: gate}
	s := New(
		h.jobRepo, h.execRepo,
		map[domain.JobType]executor.Executor{domain.JobTypeCommand: exec},
		nil,
		1, quietLogger(),
	)

	ctx, cancel := context.WithCancel(context.Background())
	runDone := make(chan error, 1)
	go func() { runDone <- s.Run(ctx) }()

	// Wait until the job has actually been picked up (executor was called).
	deadline := time.Now().Add(5 * time.Second)
	for exec.calls.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("executor never invoked — scheduler didn't pick the job up")
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel() // request shutdown while job is in flight
	time.Sleep(50 * time.Millisecond)
	close(gate) // let the executor finish

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("Run returned err: %v", err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("Run did not return after drain")
	}

	var status string
	if err := h.pool.QueryRow(context.Background(),
		`SELECT status FROM executions WHERE job_id = $1`, jobID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status == "running" {
		t.Errorf("execution still 'running' after drain — terminal state was not persisted")
	}
}

// shouldNotify drives whether Dispatch is invoked. Verify the JSON condition
// schema is honored: failure default, "always", "success", and unknown values.
func TestScheduler_ShouldNotify(t *testing.T) {
	s := &Scheduler{}
	cases := []struct {
		name string
		cond string
		exec domain.ExecStatus
		want bool
	}{
		{"empty defaults to failure-only, success skipped", ``, domain.ExecStatusSuccess, false},
		{"empty defaults to failure-only, failure fires", ``, domain.ExecStatusFailure, true},
		{"on=always fires on success", `{"on":"always"}`, domain.ExecStatusSuccess, true},
		{"on=success skips failure", `{"on":"success"}`, domain.ExecStatusFailure, false},
		{"on=success fires on success", `{"on":"success"}`, domain.ExecStatusSuccess, true},
		{"unknown 'on' value falls back to failure-only", `{"on":"sometimes"}`, domain.ExecStatusSuccess, false},
		{"malformed JSON disables notifications", `{not json`, domain.ExecStatusFailure, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			job := &domain.Job{Condition: json.RawMessage(tc.cond)}
			exec := &domain.Execution{Status: tc.exec}
			if got := s.shouldNotify(job, exec); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestScheduler_ExecuteJob_UnknownTypeRecorded(t *testing.T) {
	h := newHarness(t)
	user := h.seedUser(t)
	jobID := h.seedDueJob(t, user)

	// Insert a 'running' exec row directly so executeJob has something to update.
	var execID string
	if err := h.pool.QueryRow(context.Background(),
		`INSERT INTO executions (job_id, status, started_at) VALUES ($1, 'running', now()) RETURNING id`,
		jobID).Scan(&execID); err != nil {
		t.Fatal(err)
	}

	s := New(h.jobRepo, h.execRepo,
		map[domain.JobType]executor.Executor{}, // no executors registered
		nil, 1, quietLogger(),
	)

	job, _ := h.jobRepo.GetByID(context.Background(), jobID)
	s.executeJob(context.Background(), &domain.JobClaim{Job: *job, ExecutionID: execID})

	var status, errMsg string
	if err := h.pool.QueryRow(context.Background(),
		`SELECT status, error FROM executions WHERE id = $1`, execID,
	).Scan(&status, &errMsg); err != nil {
		t.Fatal(err)
	}
	if status != "error" {
		t.Errorf("status = %q, want error", status)
	}
	if !strings.Contains(errMsg, "unknown job type") {
		t.Errorf("error = %q, want 'unknown job type ...'", errMsg)
	}
}

func TestScheduler_ExecuteJob_ExecutorErrorRecorded(t *testing.T) {
	h := newHarness(t)
	user := h.seedUser(t)
	jobID := h.seedDueJob(t, user)

	var execID string
	if err := h.pool.QueryRow(context.Background(),
		`INSERT INTO executions (job_id, status, started_at) VALUES ($1, 'running', now()) RETURNING id`,
		jobID).Scan(&execID); err != nil {
		t.Fatal(err)
	}

	s := New(h.jobRepo, h.execRepo,
		map[domain.JobType]executor.Executor{
			domain.JobTypeCommand: &fakeExecutor{err: errors.New("kaboom")},
		},
		nil, 1, quietLogger(),
	)

	job, _ := h.jobRepo.GetByID(context.Background(), jobID)
	s.executeJob(context.Background(), &domain.JobClaim{Job: *job, ExecutionID: execID})

	var status, errMsg string
	if err := h.pool.QueryRow(context.Background(),
		`SELECT status, error FROM executions WHERE id = $1`, execID,
	).Scan(&status, &errMsg); err != nil {
		t.Fatal(err)
	}
	if status != "error" {
		t.Errorf("status = %q, want error", status)
	}
	if errMsg != "kaboom" {
		t.Errorf("error = %q, want 'kaboom'", errMsg)
	}
}
