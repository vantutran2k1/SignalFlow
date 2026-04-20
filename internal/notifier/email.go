package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/smtp"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

type EmailConfig struct {
	SMTPHost string `json:"smtp_host"`
	SMTPPort int    `json:"smtp_port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	To       string `json:"to"`
}

type EmailNotifier struct{}

func NewEmailNotifier() *EmailNotifier {
	return &EmailNotifier{}
}

func (e *EmailNotifier) Type() domain.ChannelType {
	return domain.ChannelTypeEmail
}

func (e *EmailNotifier) Send(ctx context.Context, ch *domain.Channel, msg Message) error {
	var cfg EmailConfig
	if err := json.Unmarshal(ch.Config, &cfg); err != nil {
		return fmt.Errorf("invalid email config: %w", err)
	}
	if cfg.SMTPHost == "" || cfg.SMTPPort == 0 || cfg.From == "" || cfg.To == "" {
		return fmt.Errorf("email config missing required fields")
	}

	subject := fmt.Sprintf("[SignalFlow] %s - %s", msg.JobName, msg.Status)
	body := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		cfg.From, cfg.To, subject, msg.String())

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.SMTPHost)
	}

	// net/smtp.SendMail doesn't support context; ctx deadline applies indirectly
	// through the outer send timeout + the SMTP server's own timeouts.
	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, auth, cfg.From, []string{cfg.To}, []byte(body))
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
