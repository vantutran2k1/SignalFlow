package executor

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

func TestCommand_Success(t *testing.T) {
	cfg, _ := json.Marshal(CommandConfig{Command: "echo hello"})
	res, err := NewCommand().Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute returned err: %v", err)
	}
	if res.Status != domain.ExecStatusSuccess {
		t.Errorf("status = %q, want success", res.Status)
	}
	if !strings.Contains(res.Output, "hello") {
		t.Errorf("output = %q, expected to contain 'hello'", res.Output)
	}
}

func TestCommand_NonZeroExit(t *testing.T) {
	cfg, _ := json.Marshal(CommandConfig{Command: "exit 1"})
	res, err := NewCommand().Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute returned err: %v", err)
	}
	if res.Status != domain.ExecStatusFailure {
		t.Errorf("status = %q, want failure", res.Status)
	}
}

func TestCommand_TimeoutCancels(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	cfg, _ := json.Marshal(CommandConfig{Command: "sleep 5"})
	start := time.Now()
	res, err := NewCommand().Execute(ctx, cfg)
	if err != nil {
		t.Fatalf("Execute returned err: %v", err)
	}
	if res.Status != domain.ExecStatusFailure {
		t.Errorf("status = %q, want failure on timeout", res.Status)
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("ctx cancel did not kill child: elapsed=%v", elapsed)
	}
}

func TestCommand_OutputTruncated(t *testing.T) {
	// Produce more than maxOutputBytes (10KB) of output.
	cfg, _ := json.Marshal(CommandConfig{
		Command: "yes | head -c 20000",
	})
	res, err := NewCommand().Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute returned err: %v", err)
	}
	if !strings.HasSuffix(res.Output, "(truncated)") {
		t.Errorf("expected truncation marker, got tail: %q", tail(res.Output, 30))
	}
	if len(res.Output) > maxOutputBytes+64 {
		t.Errorf("output exceeded truncation cap: len=%d", len(res.Output))
	}
}

func TestCommand_DefaultShell(t *testing.T) {
	// Empty shell falls back to /bin/sh; pipes (|) only work if a shell is
	// invoked, so this also verifies we run via the shell.
	cfg, _ := json.Marshal(CommandConfig{Command: "echo a | tr a-z A-Z"})
	res, err := NewCommand().Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Execute returned err: %v", err)
	}
	if !strings.Contains(res.Output, "A") {
		t.Errorf("output = %q, expected uppercase 'A'", res.Output)
	}
}

func TestCommand_BadConfig(t *testing.T) {
	_, err := NewCommand().Execute(context.Background(), json.RawMessage(`{`))
	if err == nil {
		t.Fatal("expected error for invalid config JSON")
	}
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
