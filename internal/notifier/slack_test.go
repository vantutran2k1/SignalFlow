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

func TestSlack_PostsTextField(t *testing.T) {
	var got atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got.Store(string(body))
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ch := &domain.Channel{Config: jsonRaw(t, SlackConfig{WebhookURL: srv.URL})}
	if err := NewSlackNotifier().Send(context.Background(), ch, Message{
		JobName: "deploy", Status: domain.ExecStatusFailure, Output: "boom",
	}); err != nil {
		t.Fatalf("Send returned err: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(got.Load().(string)), &payload); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if payload["text"] == "" {
		t.Errorf("expected non-empty text field, got %v", payload)
	}
}

func TestSlack_NonOKIsError(t *testing.T) {
	// Slack treats anything other than 200 as failure (note: not just >=400).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	ch := &domain.Channel{Config: jsonRaw(t, SlackConfig{WebhookURL: srv.URL})}
	if err := NewSlackNotifier().Send(context.Background(), ch, Message{}); err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestSlack_MissingWebhookURL(t *testing.T) {
	ch := &domain.Channel{Config: jsonRaw(t, SlackConfig{})}
	if err := NewSlackNotifier().Send(context.Background(), ch, Message{}); err == nil {
		t.Fatal("expected error when webhook_url missing")
	}
}
