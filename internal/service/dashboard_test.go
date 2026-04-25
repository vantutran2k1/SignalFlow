package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

func TestDashboardService_StatsCounts(t *testing.T) {
	jobs := newFakeJobRepo()
	execs := newFakeExecRepo()
	notifs := newFakeNotifRepo()

	jobSvc := NewJobService(jobs)

	// Two active jobs, one paused — for user u1.
	for _, name := range []string{"a", "b"} {
		if _, err := jobSvc.Create(context.Background(), CreateJobInput{
			UserID: "u1", Name: name, Type: domain.JobTypeCommand,
			Schedule: "* * * * *", Config: json.RawMessage(`{}`),
		}); err != nil {
			t.Fatal(err)
		}
	}
	paused, _ := jobSvc.Create(context.Background(), CreateJobInput{
		UserID: "u1", Name: "c", Type: domain.JobTypeCommand,
		Schedule: "* * * * *", Config: json.RawMessage(`{}`),
	})
	if _, err := jobSvc.Pause(context.Background(), paused.ID); err != nil {
		t.Fatal(err)
	}

	// One job for a different user — must not leak into u1's counts.
	if _, err := jobSvc.Create(context.Background(), CreateJobInput{
		UserID: "u2", Name: "other", Type: domain.JobTypeCommand,
		Schedule: "* * * * *", Config: json.RawMessage(`{}`),
	}); err != nil {
		t.Fatal(err)
	}

	dash := NewDashboardService(jobs, execs, notifs)
	stats, err := dash.Stats(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if stats.TotalJobs != 3 {
		t.Errorf("TotalJobs = %d, want 3", stats.TotalJobs)
	}
	if stats.ActiveJobs != 2 {
		t.Errorf("ActiveJobs = %d, want 2", stats.ActiveJobs)
	}
}
