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

type TelegramConfig struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

type TelegramNotifier struct {
	client *http.Client
}

func NewTelegramNotifier() *TelegramNotifier {
	return &TelegramNotifier{client: &http.Client{Timeout: 10 * time.Second}}
}

func (t *TelegramNotifier) Type() domain.ChannelType {
	return domain.ChannelTypeTelegram
}

func (t *TelegramNotifier) Send(ctx context.Context, ch *domain.Channel, msg Message) error {
	var cfg TelegramConfig
	if err := json.Unmarshal(ch.Config, &cfg); err != nil {
		return fmt.Errorf("invalid telegram config: %w", err)
	}
	if cfg.BotToken == "" || cfg.ChatID == "" {
		return fmt.Errorf("telegram config missing bot_token or chat_id")
	}

	body, err := json.Marshal(map[string]string{
		"chat_id":    cfg.ChatID,
		"text":       msg.String(),
		"parse_mode": "Markdown",
	})
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram returned HTTP %d", resp.StatusCode)
	}
	return nil
}
