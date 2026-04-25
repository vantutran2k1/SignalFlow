package executor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

func TestHTTPCheck_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg, _ := json.Marshal(HTTPCheckConfig{URL: srv.URL, ExpectedStatus: 200})
	res, err := NewHTTPCheck().Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute returned err: %v", err)
	}
	if res.Status != domain.ExecStatusSuccess {
		t.Errorf("status = %q, want success", res.Status)
	}
}

func TestHTTPCheck_StatusMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg, _ := json.Marshal(HTTPCheckConfig{URL: srv.URL, ExpectedStatus: 200})
	res, err := NewHTTPCheck().Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute returned err: %v", err)
	}
	if res.Status != domain.ExecStatusFailure {
		t.Errorf("status = %q, want failure", res.Status)
	}
	if !strings.Contains(res.Output, "HTTP 500") {
		t.Errorf("output = %q, expected to mention HTTP 500", res.Output)
	}
}

// HTTPCheck reports network errors as failure, not error — operationally these
// are indistinguishable from the target being down, which is what an HTTP
// check is supposed to detect.
func TestHTTPCheck_NetworkError(t *testing.T) {
	cfg, _ := json.Marshal(HTTPCheckConfig{URL: "http://127.0.0.1:1", ExpectedStatus: 200})
	res, err := NewHTTPCheck().Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute returned err: %v", err)
	}
	if res.Status != domain.ExecStatusFailure {
		t.Errorf("status = %q, want failure", res.Status)
	}
}

func TestHTTPCheck_DefaultsExpected200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Omit ExpectedStatus — should default to 200.
	cfg, _ := json.Marshal(HTTPCheckConfig{URL: srv.URL})
	res, err := NewHTTPCheck().Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute returned err: %v", err)
	}
	if res.Status != domain.ExecStatusSuccess {
		t.Errorf("status = %q, want success", res.Status)
	}
}

func TestHTTPCheck_CtxCancelStopsRequest(t *testing.T) {
	// Server hangs longer than the test cares to wait.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cfg, _ := json.Marshal(HTTPCheckConfig{URL: srv.URL, ExpectedStatus: 200})
	start := time.Now()
	res, err := NewHTTPCheck().Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Execute returned err: %v", err)
	}
	if res.Status != domain.ExecStatusFailure {
		t.Errorf("status = %q, want failure on timeout", res.Status)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("ctx cancel did not unblock request: elapsed=%v", elapsed)
	}
}

func TestHTTPCheck_BadConfig(t *testing.T) {
	_, err := NewHTTPCheck().Execute(context.Background(), json.RawMessage(`not json`))
	if err == nil {
		t.Fatal("expected error for invalid config JSON")
	}
}
