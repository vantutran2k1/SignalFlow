package service

import (
	"context"
	"errors"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

const authTestSecret = "auth-test-secret"

func TestAuthService_RegisterAndLogin(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewAuthService(repo, authTestSecret)
	ctx := context.Background()

	user, err := svc.Register(ctx, "alice@example.com", "hunter2", "Alice")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if user.PasswordHash == "hunter2" {
		t.Error("password stored in plaintext!")
	}

	tokenStr, err := svc.Login(ctx, "alice@example.com", "hunter2")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	parsed, err := jwt.Parse(tokenStr, func(_ *jwt.Token) (any, error) {
		return []byte(authTestSecret), nil
	})
	if err != nil || !parsed.Valid {
		t.Fatalf("parse token: err=%v valid=%v", err, parsed.Valid)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if claims["sub"] != user.ID {
		t.Errorf("token sub = %v, want %v", claims["sub"], user.ID)
	}
}

func TestAuthService_Register_RejectsEmpty(t *testing.T) {
	svc := NewAuthService(newFakeUserRepo(), authTestSecret)
	if _, err := svc.Register(context.Background(), "", "x", "n"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput for empty email", err)
	}
	if _, err := svc.Register(context.Background(), "a@b", "", "n"); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("err = %v, want ErrInvalidInput for empty password", err)
	}
}

func TestAuthService_Register_DuplicateEmail(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewAuthService(repo, authTestSecret)
	if _, err := svc.Register(context.Background(), "dup@x", "p", "n"); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if _, err := svc.Register(context.Background(), "dup@x", "p2", "n2"); !errors.Is(err, ErrConflict) {
		t.Errorf("err = %v, want ErrConflict on duplicate", err)
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewAuthService(repo, authTestSecret)
	if _, err := svc.Register(context.Background(), "u@x", "right", "n"); err != nil {
		t.Fatal(err)
	}
	_, err := svc.Login(context.Background(), "u@x", "wrong")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthService_Login_UnknownUser(t *testing.T) {
	svc := NewAuthService(newFakeUserRepo(), authTestSecret)
	// No users registered.
	_, err := svc.Login(context.Background(), "ghost@x", "p")
	if !errors.Is(err, ErrInvalidCredentials) {
		// Important: unknown-user must surface the same error as wrong-password.
		// Otherwise the login endpoint becomes a user-enumeration oracle.
		t.Errorf("err = %v, want ErrInvalidCredentials (no user enumeration)", err)
	}
}
