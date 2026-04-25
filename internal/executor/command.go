package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

const (
	maxOutputBytes = 10 * 1024
	// pipeDrainGrace bounds how long Wait blocks on stdout/stderr pipes after
	// the process group has been killed. Without this, an orphaned grandchild
	// holding the pipe (e.g. `sleep 5` spawned by the shell) would block Wait
	// for the grandchild's full lifetime even after a ctx-cancel.
	pipeDrainGrace = 2 * time.Second
)

type CommandConfig struct {
	Command string `json:"command"`
	Shell   string `json:"shell"`
}

type Command struct{}

func NewCommand() *Command {
	return &Command{}
}

func (c *Command) Execute(ctx context.Context, config json.RawMessage) (*Result, error) {
	var cfg CommandConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	if cfg.Shell == "" {
		cfg.Shell = "/bin/sh"
	}

	cmd := exec.CommandContext(ctx, cfg.Shell, "-c", cfg.Command)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		// Negative pid signals the whole process group, taking out any
		// grandchildren the shell forked.
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	cmd.WaitDelay = pipeDrainGrace
	output, err := cmd.CombinedOutput()

	out := string(output)
	if len(out) > maxOutputBytes {
		out = out[:maxOutputBytes] + "\n... (truncated)"
	}

	if err != nil {
		return &Result{Status: domain.ExecStatusFailure, Output: out}, nil
	}
	return &Result{Status: domain.ExecStatusSuccess, Output: out}, nil
}
