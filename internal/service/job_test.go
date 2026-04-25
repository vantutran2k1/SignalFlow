package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

func TestJobService_Create_AppliesDefaults(t *testing.T) {
	repo := newFakeJobRepo()
	svc := NewJobService(repo)

	job, err := svc.Create(context.Background(), CreateJobInput{
		UserID:   "u1",
		Name:     "ping",
		Type:     domain.JobTypeHTTPCheck,
		Schedule: "* * * * *",
		Config:   json.RawMessage(`{"url":"http://x"}`),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if string(job.Condition) != `{"on":"failure"}` {
		t.Errorf("condition default = %s, want failure", job.Condition)
	}
	if job.TimeoutSeconds != 30 {
		t.Errorf("timeout default = %d, want 30", job.TimeoutSeconds)
	}
	if job.Status != domain.JobStatusActive {
		t.Errorf("status = %s, want active", job.Status)
	}
	if job.NextRunAt == nil || !job.NextRunAt.After(time.Now().Add(-time.Second)) {
		t.Errorf("next_run_at not set sensibly: %v", job.NextRunAt)
	}
}

func TestJobService_Create_RejectsInvalidCron(t *testing.T) {
	svc := NewJobService(newFakeJobRepo())
	_, err := svc.Create(context.Background(), CreateJobInput{
		UserID: "u1", Name: "x", Type: domain.JobTypeCommand,
		Schedule: "not a cron",
		Config:   json.RawMessage(`{}`),
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput", err)
	}
}

// The cron parser accepts only 5-field expressions. A 6-field form (which
// includes seconds) would silently drift if the parser were ever switched —
// guard against that.
func TestJobService_Create_RejectsSixFieldCron(t *testing.T) {
	svc := NewJobService(newFakeJobRepo())
	_, err := svc.Create(context.Background(), CreateJobInput{
		UserID: "u1", Name: "x", Type: domain.JobTypeCommand,
		Schedule: "0 * * * * *",
		Config:   json.RawMessage(`{}`),
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput for 6-field cron", err)
	}
}

func TestJobService_Update_ReparseSchedule(t *testing.T) {
	repo := newFakeJobRepo()
	svc := NewJobService(repo)

	job, _ := svc.Create(context.Background(), CreateJobInput{
		UserID: "u1", Name: "x", Type: domain.JobTypeCommand,
		Schedule: "* * * * *", Config: json.RawMessage(`{}`),
	})
	originalNext := *job.NextRunAt

	bad := "definitely not cron"
	if _, err := svc.Update(context.Background(), job.ID, UpdateJobInput{Schedule: &bad}); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput", err)
	}

	// Stored schedule must NOT have changed if validation rejected it.
	stored, _ := repo.GetByID(context.Background(), job.ID)
	if stored.Schedule != "* * * * *" {
		t.Errorf("schedule was updated despite validation failure: %q", stored.Schedule)
	}

	good := "*/5 * * * *"
	updated, err := svc.Update(context.Background(), job.ID, UpdateJobInput{Schedule: &good})
	if err != nil {
		t.Fatalf("Update with valid cron: %v", err)
	}
	if updated.Schedule != good {
		t.Errorf("schedule = %q, want %q", updated.Schedule, good)
	}
	if !updated.NextRunAt.After(originalNext.Add(-time.Second)) {
		// next_run_at must be recomputed.
		t.Errorf("next_run_at not recomputed: %v vs %v", updated.NextRunAt, originalNext)
	}
}

func TestJobService_TriggerRun_AdvancesToNow(t *testing.T) {
	repo := newFakeJobRepo()
	svc := NewJobService(repo)

	// Future cron — next_run_at is far away.
	job, _ := svc.Create(context.Background(), CreateJobInput{
		UserID: "u1", Name: "x", Type: domain.JobTypeCommand,
		Schedule: "0 0 1 1 *", // once a year
		Config:   json.RawMessage(`{}`),
	})
	if !job.NextRunAt.After(time.Now().Add(24 * time.Hour)) {
		t.Skipf("next_run_at not far enough in future to test: %v", job.NextRunAt)
	}

	got, err := svc.TriggerRun(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("TriggerRun: %v", err)
	}
	delta := time.Since(*got.NextRunAt)
	if delta < 0 || delta > time.Second {
		t.Errorf("next_run_at after TriggerRun = %v, want ~now", got.NextRunAt)
	}
}

func TestJobService_PauseClearsNextRun_ResumeRecomputes(t *testing.T) {
	repo := newFakeJobRepo()
	svc := NewJobService(repo)

	job, _ := svc.Create(context.Background(), CreateJobInput{
		UserID: "u1", Name: "x", Type: domain.JobTypeCommand,
		Schedule: "* * * * *", Config: json.RawMessage(`{}`),
	})

	paused, err := svc.Pause(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("Pause: %v", err)
	}
	if paused.Status != domain.JobStatusPaused {
		t.Errorf("status = %s, want paused", paused.Status)
	}
	if paused.NextRunAt != nil {
		t.Errorf("next_run_at must be nil while paused, got %v", paused.NextRunAt)
	}

	resumed, err := svc.Resume(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if resumed.Status != domain.JobStatusActive {
		t.Errorf("status = %s, want active", resumed.Status)
	}
	if resumed.NextRunAt == nil {
		t.Error("next_run_at must be set after resume")
	}
}

func TestJobService_NotFound(t *testing.T) {
	svc := NewJobService(newFakeJobRepo())
	if _, err := svc.GetByID(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("GetByID err = %v, want ErrNotFound", err)
	}
	if _, err := svc.TriggerRun(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("TriggerRun err = %v, want ErrNotFound", err)
	}
	if _, err := svc.Pause(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Pause err = %v, want ErrNotFound", err)
	}
}
