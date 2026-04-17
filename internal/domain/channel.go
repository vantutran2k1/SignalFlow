package domain

import (
	"context"
	"encoding/json"
	"time"
)

type ChannelType string

const (
	ChannelTypeWebhook  ChannelType = "webhook"
	ChannelTypeEmail    ChannelType = "email"
	ChannelTypeSlack    ChannelType = "slack"
	ChannelTypeTelegram ChannelType = "telegram"
	ChannelTypeDiscord  ChannelType = "discord"
)

type Channel struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	Name      string          `json:"name"`
	Type      ChannelType     `json:"type"`
	Config    json.RawMessage `json:"config"`
	Enabled   bool            `json:"enabled"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type ChannelRepository interface {
	Create(ctx context.Context, ch *Channel) error
	GetByID(ctx context.Context, id string) (*Channel, error)
	GetByIDs(ctx context.Context, ids []string) ([]Channel, error)
	List(ctx context.Context, userID string) ([]Channel, error)
	Update(ctx context.Context, ch *Channel) error
	Delete(ctx context.Context, id string) error
}
