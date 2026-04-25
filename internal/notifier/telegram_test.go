package notifier

import (
	"context"
	"testing"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

// Telegram's URL is hardcoded to api.telegram.org, so we can only test config
// validation here (not the full request flow without a network mock layer).

func TestTelegram_MissingFields(t *testing.T) {
	cases := []struct {
		name string
		cfg  TelegramConfig
	}{
		{"both missing", TelegramConfig{}},
		{"missing token", TelegramConfig{ChatID: "1"}},
		{"missing chat_id", TelegramConfig{BotToken: "t"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ch := &domain.Channel{Config: jsonRaw(t, tc.cfg)}
			if err := NewTelegramNotifier().Send(context.Background(), ch, Message{}); err == nil {
				t.Fatalf("expected error for cfg=%+v", tc.cfg)
			}
		})
	}
}
