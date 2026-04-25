package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vantutran2k1/SignalFlow/internal/domain"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, email, password_hash, name, created_at, updated_at)
         VALUES ($1, $2, $3, $4, now(), now())`,
		user.ID, user.Email, user.PasswordHash, user.Name,
	)
	return err
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	user := &domain.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, name, created_at, updated_at
         FROM users WHERE id = $1`, id,
	).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	user := &domain.User{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, name, created_at, updated_at
         FROM users WHERE email = $1`, email,
	).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (r *UserRepository) Update(ctx context.Context, user *domain.User) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET email=$1, password_hash=$2, name=$3, updated_at=now()
         WHERE id=$4`,
		user.Email, user.PasswordHash, user.Name, user.ID,
	)
	return err
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}
