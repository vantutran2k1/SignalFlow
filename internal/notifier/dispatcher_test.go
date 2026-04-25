package notifier

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

// fakeChannelRepo is the minimum surface Dispatcher uses: GetByIDs.
type fakeChannelRepo struct {
	channels map[string]domain.Channel
}

func (f *fakeChannelRepo) Create(context.Context, *domain.Channel) error { return nil }
func (f *fakeChannelRepo) Update(context.Context, *domain.Channel) error { return nil }
func (f *fakeChannelRepo) Delete(context.Context, string) error          { return nil }
func (f *fakeChannelRepo) List(context.Context, string) ([]domain.Channel, error) {
	return nil, nil
}
func (f *fakeChannelRepo) GetByID(_ context.Context, id string) (*domain.Channel, error) {
	if c, ok := f.channels[id]; ok {
		return &c, nil
	}
	return nil, errors.New("not found")
}
func (f *fakeChannelRepo) GetByIDs(_ context.Context, ids []string) ([]domain.Channel, error) {
	out := make([]domain.Channel, 0, len(ids))
	for _, id := range ids {
		if c, ok := f.channels[id]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

type notifRow struct {
	notif  domain.Notification
	status domain.NotificationStatus
	errMsg string
}

type fakeNotifRepo struct {
	mu   sync.Mutex
	rows map[string]*notifRow
}

func newFakeNotifRepo() *fakeNotifRepo {
	return &fakeNotifRepo{rows: make(map[string]*notifRow)}
}
func (f *fakeNotifRepo) Create(_ context.Context, n *domain.Notification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rows[n.ID] = &notifRow{notif: *n, status: n.Status}
	return nil
}
func (f *fakeNotifRepo) ListByExecution(context.Context, string) ([]domain.Notification, error) {
	return nil, nil
}
func (f *fakeNotifRepo) ListRecent(context.Context, int) ([]domain.Notification, error) {
	return nil, nil
}
func (f *fakeNotifRepo) UpdateStatus(_ context.Context, id string, status domain.NotificationStatus, errMsg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r, ok := f.rows[id]; ok {
		r.status = status
		r.errMsg = errMsg
	}
	return nil
}
func (f *fakeNotifRepo) CountSentByUserSince(context.Context, string, time.Time) (int, error) {
	return 0, nil
}

func (f *fakeNotifRepo) snapshot() map[string]notifRow {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make(map[string]notifRow, len(f.rows))
	for k, v := range f.rows {
		out[k] = *v
	}
	return out
}

// recordingNotifier returns a configurable error and counts calls.
type recordingNotifier struct {
	t        domain.ChannelType
	err      error
	calls    atomic.Int32
	released chan struct{} // optional: blocks Send until closed (for Wait test)
}

func (r *recordingNotifier) Type() domain.ChannelType { return r.t }
func (r *recordingNotifier) Send(_ context.Context, _ *domain.Channel, _ Message) error {
	if r.released != nil {
		<-r.released
	}
	r.calls.Add(1)
	return r.err
}

func quietLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func TestDispatcher_NoChannelsIsNoop(t *testing.T) {
	chRepo := &fakeChannelRepo{}
	notifRepo := newFakeNotifRepo()
	d := NewDispatcher(chRepo, notifRepo, quietLogger())

	d.Dispatch(context.Background(), &domain.Job{}, &domain.Execution{})
	d.Wait()
	if got := len(notifRepo.snapshot()); got != 0 {
		t.Errorf("expected 0 notification rows, got %d", got)
	}
}

func TestDispatcher_SkipsDisabledChannels(t *testing.T) {
	chRepo := &fakeChannelRepo{channels: map[string]domain.Channel{
		"on":  {ID: "on", Type: domain.ChannelTypeWebhook, Enabled: true, Config: json.RawMessage(`{}`)},
		"off": {ID: "off", Type: domain.ChannelTypeWebhook, Enabled: false, Config: json.RawMessage(`{}`)},
	}}
	notifRepo := newFakeNotifRepo()
	notif := &recordingNotifier{t: domain.ChannelTypeWebhook}
	d := NewDispatcher(chRepo, notifRepo, quietLogger())
	d.Register(notif)

	d.Dispatch(context.Background(), &domain.Job{NotifyChannels: []string{"on", "off"}}, &domain.Execution{})
	d.Wait()

	if got := notif.calls.Load(); got != 1 {
		t.Errorf("notifier sends = %d, want 1 (disabled must be skipped)", got)
	}
	rows := notifRepo.snapshot()
	if got := len(rows); got != 1 {
		t.Errorf("notification rows = %d, want 1", got)
	}
	for _, r := range rows {
		if r.notif.ChannelID != "on" {
			t.Errorf("row created for wrong channel: %s", r.notif.ChannelID)
		}
	}
}

func TestDispatcher_SuccessMarksSent(t *testing.T) {
	chRepo := &fakeChannelRepo{channels: map[string]domain.Channel{
		"a": {ID: "a", Type: domain.ChannelTypeWebhook, Enabled: true, Config: json.RawMessage(`{}`)},
	}}
	notifRepo := newFakeNotifRepo()
	d := NewDispatcher(chRepo, notifRepo, quietLogger())
	d.Register(&recordingNotifier{t: domain.ChannelTypeWebhook})

	d.Dispatch(context.Background(), &domain.Job{NotifyChannels: []string{"a"}}, &domain.Execution{})
	d.Wait()

	for _, r := range notifRepo.snapshot() {
		if r.status != domain.NotifStatusSent {
			t.Errorf("status = %s, want sent", r.status)
		}
	}
}

func TestDispatcher_SendErrorMarksFailed(t *testing.T) {
	chRepo := &fakeChannelRepo{channels: map[string]domain.Channel{
		"a": {ID: "a", Type: domain.ChannelTypeWebhook, Enabled: true, Config: json.RawMessage(`{}`)},
	}}
	notifRepo := newFakeNotifRepo()
	d := NewDispatcher(chRepo, notifRepo, quietLogger())
	d.Register(&recordingNotifier{t: domain.ChannelTypeWebhook, err: errors.New("boom")})

	d.Dispatch(context.Background(), &domain.Job{NotifyChannels: []string{"a"}}, &domain.Execution{})
	d.Wait()

	for _, r := range notifRepo.snapshot() {
		if r.status != domain.NotifStatusFailed {
			t.Errorf("status = %s, want failed", r.status)
		}
		if r.errMsg != "boom" {
			t.Errorf("errMsg = %q, want boom", r.errMsg)
		}
	}
}

func TestDispatcher_UnknownTypeMarksFailed(t *testing.T) {
	chRepo := &fakeChannelRepo{channels: map[string]domain.Channel{
		"a": {ID: "a", Type: "carrier-pigeon", Enabled: true, Config: json.RawMessage(`{}`)},
	}}
	notifRepo := newFakeNotifRepo()
	d := NewDispatcher(chRepo, notifRepo, quietLogger())
	// No Register call.

	d.Dispatch(context.Background(), &domain.Job{NotifyChannels: []string{"a"}}, &domain.Execution{})
	d.Wait()

	rows := notifRepo.snapshot()
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	for _, r := range rows {
		if r.status != domain.NotifStatusFailed {
			t.Errorf("status = %s, want failed", r.status)
		}
	}
}

func TestDispatcher_WaitBlocksUntilSendReturns(t *testing.T) {
	chRepo := &fakeChannelRepo{channels: map[string]domain.Channel{
		"a": {ID: "a", Type: domain.ChannelTypeWebhook, Enabled: true, Config: json.RawMessage(`{}`)},
	}}
	notifRepo := newFakeNotifRepo()
	gate := make(chan struct{})
	d := NewDispatcher(chRepo, notifRepo, quietLogger())
	d.Register(&recordingNotifier{t: domain.ChannelTypeWebhook, released: gate})

	d.Dispatch(context.Background(), &domain.Job{NotifyChannels: []string{"a"}}, &domain.Execution{})

	waitDone := make(chan struct{})
	go func() {
		d.Wait()
		close(waitDone)
	}()

	select {
	case <-waitDone:
		t.Fatal("Wait returned before Send completed")
	case <-time.After(50 * time.Millisecond):
	}

	close(gate)
	select {
	case <-waitDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Wait did not return after Send unblocked")
	}
}
