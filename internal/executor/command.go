package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

const maxOutputBytes = 10 * 1024

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