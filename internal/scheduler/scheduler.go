package scheduler

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	cronlib "github.com/robfig/cron/v3"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
	"github.com/vantutran2k1/SignalFlow/internal/executor"
	"github.com/vantutran2k1/SignalFlow/internal/notifier"
)

const (
	tickInterval      = 1 * time.Second
	defaultJobTimeout = 30 * time.Second
	dispatchTimeout   = 500 * time.Millisecond
	drainGracePeriod  = 15 * time.Second
	// staleRunningAge is the age after which a 'running' execution row is
	// assumed to be from a crashed scheduler and marked as 'error' on startup.
	staleRunningAge = 5 * time.Minute
)

type Scheduler struct {
	jobRepo    domain.JobRepository
	execRepo   domain.ExecutionRepository
	executors  map[domain.JobType]executor.Executor
	dispatcher *notifier.Dispatcher
	parser     cronlib.Parser
	workers    int
	jobChan    chan *domain.JobClaim
	logger     *slog.Logger
}

func New(
	jobRepo domain.JobRepository,
	execRepo domain.ExecutionRepository,
	executors map[domain.JobType]executor.Executor,
	dispatcher *notifier.Dispatcher,
	workers int,
	logger *slog.Logger,
) *Scheduler {
	return &Scheduler{
		jobRepo:    jobRepo,
		execRepo:   execRepo,
		executors:  executors,
		dispatcher: dispatcher,
		parser:     cronlib.NewParser(cronlib.Minute | cronlib.Hour | cronlib.Dom | cronlib.Month | cronlib.Dow),
		workers:    workers,
		jobChan:    make(chan *domain.JobClaim, workers*2),
		logger:     logger,
	}
}

// Run starts the worker pool and tick loop. It blocks until ctx is canceled,
// then drains in-flight jobs up to drainGracePeriod before returning. Execution
// goroutines use an internal context so in-flight work survives dispatch-ctx
// cancellation until the grace period expires.
func (s *Scheduler) Run(ctx context.Context) error {
	s.recoverStale(ctx)

	execCtx, execCancel := context.WithCancel(context.Background())
	defer execCancel()

	var wg sync.WaitGroup
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.worker(execCtx)
		}()
	}
	s.logger.Info("scheduler started", "workers", s.workers)

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

dispatchLoop:
	for {
		select {
		case <-ctx.Done():
			break dispatchLoop
		case <-ticker.C:
			s.tick(ctx)
		}
	}

	s.logger.Info("scheduler draining")
	close(s.jobChan)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("scheduler drained cleanly")
	case <-time.After(drainGracePeriod):
		s.logger.Warn("drain grace period expired, cancelling in-flight jobs")
		execCancel()
		<-done
	}

	if s.dispatcher != nil {
		s.logger.Info("waiting for in-flight notifications")
		s.dispatcher.Wait()
	}
	return nil
}

func (s *Scheduler) recoverStale(ctx context.Context) {
	n, err := s.execRepo.RecoverStaleRunning(ctx, time.Now().Add(-staleRunningAge))
	if err != nil {
		s.logger.Error("failed to recover stale executions", "error", err)
		return
	}
	if n > 0 {
		s.logger.Warn("recovered stale running executions", "count", n)
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	claims, err := s.jobRepo.ClaimDue(ctx, s.workers*2, s.nextRun)
	if err != nil {
		s.logger.Error("failed to claim due jobs", "error", err)
		return
	}

	for i := range claims {
		claim := &claims[i]
		select {
		case s.jobChan <- claim:
		case <-ctx.Done():
			s.markFailed(context.Background(), claim.ExecutionID, "scheduler shutting down")
		case <-time.After(dispatchTimeout):
			s.logger.Warn("worker pool saturated",
				"job_id", claim.Job.ID, "exec_id", claim.ExecutionID)
			s.markFailed(ctx, claim.ExecutionID, "worker pool saturated")
		}
	}
}

func (s *Scheduler) nextRun(schedule string) (time.Time, error) {
	sched, err := s.parser.Parse(schedule)
	if err != nil {
		return time.Time{}, err
	}
	return sched.Next(time.Now()), nil
}

func (s *Scheduler) worker(ctx context.Context) {
	for claim := range s.jobChan {
		s.executeJob(ctx, claim)
	}
}

func (s *Scheduler) executeJob(ctx context.Context, claim *domain.JobClaim) {
	job := &claim.Job
	start := time.Now()

	timeout := time.Duration(job.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = defaultJobTimeout
	}

	exec := &domain.Execution{
		ID:        claim.ExecutionID,
		JobID:     job.ID,
		StartedAt: start,
	}

	e, ok := s.executors[job.Type]
	if !ok {
		exec.Status = domain.ExecStatusError
		exec.Error = "unknown job type: " + string(job.Type)
	} else {
		runCtx, cancel := context.WithTimeout(ctx, timeout)
		result, err := e.Execute(runCtx, job.Config)
		cancel()

		if err != nil {
			exec.Status = domain.ExecStatusError
			exec.Error = err.Error()
		} else {
			exec.Status = result.Status
			exec.Output = result.Output
		}
	}

	finished := time.Now()
	exec.FinishedAt = &finished
	exec.DurationMs = finished.Sub(start).Milliseconds()

	// Use Background for DB writes so terminal state is persisted even if the
	// scheduler is past its grace period and ctx is canceled.
	writeCtx := context.Background()
	if err := s.execRepo.Update(writeCtx, exec); err != nil {
		s.logger.Error("failed to save execution", "exec_id", exec.ID, "error", err)
	}

	job.LastRunAt = &finished
	if err := s.jobRepo.Update(writeCtx, job); err != nil {
		s.logger.Error("failed to update last_run_at", "job_id", job.ID, "error", err)
	}

	s.logger.Info("job executed",
		"job_id", job.ID,
		"job_name", job.Name,
		"status", exec.Status,
		"duration_ms", exec.DurationMs,
	)

	if s.dispatcher != nil && s.shouldNotify(job, exec) {
		s.dispatcher.Dispatch(writeCtx, job, exec)
	}
}

func (s *Scheduler) shouldNotify(job *domain.Job, exec *domain.Execution) bool {
	if len(job.Condition) == 0 {
		return isFailure(exec.Status)
	}
	var cond struct {
		On string `json:"on"`
	}
	if err := json.Unmarshal(job.Condition, &cond); err != nil {
		return false
	}
	switch cond.On {
	case "failure":
		return isFailure(exec.Status)
	case "success":
		return exec.Status == domain.ExecStatusSuccess
	case "always":
		return true
	default:
		return isFailure(exec.Status)
	}
}

func isFailure(s domain.ExecStatus) bool {
	return s == domain.ExecStatusFailure || s == domain.ExecStatusError
}

func (s *Scheduler) markFailed(ctx context.Context, execID, reason string) {
	exec := &domain.Execution{
		ID:         execID,
		Status:     domain.ExecStatusError,
		Error:      reason,
		FinishedAt: new(time.Now()),
	}
	if err := s.execRepo.Update(ctx, exec); err != nil {
		s.logger.Error("failed to mark execution as failed",
			"exec_id", execID, "reason", reason, "error", err)
	}
}
