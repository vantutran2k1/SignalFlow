package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

type HTTPCheckConfig struct {
	URL            string `json:"url"`
	ExpectedStatus int    `json:"expected_status"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

type HTTPCheck struct {
	client *http.Client
}

func NewHTTPCheck() *HTTPCheck {
	return &HTTPCheck{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (h *HTTPCheck) Execute(ctx context.Context, config json.RawMessage) (*Result, error) {
	var cfg HTTPCheckConfig
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	if cfg.ExpectedStatus == 0 {
		cfg.ExpectedStatus = 200
	}
	if cfg.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(cfg.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.URL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return &Result{
			Status: domain.ExecStatusFailure,
			Output: fmt.Sprintf("request failed: %s", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == cfg.ExpectedStatus {
		return &Result{
			Status: domain.ExecStatusSuccess,
			Output: fmt.Sprintf("HTTP %d (expected %d)", resp.StatusCode, cfg.ExpectedStatus),
		}, nil
	}

	return &Result{
		Status: domain.ExecStatusFailure,
		Output: fmt.Sprintf("HTTP %d (expected %d)", resp.StatusCode, cfg.ExpectedStatus),
	}, nil
}
