package notifier

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

const sendTimeout = 15 * time.Second

type Message struct {
	JobName   string
	Status    domain.ExecStatus
	Output    string
	Error     string
	Duration  time.Duration
	Timestamp time.Time
}

func (m Message) String() string {
	if m.Error != "" {
		return fmt.Sprintf("[%s] Job '%s' - %s (took %s)\nOutput: %s",
			m.Status, m.JobName, m.Error, m.Duration, m.Output)
	}
	return fmt.Sprintf("[%s] Job '%s' (took %s)\nOutput: %s",
		m.Status, m.JobName, m.Duration, m.Output)
}

type Notifier interface {
	Type() domain.ChannelType
	Send(ctx context.Context, channel *domain.Channel, msg Message) error
}

type Dispatcher struct {
	notifiers map[domain.ChannelType]Notifier
	chanRepo  domain.ChannelRepository
	notifRepo domain.NotificationRepository
	logger    *slog.Logger
	wg        sync.WaitGroup
}

func NewDispatcher(
	chanRepo domain.ChannelRepository,
	notifRepo domain.NotificationRepository,
	logger *slog.Logger,
) *Dispatcher {
	return &Dispatcher{
		notifiers: make(map[domain.ChannelType]Notifier),
		chanRepo:  chanRepo,
		notifRepo: notifRepo,
		logger:    logger,
	}
}

func (d *Dispatcher) Register(n Notifier) {
	d.notifiers[n.Type()] = n
}

// Dispatch fires notifications to all enabled channels referenced by the job.
// Each channel send runs in its own goroutine; Dispatch returns once the
// notification rows are created in the DB. Call Wait() to block until all
// in-flight sends have finished.
func (d *Dispatcher) Dispatch(ctx context.Context, job *domain.Job, exec *domain.Execution) {
	if len(job.NotifyChannels) == 0 {
		return
	}

	channels, err := d.chanRepo.GetByIDs(ctx, job.NotifyChannels)
	if err != nil {
		d.logger.Error("failed to fetch channels", "error", err)
		return
	}

	msg := executionMessage(job, exec)

	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}

		notif := &domain.Notification{
			ID:          uuid.NewString(),
			ExecutionID: exec.ID,
			ChannelID:   ch.ID,
			Status:      domain.NotifStatusPending,
			Payload:     msg.String(),
			CreatedAt:   time.Now(),
		}
		if err := d.notifRepo.Create(ctx, notif); err != nil {
			d.logger.Error("failed to create notification record",
				"channel_id", ch.ID, "error", err)
			continue
		}

		d.wg.Add(1)
		go func(ch domain.Channel, notif *domain.Notification) {
			defer d.wg.Done()
			d.send(ch, notif, msg)
		}(ch, notif)
	}
}

func (d *Dispatcher) Wait() {
	d.wg.Wait()
}

func (d *Dispatcher) send(ch domain.Channel, notif *domain.Notification, msg Message) {
	// Use Background so terminal status is persisted even if the scheduler's
	// dispatch ctx has been canceled by shutdown.
	writeCtx := context.Background()

	n, ok := d.notifiers[ch.Type]
	if !ok {
		_ = d.notifRepo.UpdateStatus(writeCtx, notif.ID, domain.NotifStatusFailed,
			"no notifier registered for channel type "+string(ch.Type))
		return
	}

	sendCtx, cancel := context.WithTimeout(writeCtx, sendTimeout)
	defer cancel()

	if err := n.Send(sendCtx, &ch, msg); err != nil {
		d.logger.Error("notification failed",
			"channel_id", ch.ID, "channel_name", ch.Name, "error", err)
		if e := d.notifRepo.UpdateStatus(writeCtx, notif.ID, domain.NotifStatusFailed, err.Error()); e != nil {
			d.logger.Error("failed to persist notification failure",
				"notif_id", notif.ID, "error", e)
		}
		return
	}

	if err := d.notifRepo.UpdateStatus(writeCtx, notif.ID, domain.NotifStatusSent, ""); err != nil {
		d.logger.Error("failed to persist notification success",
			"notif_id", notif.ID, "error", err)
	}
}

func executionMessage(job *domain.Job, exec *domain.Execution) Message {
	ts := time.Now()
	if exec.FinishedAt != nil {
		ts = *exec.FinishedAt
	}
	return Message{
		JobName:   job.Name,
		Status:    exec.Status,
		Output:    exec.Output,
		Error:     exec.Error,
		Duration:  time.Duration(exec.DurationMs) * time.Millisecond,
		Timestamp: ts,
	}
}
