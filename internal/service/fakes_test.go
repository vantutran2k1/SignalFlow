package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

// ---------------- User ----------------

type fakeUserRepo struct {
	mu      sync.Mutex
	byID    map[string]domain.User
	byEmail map[string]string // email -> id
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byID: map[string]domain.User{}, byEmail: map[string]string{}}
}

func (f *fakeUserRepo) Create(_ context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, exists := f.byEmail[u.Email]; exists {
		return errors.New("duplicate email")
	}
	now := time.Now()
	u.CreatedAt, u.UpdatedAt = now, now
	f.byID[u.ID] = *u
	f.byEmail[u.Email] = u.ID
	return nil
}
func (f *fakeUserRepo) GetByID(_ context.Context, id string) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &u, nil
}
func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.byEmail[email]
	if !ok {
		return nil, errors.New("not found")
	}
	u := f.byID[id]
	return &u, nil
}
func (f *fakeUserRepo) Update(_ context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byID[u.ID] = *u
	return nil
}
func (f *fakeUserRepo) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if u, ok := f.byID[id]; ok {
		delete(f.byEmail, u.Email)
		delete(f.byID, id)
	}
	return nil
}

// ---------------- Job ----------------

type fakeJobRepo struct {
	mu   sync.Mutex
	jobs map[string]domain.Job
}

func newFakeJobRepo() *fakeJobRepo { return &fakeJobRepo{jobs: map[string]domain.Job{}} }

func (f *fakeJobRepo) Create(_ context.Context, j *domain.Job) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	j.CreatedAt, j.UpdatedAt = now, now
	f.jobs[j.ID] = *j
	return nil
}
func (f *fakeJobRepo) GetByID(_ context.Context, id string) (*domain.Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	j, ok := f.jobs[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &j, nil
}
func (f *fakeJobRepo) List(_ context.Context, userID string, offset, limit int) ([]domain.Job, int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var owned []domain.Job
	for _, j := range f.jobs {
		if j.UserID == userID {
			owned = append(owned, j)
		}
	}
	total := len(owned)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return owned[offset:end], total, nil
}
func (f *fakeJobRepo) ListActive(context.Context) ([]domain.Job, error) { return nil, nil }
func (f *fakeJobRepo) ClaimDue(context.Context, int, domain.NextRunFunc) ([]domain.JobClaim, error) {
	return nil, nil
}
func (f *fakeJobRepo) Update(_ context.Context, j *domain.Job) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.jobs[j.ID]; !ok {
		return errors.New("not found")
	}
	j.UpdatedAt = time.Now()
	f.jobs[j.ID] = *j
	return nil
}
func (f *fakeJobRepo) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.jobs, id)
	return nil
}
func (f *fakeJobRepo) CountByUser(_ context.Context, userID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, j := range f.jobs {
		if j.UserID == userID {
			n++
		}
	}
	return n, nil
}
func (f *fakeJobRepo) CountActiveByUser(_ context.Context, userID string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, j := range f.jobs {
		if j.UserID == userID && j.Status == domain.JobStatusActive {
			n++
		}
	}
	return n, nil
}

// ---------------- Channel ----------------

type fakeChannelRepo struct {
	mu       sync.Mutex
	channels map[string]domain.Channel
}

func newFakeChannelRepo() *fakeChannelRepo {
	return &fakeChannelRepo{channels: map[string]domain.Channel{}}
}

func (f *fakeChannelRepo) Create(_ context.Context, c *domain.Channel) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	c.CreatedAt, c.UpdatedAt = now, now
	f.channels[c.ID] = *c
	return nil
}
func (f *fakeChannelRepo) GetByID(_ context.Context, id string) (*domain.Channel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	c, ok := f.channels[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &c, nil
}
func (f *fakeChannelRepo) GetByIDs(_ context.Context, ids []string) ([]domain.Channel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]domain.Channel, 0, len(ids))
	for _, id := range ids {
		if c, ok := f.channels[id]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}
func (f *fakeChannelRepo) List(_ context.Context, userID string) ([]domain.Channel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []domain.Channel
	for _, c := range f.channels {
		if c.UserID == userID {
			out = append(out, c)
		}
	}
	return out, nil
}
func (f *fakeChannelRepo) Update(_ context.Context, c *domain.Channel) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.channels[c.ID]; !ok {
		return errors.New("not found")
	}
	c.UpdatedAt = time.Now()
	f.channels[c.ID] = *c
	return nil
}
func (f *fakeChannelRepo) Delete(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.channels, id)
	return nil
}

// ---------------- Execution ----------------

type fakeExecRepo struct {
	mu    sync.Mutex
	execs map[string]domain.Execution
}

func newFakeExecRepo() *fakeExecRepo { return &fakeExecRepo{execs: map[string]domain.Execution{}} }

func (f *fakeExecRepo) Create(_ context.Context, e *domain.Execution) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.execs[e.ID] = *e
	return nil
}
func (f *fakeExecRepo) Update(_ context.Context, e *domain.Execution) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.execs[e.ID]; !ok {
		return errors.New("not found")
	}
	f.execs[e.ID] = *e
	return nil
}
func (f *fakeExecRepo) GetByID(_ context.Context, id string) (*domain.Execution, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	e, ok := f.execs[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return &e, nil
}
func (f *fakeExecRepo) ListByJob(context.Context, string, int, int) ([]domain.Execution, int, error) {
	return nil, 0, nil
}
func (f *fakeExecRepo) ListRecent(context.Context, int) ([]domain.Execution, error) {
	return nil, nil
}
func (f *fakeExecRepo) ListRecentByUser(context.Context, string, int) ([]domain.Execution, error) {
	return nil, nil
}
func (f *fakeExecRepo) RecoverStaleRunning(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (f *fakeExecRepo) CountFailuresByUserSince(context.Context, string, time.Time) (int, error) {
	return 0, nil
}
func (f *fakeExecRepo) DeleteOlderThan(context.Context, time.Time) (int64, error) {
	return 0, nil
}

// ---------------- Notification ----------------

type fakeNotifRepo struct {
	mu     sync.Mutex
	notifs map[string]domain.Notification
}

func newFakeNotifRepo() *fakeNotifRepo {
	return &fakeNotifRepo{notifs: map[string]domain.Notification{}}
}

func (f *fakeNotifRepo) Create(_ context.Context, n *domain.Notification) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notifs[n.ID] = *n
	return nil
}
func (f *fakeNotifRepo) ListByExecution(context.Context, string) ([]domain.Notification, error) {
	return nil, nil
}
func (f *fakeNotifRepo) ListRecent(context.Context, int) ([]domain.Notification, error) {
	return nil, nil
}
func (f *fakeNotifRepo) UpdateStatus(_ context.Context, id string, s domain.NotificationStatus, msg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if n, ok := f.notifs[id]; ok {
		n.Status = s
		n.Error = msg
		f.notifs[id] = n
	}
	return nil
}
func (f *fakeNotifRepo) CountSentByUserSince(context.Context, string, time.Time) (int, error) {
	return 0, nil
}
