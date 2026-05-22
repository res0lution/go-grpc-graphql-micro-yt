package repository

import (
 "context"
 "encoding/json"
 "fmt"
 "time"

 "portal-core/internal/logger"
 "portal-core/internal/model"

 "github.com/jackc/pgx/v5/pgxpool"
 "github.com/sirupsen/logrus"
)

type SessionRepository struct {
 pool   *pgxpool.Pool
 logger *logrus.Entry
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
 return &SessionRepository{
  pool:   pool,
  logger: logger.L().WithField("component", "session_repository"),
 }
}

/*
========================
Create tables
========================
*/

func (r *SessionRepository) CreateTables(ctx context.Context) error {
 query := 
 CREATE TABLE IF NOT EXISTS sessions (
  id UUID PRIMARY KEY,
  user_id VARCHAR(255) NOT NULL,
  access_token TEXT NOT NULL,
  refresh_token TEXT NOT NULL,
  id_token TEXT NOT NULL,
  token_expiry TIMESTAMP NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
 );

 CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
 CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
 CREATE INDEX IF NOT EXISTS idx_sessions_token_expiry ON sessions(token_expiry);
 

 _, err := r.pool.Exec(ctx, query)
 if err != nil {
  return fmt.Errorf("failed to create session tables: %w", err)
 }

 r.logger.Info("session tables created or already exist")
 return nil
}

/*
========================
Create session
========================
*/

func (r *SessionRepository) Create(ctx context.Context, session model.Session) error {
 query := 
 INSERT INTO sessions (
  id, user_id, access_token, refresh_token,
  id_token, token_expiry, expires_at
 )
 VALUES ($1,$2,$3,$4,$5,$6,$7)
 RETURNING created_at, updated_at
 

 err := r.pool.QueryRow(
  ctx,
  query,
  session.ID,
  session.UserID,
  session.AccessToken,
  session.RefreshToken,
  session.IDToken,
  session.TokenExpiry,
  session.ExpiresAt,
 ).Scan(&session.CreatedAt, &session.UpdatedAt)

 if err != nil {
  return fmt.Errorf("failed to create session: %w", err)
 }

 r.logger.WithField("id", session.ID).Debug("session created")
 return nil
}

/*
========================
Get by ID
========================
*/

func (r *SessionRepository) GetByID(ctx context.Context, sessionID string) (*model.Session, error) {
 query := 
 SELECT id, user_id, access_token, refresh_token,
  id_token, token_expiry, expires_at,
  created_at, updated_at
 FROM sessions
 WHERE id = $1 AND expires_at > $2
 

 var session model.Session

 err := r.pool.QueryRow(ctx, query, sessionID, time.Now()).Scan(
  &session.ID,
  &session.UserID,
  &session.AccessToken,
  &session.RefreshToken,
  &session.IDToken,
  &session.TokenExpiry,
  &session.ExpiresAt,
  &session.CreatedAt,
  &session.UpdatedAt,
 )

 if err != nil {
  return nil, fmt.Errorf("failed to get session: %w", err)
 }

 return &session, nil
}

/*
========================
Delete
========================
*/

func (r *SessionRepository) Delete(ctx context.Context, sessionID string) error {
 query := DELETE FROM sessions WHERE id = $1

 res, err := r.pool.Exec(ctx, query, sessionID)
 if err != nil {
  return fmt.Errorf("failed to delete session: %w", err)
 }

 if res.RowsAffected() == 0 {
  return fmt.Errorf("session not found")
 }

 r.logger.WithField("id", sessionID).Debug("session deleted")
 return nil
}

/*
========================
Delete by user
========================
*/

func (r *SessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
 query := DELETE FROM sessions WHERE user_id = $1

 res, err := r.pool.Exec(ctx, query, userID)
 if err != nil {
  return fmt.Errorf("failed to delete user sessions: %w", err)
 }

 r.logger.WithFields(logrus.Fields{
  "user_id":        userID,
  "rows_affected": res.RowsAffected(),
 }).Debug("user sessions deleted")

 return nil
}

/*
========================
Exists
========================
*/

func (r *SessionRepository) Exists(ctx context.Context, sessionID string) (bool, error) {
 query := 
 SELECT EXISTS(
  SELECT 1 FROM sessions
  WHERE id = $1 AND expires_at > $2
 )

 var exists bool
 err := r.pool.QueryRow(ctx, query, sessionID, time.Now()).Scan(&exists)
 if err != nil {
  return false, fmt.Errorf("failed to check session existence: %w", err)
 }

 return exists, nil
}

/*
========================
Get by user
========================
*/

func (r *SessionRepository) GetByUserID(ctx context.Context, userID string) ([]model.Session, error) {
 query := 
 SELECT id, user_id, access_token, refresh_token,
  id_token, token_expiry, expires_at,
  created_at, updated_at
 FROM sessions
 WHERE user_id = $1 AND expires_at > $2
 ORDER BY created_at DESC
 

 rows, err := r.pool.Query(ctx, query, userID, time.Now())
 if err != nil {
  return nil, fmt.Errorf("failed to get user sessions: %w", err)
 }
 defer rows.Close()

 var sessions []model.Session

 for rows.Next() {
  var s model.Session

  if err := rows.Scan(
   &s.ID,
   &s.UserID,
   &s.AccessToken,
   &s.RefreshToken,
   &s.IDToken,
   &s.TokenExpiry,
   &s.ExpiresAt,
   &s.CreatedAt,
   &s.UpdatedAt,
  ); err != nil {
   return nil, fmt.Errorf("failed to scan session: %w", err)
  }

  sessions = append(sessions, s)
 }

 if err := rows.Err(); err != nil {
  return nil, fmt.Errorf("rows error: %w", err)
 }

 return sessions, nil
}