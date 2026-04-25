package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

func TestChannelService_Create_RequiresFields(t *testing.T) {
	svc := NewChannelService(newFakeChannelRepo())
	cases := []struct {
		name string
		in   CreateChannelInput
	}{
		{"no name", CreateChannelInput{Type: domain.ChannelTypeWebhook}},
		{"no type", CreateChannelInput{Name: "x"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := svc.Create(context.Background(), tc.in); !errors.Is(err, ErrInvalidInput) {
				t.Errorf("err = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestChannelService_Update_AppliesPartial(t *testing.T) {
	repo := newFakeChannelRepo()
	svc := NewChannelService(repo)

	ch, err := svc.Create(context.Background(), CreateChannelInput{
		UserID: "u", Name: "Old", Type: domain.ChannelTypeWebhook,
		Config: json.RawMessage(`{"url":"http://a"}`), Enabled: true,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	newName := "New"
	disabled := false
	updated, err := svc.Update(context.Background(), ch.ID, UpdateChannelInput{
		Name:    &newName,
		Enabled: &disabled,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Name != "New" || updated.Enabled {
		t.Errorf("update did not apply: name=%s enabled=%v", updated.Name, updated.Enabled)
	}
	// Config must be preserved when not updated.
	if string(updated.Config) != `{"url":"http://a"}` {
		t.Errorf("config clobbered on partial update: %s", updated.Config)
	}
}

func TestChannelService_NotFound(t *testing.T) {
	svc := NewChannelService(newFakeChannelRepo())
	if _, err := svc.GetByID(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
	name := "x"
	if _, err := svc.Update(context.Background(), "missing", UpdateChannelInput{Name: &name}); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
