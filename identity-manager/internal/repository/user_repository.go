package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"identity-manager/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

type userScanner interface {
	Scan(dest ...any) error
}

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, userID string) (*model.User, error) {
	const query = `
SELECT id, identity_id, sub, login, email, employee_number, winaccountname,
       given_name, family_name, name, groups, is_active, last_login_at,
       created_at, updated_at
FROM users
WHERE id = $1
`

	user, err := scanUser(r.pool.QueryRow(ctx, query, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return user, nil
}

func (r *PostgresUserRepository) GetByIdentityID(ctx context.Context, identityID string) (*model.User, error) {
	const query = `
SELECT id, identity_id, sub, login, email, employee_number, winaccountname,
       given_name, family_name, name, groups, is_active, last_login_at,
       created_at, updated_at
FROM users
WHERE identity_id = $1
`

	user, err := scanUser(r.pool.QueryRow(ctx, query, identityID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by identity id: %w", err)
	}
	return user, nil
}

func (r *PostgresUserRepository) GetAllActive(ctx context.Context, limit, offset int) ([]model.User, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	const totalQuery = `SELECT COUNT(*) FROM users WHERE is_active = true`
	var total int
	if err := r.pool.QueryRow(ctx, totalQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count active users: %w", err)
	}

	const query = `
SELECT id, identity_id, sub, login, email, employee_number, winaccountname,
       given_name, family_name, name, groups, is_active, last_login_at,
       created_at, updated_at
FROM users
WHERE is_active = true
ORDER BY created_at DESC
LIMIT $1 OFFSET $2
`
	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get active users: %w", err)
	}
	defer rows.Close()

	users := make([]model.User, 0, limit)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan active user: %w", err)
		}
		users = append(users, *user)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error while reading active users: %w", err)
	}

	return users, total, nil
}

func scanUser(scanner userScanner) (*model.User, error) {
	var (
		user       model.User
		groupsJSON []byte
	)

	if err := scanner.Scan(
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
	); err != nil {
		return nil, err
	}

	decodedGroups, err := decodeUserGroups(groupsJSON)
	if err != nil {
		return nil, err
	}
	user.Groups = decodedGroups

	return &user, nil
}

func decodeUserGroups(groupsJSON []byte) ([]string, error) {
	if len(groupsJSON) == 0 {
		return nil, nil
	}

	var groups []string
	if err := json.Unmarshal(groupsJSON, &groups); err != nil {
		return nil, fmt.Errorf("failed to unmarshal groups: %w", err)
	}
	return groups, nil
}

func (r *PostgresUserRepository) UpsertFromClaims(ctx context.Context, claims model.IDTokenClaims) (*model.User, bool, error) {
	identityID := claims.IdentityID
	if identityID == "" {
		identityID = claims.Sub
	}
	if identityID == "" {
		return nil, false, fmt.Errorf("identity id is empty")
	}

	groupsJSON, err := json.Marshal(claims.Group)
	if err != nil {
		return nil, false, fmt.Errorf("failed to marshal groups: %w", err)
	}

	user, err := r.GetByIdentityID(ctx, identityID)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, false, err
	}

	now := time.Now()
	if user == nil {
		const createQuery = `
INSERT INTO users (
    id, identity_id, sub, login, email, employee_number, winaccountname,
    given_name, family_name, name, groups, is_active, last_login_at
) VALUES (
    gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, true, $11
)
RETURNING id, identity_id, sub, login, email, employee_number, winaccountname,
          given_name, family_name, name, is_active, last_login_at, created_at, updated_at
`
		var created model.User
		if err := r.pool.QueryRow(
			ctx,
			createQuery,
			identityID,
			claims.Sub,
			claims.Login,
			claims.Email,
			claims.EmployeeNumber,
			claims.WinAccountName,
			claims.GivenName,
			claims.FamilyName,
			claims.Name,
			groupsJSON,
			now,
		).Scan(
			&created.ID,
			&created.IdentityID,
			&created.Sub,
			&created.Login,
			&created.Email,
			&created.EmployeeNumber,
			&created.WinAccountName,
			&created.GivenName,
			&created.FamilyName,
			&created.Name,
			&created.IsActive,
			&created.LastLoginAt,
			&created.CreatedAt,
			&created.UpdatedAt,
		); err != nil {
			return nil, false, fmt.Errorf("failed to create user: %w", err)
		}
		created.Groups = claims.Group
		return &created, true, nil
	}

	const updateQuery = `
UPDATE users
SET sub = $2,
    login = $3,
    email = $4,
    employee_number = $5,
    winaccountname = $6,
    given_name = $7,
    family_name = $8,
    name = $9,
    groups = $10,
    last_login_at = $11,
    updated_at = CURRENT_TIMESTAMP
WHERE identity_id = $1
RETURNING id, identity_id, sub, login, email, employee_number, winaccountname,
          given_name, family_name, name, is_active, last_login_at, created_at, updated_at
`
	var updated model.User
	if err := r.pool.QueryRow(
		ctx,
		updateQuery,
		identityID,
		claims.Sub,
		claims.Login,
		claims.Email,
		claims.EmployeeNumber,
		claims.WinAccountName,
		claims.GivenName,
		claims.FamilyName,
		claims.Name,
		groupsJSON,
		now,
	).Scan(
		&updated.ID,
		&updated.IdentityID,
		&updated.Sub,
		&updated.Login,
		&updated.Email,
		&updated.EmployeeNumber,
		&updated.WinAccountName,
		&updated.GivenName,
		&updated.FamilyName,
		&updated.Name,
		&updated.IsActive,
		&updated.LastLoginAt,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	); err != nil {
		return nil, false, fmt.Errorf("failed to update user: %w", err)
	}
	updated.Groups = claims.Group

	return &updated, false, nil
}
