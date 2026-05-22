package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"identity-manager/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresSessionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSessionRepository(pool *pgxpool.Pool) *PostgresSessionRepository {
	return &PostgresSessionRepository{pool: pool}
}

func (r *PostgresSessionRepository) GetByID(ctx context.Context, sessionID string) (*model.Session, error) {
	const query = `
SELECT id, user_id, access_token, refresh_token, id_token, token_expiry, expires_at, created_at, updated_at
FROM sessions
WHERE id = $1 AND expires_at > $2
`
	var session model.Session
	if err := r.pool.QueryRow(ctx, query, sessionID, time.Now()).Scan(
		&session.ID,
		&session.UserID,
		&session.AccessToken,
		&session.RefreshToken,
		&session.IDToken,
		&session.TokenExpiry,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}
	return &session, nil
}

func (r *PostgresSessionRepository) Create(ctx context.Context, session model.Session) error {
	const query = `
INSERT INTO sessions (id, user_id, access_token, refresh_token, id_token, token_expiry, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING created_at, updated_at
`
	if err := r.pool.QueryRow(
		ctx,
		query,
		session.ID,
		session.UserID,
		session.AccessToken,
		session.RefreshToken,
		session.IDToken,
		session.TokenExpiry,
		session.ExpiresAt,
	).Scan(&session.CreatedAt, &session.UpdatedAt); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	return nil
}

func (r *PostgresSessionRepository) UpdateTokens(ctx context.Context, sessionID string, token model.OAuth2TokenExchange, expiresAt time.Time) error {
	const query = `
UPDATE sessions
SET access_token = $2,
    refresh_token = $3,
    id_token = $4,
    token_expiry = $5,
    expires_at = $5,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
`
	res, err := r.pool.Exec(
		ctx,
		query,
		sessionID,
		token.AccessToken,
		token.RefreshToken,
		token.IDToken,
		expiresAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update session tokens: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (r *PostgresSessionRepository) Delete(ctx context.Context, sessionID string) error {
	const query = `DELETE FROM sessions WHERE id = $1`
	res, err := r.pool.Exec(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (r *PostgresSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	const query = `DELETE FROM sessions WHERE user_id = $1`
	if _, err := r.pool.Exec(ctx, query, userID); err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}
	return nil
}

func (r *PostgresSessionRepository) Exists(ctx context.Context, sessionID string) (bool, error) {
	const query = `
SELECT EXISTS(
  SELECT 1 FROM sessions WHERE id = $1 AND expires_at > $2
)
`
	var exists bool
	if err := r.pool.QueryRow(ctx, query, sessionID, time.Now()).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}
	return exists, nil
}

func (r *PostgresSessionRepository) GetByUserID(ctx context.Context, userID string) ([]model.Session, error) {
	const query = `
SELECT id, user_id, access_token, refresh_token, id_token, token_expiry, expires_at, created_at, updated_at
FROM sessions
WHERE user_id = $1 AND expires_at > $2
ORDER BY created_at DESC
`
	rows, err := r.pool.Query(ctx, query, userID, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to get sessions by user id: %w", err)
	}
	defer rows.Close()

	var sessions []model.Session
	for rows.Next() {
		var session model.Session
		if err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.AccessToken,
			&session.RefreshToken,
			&session.IDToken,
			&session.TokenExpiry,
			&session.ExpiresAt,
			&session.CreatedAt,
			&session.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return sessions, nil
}
