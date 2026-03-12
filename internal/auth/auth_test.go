package auth

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/example/kpi-chaser/internal/storage"
)

func setupAuthService(t *testing.T) (*Service, string) {
	t.Helper()

	db, err := storage.Open(filepath.Join(t.TempDir(), "auth-test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := storage.Migrate(db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}

	return NewService(db, "test-secret"), "test@example.com"
}

func TestVerifyPassword(t *testing.T) {
	ctx := context.Background()
	svc, email := setupAuthService(t)

	user, _, err := svc.Register(ctx, email, "password123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := svc.VerifyPassword(ctx, user.ID, "password123"); err != nil {
		t.Fatalf("verify valid password: %v", err)
	}
	if err := svc.VerifyPassword(ctx, user.ID, "wrong-password"); err == nil {
		t.Fatal("expected invalid password error")
	}
}

func TestLoginRateLimitBlocksAfterRepeatedFailures(t *testing.T) {
	ctx := context.Background()
	svc, email := setupAuthService(t)

	if _, _, err := svc.Register(ctx, email, "password123"); err != nil {
		t.Fatalf("register: %v", err)
	}

	limiterKey := "test@example.com|127.0.0.1"
	for range 5 {
		if _, _, err := svc.Login(ctx, email, "wrong-password", limiterKey); err == nil {
			t.Fatal("expected invalid credentials error")
		}
	}

	if _, _, err := svc.Login(ctx, email, "password123", limiterKey); err == nil {
		t.Fatal("expected rate limit error after repeated failures")
	}
}
