package postgres

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
	"github.com/vantutran2k1/SignalFlow/internal/testdb"
)

// fixedNext returns a NextRunFunc that always returns the same time, so test
// assertions don't drift with real cron semantics.
func fixedNext(at time.Time) domain.NextRunFunc {
	return func(string) (time.Time, error) { return at, nil }
}

func TestJobRepository_ClaimDue_BasicAtomicity(t *testing.T) {
	pool := testdb.NewPool(t)
	repo := NewJobRepository(pool)
	ctx := context.Background()

	user := seedUser(t, pool)
	dueA := seedJob(t, pool, user, -1*time.Minute)
	dueB := seedJob(t, pool, user, -1*time.Minute)
	notDue := seedJob(t, pool, user, 1*time.Hour)

	advanceTo := time.Now().Add(1 * time.Hour)
	claims, err := repo.ClaimDue(ctx, 10, fixedNext(advanceTo))
	if err != nil {
		t.Fatalf("ClaimDue: %v", err)
	}
	if len(claims) != 2 {
		t.Fatalf("claims = %d, want 2 (only due jobs)", len(claims))
	}

	claimed := map[string]bool{}
	for _, c := range claims {
		claimed[c.Job.ID] = true
		if c.ExecutionID == "" {
			t.Errorf("claim missing exec id: %+v", c)
		}
	}
	if !claimed[dueA] || !claimed[dueB] || claimed[notDue] {
		t.Errorf("claimed set wrong: %v", claimed)
	}

	// Each claimed job must have a 'running' execution row inserted.
	var nRunning int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM executions WHERE status = 'running'`).Scan(&nRunning); err != nil {
		t.Fatalf("count running: %v", err)
	}
	if nRunning != 2 {
		t.Errorf("running execution rows = %d, want 2", nRunning)
	}

	// next_run_at must have been advanced for both claimed jobs (so the next
	// tick doesn't re-claim them), but NOT for the unclaimed one.
	for _, id := range []string{dueA, dueB} {
		j, err := repo.GetByID(ctx, id)
		if err != nil {
			t.Fatalf("GetByID(%s): %v", id, err)
		}
		// Postgres timestamptz has microsecond precision; Go time has nanos.
		// Compare with millisecond tolerance — granular enough to catch
		// "didn't advance at all" while ignoring round-trip truncation.
		if j.NextRunAt == nil || j.NextRunAt.Sub(advanceTo).Abs() > time.Millisecond {
			t.Errorf("job %s next_run_at = %v, want ~%v", id, j.NextRunAt, advanceTo)
		}
	}
}

// The crown jewel of the scheduler: two concurrent ClaimDue transactions
// must never claim the same row. SKIP LOCKED makes this safe for multi-replica
// deploys without a leader election. If this test ever fails, the scheduler
// can dispatch the same job twice and execution duplicates will appear.
func TestJobRepository_ClaimDue_ConcurrentReplicasDoNotOverlap(t *testing.T) {
	pool := testdb.NewPool(t)
	repo := NewJobRepository(pool)
	ctx := context.Background()

	user := seedUser(t, pool)
	const total = 6
	want := map[string]bool{}
	for i := 0; i < total; i++ {
		id := seedJob(t, pool, user, -1*time.Minute)
		want[id] = true
	}

	const replicas = 3
	advanceTo := time.Now().Add(1 * time.Hour)
	var (
		mu    sync.Mutex
		seen  = map[string]int{}
		errs  []error
		wg    sync.WaitGroup
		ready sync.WaitGroup
		gate  = make(chan struct{})
	)
	ready.Add(replicas)
	wg.Add(replicas)
	for r := 0; r < replicas; r++ {
		go func() {
			defer wg.Done()
			ready.Done()
			<-gate // start all replicas as close to simultaneously as possible
			claims, err := repo.ClaimDue(ctx, total, fixedNext(advanceTo))
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs = append(errs, err)
				return
			}
			for _, c := range claims {
				seen[c.Job.ID]++
			}
		}()
	}
	ready.Wait()
	close(gate)
	wg.Wait()

	if len(errs) > 0 {
		t.Fatalf("ClaimDue errors: %v", errs)
	}
	if len(seen) != total {
		t.Errorf("distinct claimed jobs = %d, want %d (some rows missed?)", len(seen), total)
	}
	for id, n := range seen {
		if n != 1 {
			t.Errorf("job %s claimed %d times — SKIP LOCKED broken!", id, n)
		}
		if !want[id] {
			t.Errorf("claimed unknown job id: %s", id)
		}
	}
}

func TestJobRepository_ClaimDue_SkipsPaused(t *testing.T) {
	pool := testdb.NewPool(t)
	repo := NewJobRepository(pool)
	ctx := context.Background()

	user := seedUser(t, pool)
	dueActive := seedJob(t, pool, user, -1*time.Minute)
	duePaused := seedJob(t, pool, user, -1*time.Minute)
	if _, err := pool.Exec(ctx,
		`UPDATE jobs SET status = 'paused' WHERE id = $1`, duePaused); err != nil {
		t.Fatal(err)
	}

	claims, err := repo.ClaimDue(ctx, 10, fixedNext(time.Now().Add(1*time.Hour)))
	if err != nil {
		t.Fatalf("ClaimDue: %v", err)
	}
	if len(claims) != 1 || claims[0].Job.ID != dueActive {
		t.Errorf("claims = %+v, want only the active job %s", claims, dueActive)
	}
}

// If next() returns an error (corrupted schedule), the row should be skipped,
// not block the whole claim transaction.
func TestJobRepository_ClaimDue_BadCronSkipped(t *testing.T) {
	pool := testdb.NewPool(t)
	repo := NewJobRepository(pool)
	ctx := context.Background()

	user := seedUser(t, pool)
	jobID := seedJob(t, pool, user, -1*time.Minute)

	failingNext := func(string) (time.Time, error) { return time.Time{}, errExpected }
	claims, err := repo.ClaimDue(ctx, 10, failingNext)
	if err != nil {
		t.Fatalf("ClaimDue: %v", err)
	}
	if len(claims) != 0 {
		t.Errorf("claims = %d, want 0 when nextRun fails", len(claims))
	}
	// Job remains in DB with original next_run_at (we'll re-attempt next tick).
	j, _ := repo.GetByID(ctx, jobID)
	if j.NextRunAt == nil || j.NextRunAt.After(time.Now()) {
		t.Errorf("next_run_at advanced despite cron failure: %v", j.NextRunAt)
	}
}

var errExpected = &repoErr{"intentional test failure"}

type repoErr struct{ msg string }

func (e *repoErr) Error() string { return e.msg }
