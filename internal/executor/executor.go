package executor

import (
	"context"
	"encoding/json"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

type Result struct {
	Status domain.ExecStatus
	Output string
}

type Executor interface {
	Execute(ctx context.Context, config json.RawMessage) (*Result, error)
}