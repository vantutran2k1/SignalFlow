package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

type ChannelService struct {
	repo domain.ChannelRepository
}

func NewChannelService(repo domain.ChannelRepository) *ChannelService {
	return &ChannelService{repo: repo}
}

type CreateChannelInput struct {
	UserID  string
	Name    string
	Type    domain.ChannelType
	Config  json.RawMessage
	Enabled bool
}

type UpdateChannelInput struct {
	Name    *string
	Type    *domain.ChannelType
	Config  *json.RawMessage
	Enabled *bool
}

func (s *ChannelService) Create(ctx context.Context, in CreateChannelInput) (*domain.Channel, error) {
	if in.Name == "" || in.Type == "" {
		return nil, fmt.Errorf("%w: name and type required", ErrInvalidInput)
	}
	ch := &domain.Channel{
		ID:      uuid.NewString(),
		UserID:  in.UserID,
		Name:    in.Name,
		Type:    in.Type,
		Config:  in.Config,
		Enabled: in.Enabled,
	}
	if err := s.repo.Create(ctx, ch); err != nil {
		return nil, err
	}
	return ch, nil
}

func (s *ChannelService) GetByID(ctx context.Context, id string) (*domain.Channel, error) {
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	return ch, nil
}

func (s *ChannelService) List(ctx context.Context, userID string) ([]domain.Channel, error) {
	return s.repo.List(ctx, userID)
}

func (s *ChannelService) Update(ctx context.Context, id string, in UpdateChannelInput) (*domain.Channel, error) {
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotFound, err)
	}
	if in.Name != nil {
		ch.Name = *in.Name
	}
	if in.Type != nil {
		ch.Type = *in.Type
	}
	if in.Config != nil {
		ch.Config = *in.Config
	}
	if in.Enabled != nil {
		ch.Enabled = *in.Enabled
	}
	if err := s.repo.Update(ctx, ch); err != nil {
		return nil, err
	}
	return ch, nil
}

func (s *ChannelService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
