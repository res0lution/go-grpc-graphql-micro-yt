package repository

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"identity-manager/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

func openTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set; skipping integration tests")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(context.Background(), `CREATE EXTENSION IF NOT EXISTS pgcrypto;`); err != nil {
		t.Fatalf("enable pgcrypto: %v", err)
	}
	if err := applyMigration(t, pool); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `TRUNCATE TABLE sessions, users RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
	return pool
}

func applyMigration(t *testing.T, pool *pgxpool.Pool) error {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	migrationPath := filepath.Join(filepath.Dir(thisFile), "..", "db", "migrations", "001_init_users_sessions.sql")
	data, err := os.ReadFile(migrationPath)
	if err != nil {
		return err
	}
	_, err = pool.Exec(context.Background(), string(data))
	return err
}

func TestUserRepositoryIntegration_UpsertCreateUpdate(t *testing.T) {
	pool := openTestDB(t)
	repo := NewPostgresUserRepository(pool)

	created, isNew, err := repo.UpsertFromClaims(context.Background(), model.IDTokenClaims{
		Sub:        "sub-1",
		IdentityID: "id-1",
		Login:      "alice",
		Email:      "alice@corp.local",
		Group:      []string{"A"},
	})
	if err != nil {
		t.Fatalf("create upsert failed: %v", err)
	}
	if !isNew || created.ID == "" {
		t.Fatalf("expected created user")
	}

	updated, isNew, err := repo.UpsertFromClaims(context.Background(), model.IDTokenClaims{
		Sub:        "sub-1",
		IdentityID: "id-1",
		Login:      "alice-updated",
		Email:      "alice2@corp.local",
		Group:      []string{"A", "B"},
	})
	if err != nil {
		t.Fatalf("update upsert failed: %v", err)
	}
	if isNew || updated.Login != "alice-updated" {
		t.Fatalf("expected updated user")
	}
}

func TestUserRepositoryIntegration_GetAllActive(t *testing.T) {
	pool := openTestDB(t)
	repo := NewPostgresUserRepository(pool)

	_, _, _ = repo.UpsertFromClaims(context.Background(), model.IDTokenClaims{
		Sub:        "sub-2",
		IdentityID: "id-2",
		Login:      "bob",
		Email:      "bob@corp.local",
		Group:      []string{"A"},
	})
	_, err := pool.Exec(context.Background(), `UPDATE users SET is_active = false WHERE identity_id = 'id-2'`)
	if err != nil {
		t.Fatalf("deactivate user: %v", err)
	}
	_, _, _ = repo.UpsertFromClaims(context.Background(), model.IDTokenClaims{
		Sub:        "sub-3",
		IdentityID: "id-3",
		Login:      "carol",
		Email:      "carol@corp.local",
		Group:      []string{"A"},
	})

	users, total, err := repo.GetAllActive(context.Background(), 10, 0)
	if err != nil {
		t.Fatalf("GetAllActive failed: %v", err)
	}
	if total < 1 || len(users) < 1 {
		t.Fatalf("expected at least one active user")
	}
}

func TestSessionRepositoryIntegration_SQLPaths(t *testing.T) {
	pool := openTestDB(t)
	users := NewPostgresUserRepository(pool)
	sessions := NewPostgresSessionRepository(pool)

	user, _, err := users.UpsertFromClaims(context.Background(), model.IDTokenClaims{
		Sub:        "sub-10",
		IdentityID: "id-10",
		Login:      "david",
		Email:      "david@corp.local",
		Group:      []string{"A"},
	})
	if err != nil {
		t.Fatalf("upsert user: %v", err)
	}

	exp := time.Now().Add(1 * time.Hour)
	err = sessions.Create(context.Background(), model.Session{
		ID:           "11111111-1111-1111-1111-111111111111",
		UserID:       user.ID,
		AccessToken:  "a1",
		RefreshToken: "r1",
		IDToken:      "i1",
		TokenExpiry:  exp,
		ExpiresAt:    exp,
	})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := sessions.UpdateTokens(context.Background(), "11111111-1111-1111-1111-111111111111", model.OAuth2TokenExchange{
		AccessToken:  "a2",
		RefreshToken: "r2",
		IDToken:      "i2",
	}, time.Now().Add(2*time.Hour)); err != nil {
		t.Fatalf("update tokens: %v", err)
	}

	if _, err := sessions.GetByID(context.Background(), "11111111-1111-1111-1111-111111111111"); err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if err := sessions.DeleteByUserID(context.Background(), user.ID); err != nil {
		t.Fatalf("delete by user: %v", err)
	}
}
