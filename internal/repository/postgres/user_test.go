package postgres

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/vantutran2k1/SignalFlow/internal/domain"
	"github.com/vantutran2k1/SignalFlow/internal/testdb"
)

func TestUserRepository_RoundTrip(t *testing.T) {
	pool := testdb.NewPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	user := &domain.User{
		ID:           uuid.NewString(),
		Email:        "alice@example.com",
		PasswordHash: "hash",
		Name:         "Alice",
	}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := repo.GetByEmail(ctx, "alice@example.com")
	if err != nil {
		t.Fatalf("GetByEmail: %v", err)
	}
	if got.ID != user.ID || got.Name != "Alice" {
		t.Errorf("got %+v, want id=%s name=Alice", got, user.ID)
	}
}

func TestUserRepository_DuplicateEmailRejected(t *testing.T) {
	pool := testdb.NewPool(t)
	repo := NewUserRepository(pool)
	ctx := context.Background()

	first := &domain.User{ID: uuid.NewString(), Email: "dup@example.com", PasswordHash: "h"}
	if err := repo.Create(ctx, first); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	second := &domain.User{ID: uuid.NewString(), Email: "dup@example.com", PasswordHash: "h2"}
	err := repo.Create(ctx, second)
	if err == nil {
		t.Fatal("expected unique-violation error on duplicate email")
	}
	// Postgres returns "duplicate key value violates unique constraint".
	if !strings.Contains(err.Error(), "duplicate") && !strings.Contains(err.Error(), "unique") {
		t.Errorf("err = %v, want unique-violation message", err)
	}
}
