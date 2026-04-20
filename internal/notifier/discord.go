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

type DiscordConfig struct {
	WebhookURL string `json:"webhook_url"`
}

type DiscordNotifier struct {
	client *http.Client
}

func NewDiscordNotifier() *DiscordNotifier {
	return &DiscordNotifier{client: &http.Client{Timeout: 10 * time.Second}}
}

func (d *DiscordNotifier) Type() domain.ChannelType {
	return domain.ChannelTypeDiscord
}

func (d *DiscordNotifier) Send(ctx context.Context, ch *domain.Channel, msg Message) error {
	var cfg DiscordConfig
	if err := json.Unmarshal(ch.Config, &cfg); err != nil {
		return fmt.Errorf("invalid discord config: %w", err)
	}
	if cfg.WebhookURL == "" {
		return fmt.Errorf("discord config missing webhook_url")
	}

	body, err := json.Marshal(map[string]string{"content": msg.String()})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Discord returns 204 No Content on success for webhooks
	if resp.StatusCode >= 400 {
		return fmt.Errorf("discord returned HTTP %d", resp.StatusCode)
	}
	return nil
}
