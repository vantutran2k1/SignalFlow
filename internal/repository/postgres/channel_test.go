package postgres

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
	"github.com/vantutran2k1/SignalFlow/internal/testdb"
)

func TestChannelRepository_GetByIDs(t *testing.T) {
	pool := testdb.NewPool(t)
	repo := NewChannelRepository(pool)
	ctx := context.Background()

	user := seedUser(t, pool)

	mk := func(name string, enabled bool) string {
		ch := &domain.Channel{
			ID:      uuid.NewString(),
			UserID:  user,
			Name:    name,
			Type:    domain.ChannelTypeWebhook,
			Config:  json.RawMessage(`{"url":"http://x"}`),
			Enabled: enabled,
		}
		if err := repo.Create(ctx, ch); err != nil {
			t.Fatalf("Create %s: %v", name, err)
		}
		return ch.ID
	}
	a := mk("a", true)
	b := mk("b", false)
	mk("c", true) // unrelated

	got, err := repo.GetByIDs(ctx, []string{a, b})
	if err != nil {
		t.Fatalf("GetByIDs: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d channels, want 2", len(got))
	}

	// Empty slice must short-circuit cleanly without a SQL error.
	if got, err := repo.GetByIDs(ctx, nil); err != nil || got != nil {
		t.Errorf("GetByIDs(nil) = %v, %v; want nil, nil", got, err)
	}
}
