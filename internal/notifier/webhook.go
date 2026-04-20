package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

type WebhookConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type WebhookNotifier struct {
	client *http.Client
}

func NewWebhookNotifier() *WebhookNotifier {
	return &WebhookNotifier{client: &http.Client{Timeout: 10 * time.Second}}
}

func (w *WebhookNotifier) Type() domain.ChannelType {
	return domain.ChannelTypeWebhook
}

func (w *WebhookNotifier) Send(ctx context.Context, ch *domain.Channel, msg Message) error {
	var cfg WebhookConfig
	if err := json.Unmarshal(ch.Config, &cfg); err != nil {
		return fmt.Errorf("invalid webhook config: %w", err)
	}
	if cfg.URL == "" {
		return fmt.Errorf("webhook config missing url")
	}

	body, err := json.Marshal(map[string]any{
		"job_name":  msg.JobName,
		"status":    msg.Status,
		"output":    msg.Output,
		"error":     msg.Error,
		"duration":  msg.Duration.String(),
		"timestamp": msg.Timestamp,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned HTTP %d", resp.StatusCode)
	}
	return nil
}
