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

type UserRepository struct {
 pool   *pgxpool.Pool
 logger *logrus.Entry
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
 return &UserRepository{
  pool:   pool,
  logger: logger.L().WithField("component", "user_repository"),
 }
}

/*
========================
Create table
========================
*/

func (r *UserRepository) CreateTables(ctx context.Context) error {
 usersTable := 
 CREATE TABLE IF NOT EXISTS users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  identity_id VARCHAR(255) UNIQUE NOT NULL,
  sub VARCHAR(255) NOT NULL,
  login VARCHAR(255),
  email VARCHAR(255),
  employee_number VARCHAR(50),
  winaccountname VARCHAR(255),
  given_name VARCHAR(255),
  family_name VARCHAR(255),
  name VARCHAR(255),
  groups JSONB,
  is_active BOOLEAN DEFAULT TRUE,
  last_login_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
 );
 

 indexes := []string{
  CREATE INDEX IF NOT EXISTS idx_users_identity_id ON users(identity_id);,
  CREATE INDEX IF NOT EXISTS idx_users_sub ON users(sub);,
  CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);,
  CREATE INDEX IF NOT EXISTS idx_users_login ON users(login);,
  CREATE INDEX IF NOT EXISTS idx_users_employee_number ON users(employee_number);,
  CREATE INDEX IF NOT EXISTS idx_users_is_active ON users(is_active);,
 }

 trigger := 
 CREATE OR REPLACE FUNCTION update_users_updated_at()
 RETURNS TRIGGER AS $$
 BEGIN
  NEW.updated_at = CURRENT_TIMESTAMP;
  RETURN NEW;
 END;
 $$ LANGUAGE plpgsql;

 DROP TRIGGER IF EXISTS update_users_updated_at ON users;

 CREATE TRIGGER update_users_updated_at
 BEFORE UPDATE ON users
 FOR EACH ROW
 EXECUTE FUNCTION update_users_updated_at();
 

 queries := append([]string{usersTable, trigger}, indexes...)

 for _, q := range queries {
  if _, err := r.pool.Exec(ctx, q); err != nil {
   return fmt.Errorf("failed to execute migration: %w", err)
  }
 }

 r.logger.Info("user table migration completed")
 return nil
}

/*
========================
Create user
========================
*/

func (r *UserRepository) Create(ctx context.Context, user model.User) error {
 groupsJSON, err := json.Marshal(user.Groups)
 if err != nil {
  return fmt.Errorf("failed to marshal groups: %w", err)
 }

 query := 
 INSERT INTO users (
  id, identity_id, sub, login, email,
  employee_number, winaccountname,
  given_name, family_name, name,
  groups, is_active, last_login_at
 )
 VALUES (
  $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13
 )
 RETURNING created_at, updated_at
 

 err = r.pool.QueryRow(
  ctx,
  query,
  user.ID,
  user.IdentityID,
  user.Sub,
  user.Login,
  user.Email,
  user.EmployeeNumber,
  user.WinAccountName,
  user.GivenName,
  user.FamilyName,
  user.Name,
  groupsJSON,
  user.IsActive,
  user.LastLoginAt,
 ).Scan(&user.CreatedAt, &user.UpdatedAt)

 if err != nil {
  return fmt.Errorf("failed to create user: %w", err)
 }

 r.logger.WithFields(logrus.Fields{
  "user_id":     user.ID,
  "identity_id": user.IdentityID,
  "login":       user.Login,
 }).Debug("user created")

 return nil
}

/*
========================
Get by identity ID
========================
*/

func (r *UserRepository) GetByIdentityID(ctx context.Context, identityID string) (*model.User, error) {
 query := 
 SELECT
  id, identity_id, sub, login, email,
  employee_number, winaccountname,
  given_name, family_name, name,
  groups, is_active, last_login_at,
  created_at, updated_at
 FROM users
 WHERE identity_id = $1
 

 var user model.User
 var groupsJSON []byte

 err := r.pool.QueryRow(ctx, query, identityID).Scan(
  &user.ID,
  &user.IdentityID,
  &user.Sub,
  &user.Login,
  &user.Email,
  &user.EmployeeNumber,
  &user.WinAccountName,
  &user.GivenName,
  &user.FamilyName,
  &user.Name,
  &groupsJSON,
  &user.IsActive,
  &user.LastLoginAt,
  &user.CreatedAt,
  &user.UpdatedAt,
 )
 if err != nil {
  return nil, fmt.Errorf("failed to get user: %w", err)
 }

 if len(groupsJSON) > 0 {
  if err := json.Unmarshal(groupsJSON, &user.Groups); err != nil {
   r.logger.WithError(err).Warn("failed to unmarshal groups")
   user.Groups = []string{}
  }
 }

 return &user, nil
}

/*
========================
Exists
========================
*/

func (r *UserRepository) Exists(ctx context.Context, id string) (bool, error) {
 query := SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)

 var exists bool
 if err := r.pool.QueryRow(ctx, query, id).Scan(&exists); err != nil {
  return false, fmt.Errorf("failed to check existence: %w", err)
 }

 return exists, nil
}

func (r *UserRepository) ExistsByIdentityID(ctx context.Context, identityID string) (bool, error) {
 query := SELECT EXISTS(SELECT 1 FROM users WHERE identity_id = $1)

 var exists bool
 if err := r.pool.QueryRow(ctx, query, identityID).Scan(&exists); err != nil {
  return false, fmt.Errorf("failed to check existence by identity_id: %w", err)
 }

 return exists, nil
}

/*
========================
Update last login
========================
*/

func (r *UserRepository) UpdateLastLogin(ctx context.Context, identityID string) error {
 query := 
 UPDATE users
 SET last_login_at = $1
 WHERE identity_id = $2
 

 res, err := r.pool.Exec(ctx, query, time.Now(), identityID)
 if err != nil {
  return fmt.Errorf("failed to update last login: %w", err)
 }

 if res.RowsAffected() == 0 {
  return fmt.Errorf("user not found")
 }

 return nil
}