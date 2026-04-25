package notifier

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

func TestWebhook_PostsExpectedPayload(t *testing.T) {
	var (
		gotBody    atomic.Value
		gotHeaders atomic.Value
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody.Store(string(body))
		gotHeaders.Store(r.Header.Clone())
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	ch := &domain.Channel{
		Type: domain.ChannelTypeWebhook,
		Config: jsonRaw(t, WebhookConfig{
			URL:     srv.URL,
			Headers: map[string]string{"X-Signature": "abc"},
		}),
	}
	msg := Message{
		JobName:   "deploy-check",
		Status:    domain.ExecStatusFailure,
		Output:    "boom",
		Duration:  150 * time.Millisecond,
		Timestamp: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}

	if err := NewWebhookNotifier().Send(context.Background(), ch, msg); err != nil {
		t.Fatalf("Send returned err: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(gotBody.Load().(string)), &payload); err != nil {
		t.Fatalf("body not JSON: %v", err)
	}
	if payload["job_name"] != "deploy-check" {
		t.Errorf("job_name = %v, want deploy-check", payload["job_name"])
	}
	if payload["status"] != "failure" {
		t.Errorf("status = %v, want failure", payload["status"])
	}

	hdr := gotHeaders.Load().(http.Header)
	if got := hdr.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := hdr.Get("X-Signature"); got != "abc" {
		t.Errorf("X-Signature = %q, want abc — custom headers must propagate", got)
	}
}

func TestWebhook_4xxIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	ch := &domain.Channel{Config: jsonRaw(t, WebhookConfig{URL: srv.URL})}
	if err := NewWebhookNotifier().Send(context.Background(), ch, Message{}); err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestWebhook_MissingURL(t *testing.T) {
	ch := &domain.Channel{Config: jsonRaw(t, WebhookConfig{})}
	if err := NewWebhookNotifier().Send(context.Background(), ch, Message{}); err == nil {
		t.Fatal("expected error when URL missing")
	}
}

func TestWebhook_BadConfig(t *testing.T) {
	ch := &domain.Channel{Config: json.RawMessage(`not json`)}
	if err := NewWebhookNotifier().Send(context.Background(), ch, Message{}); err == nil {
		t.Fatal("expected error for invalid config JSON")
	}
}

func jsonRaw(t *testing.T, v any) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return b
}
