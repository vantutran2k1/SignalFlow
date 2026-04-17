package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

type ChannelRepository struct {
	pool *pgxpool.Pool
}

func NewChannelRepository(pool *pgxpool.Pool) *ChannelRepository {
	return &ChannelRepository{pool: pool}
}

func (r *ChannelRepository) Create(ctx context.Context, ch *domain.Channel) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO channels (id, user_id, name, type, config, enabled, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, now(), now())`,
		ch.ID, ch.UserID, ch.Name, ch.Type, ch.Config, ch.Enabled,
	)
	return err
}

func (r *ChannelRepository) GetByID(ctx context.Context, id string) (*domain.Channel, error) {
	ch := &domain.Channel{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, user_id, name, type, config, enabled, created_at, updated_at
         FROM channels WHERE id = $1`, id,
	).Scan(
		&ch.ID, &ch.UserID, &ch.Name, &ch.Type, &ch.Config, &ch.Enabled,
		&ch.CreatedAt, &ch.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return ch, nil
}

func (r *ChannelRepository) GetByIDs(ctx context.Context, ids []string) ([]domain.Channel, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, name, type, config, enabled, created_at, updated_at
         FROM channels WHERE id = ANY($1)`, ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []domain.Channel
	for rows.Next() {
		var ch domain.Channel
		if err := rows.Scan(
			&ch.ID, &ch.UserID, &ch.Name, &ch.Type, &ch.Config, &ch.Enabled,
			&ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}
	return channels, nil
}

func (r *ChannelRepository) List(ctx context.Context, userID string) ([]domain.Channel, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, user_id, name, type, config, enabled, created_at, updated_at
         FROM channels WHERE user_id = $1 ORDER BY created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []domain.Channel
	for rows.Next() {
		var ch domain.Channel
		if err := rows.Scan(
			&ch.ID, &ch.UserID, &ch.Name, &ch.Type, &ch.Config, &ch.Enabled,
			&ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}
	return channels, nil
}

func (r *ChannelRepository) Update(ctx context.Context, ch *domain.Channel) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE channels SET name=$1, type=$2, config=$3, enabled=$4, updated_at=now()
         WHERE id=$5`,
		ch.Name, ch.Type, ch.Config, ch.Enabled, ch.ID,
	)
	return err
}

func (r *ChannelRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM channels WHERE id = $1`, id)
	return err
}