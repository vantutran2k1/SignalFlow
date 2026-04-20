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

type SlackConfig struct {
	WebhookURL string `json:"webhook_url"`
}

type SlackNotifier struct {
	client *http.Client
}

func NewSlackNotifier() *SlackNotifier {
	return &SlackNotifier{client: &http.Client{Timeout: 10 * time.Second}}
}

func (s *SlackNotifier) Type() domain.ChannelType {
	return domain.ChannelTypeSlack
}

func (s *SlackNotifier) Send(ctx context.Context, ch *domain.Channel, msg Message) error {
	var cfg SlackConfig
	if err := json.Unmarshal(ch.Config, &cfg); err != nil {
		return fmt.Errorf("invalid slack config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return fmt.Errorf("slack config missing webhook_url")
	}

	body, err := json.Marshal(map[string]string{"text": msg.String()})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned HTTP %d", resp.StatusCode)
	}
	return nil
}
