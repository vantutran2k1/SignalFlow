package notifier

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

func TestDiscord_Accepts204(t *testing.T) {
	// Discord webhooks return 204 No Content on success. Make sure we don't
	// treat that as an error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	ch := &domain.Channel{Config: jsonRaw(t, DiscordConfig{WebhookURL: srv.URL})}
	if err := NewDiscordNotifier().Send(context.Background(), ch, Message{}); err != nil {
		t.Fatalf("204 should be accepted, got err: %v", err)
	}
}

func TestDiscord_PostsContentField(t *testing.T) {
	var got atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got.Store(string(body))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	ch := &domain.Channel{Config: jsonRaw(t, DiscordConfig{WebhookURL: srv.URL})}
	if err := NewDiscordNotifier().Send(context.Background(), ch, Message{JobName: "x"}); err != nil {
		t.Fatalf("Send returned err: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(got.Load().(string)), &payload); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if payload["content"] == "" {
		t.Errorf("expected non-empty content field, got %v", payload)
	}
}

func TestDiscord_4xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	ch := &domain.Channel{Config: jsonRaw(t, DiscordConfig{WebhookURL: srv.URL})}
	if err := NewDiscordNotifier().Send(context.Background(), ch, Message{}); err == nil {
		t.Fatal("expected error for 400")
	}
}

func TestDiscord_MissingWebhookURL(t *testing.T) {
	ch := &domain.Channel{Config: jsonRaw(t, DiscordConfig{})}
	if err := NewDiscordNotifier().Send(context.Background(), ch, Message{}); err == nil {
		t.Fatal("expected error when webhook_url missing")
	}
}
