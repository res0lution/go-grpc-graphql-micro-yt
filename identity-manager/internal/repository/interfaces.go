package repository

import (
	"context"
	"time"

	"identity-manager/internal/model"
)

type UserRepository interface {
	GetByID(ctx context.Context, userID string) (*model.User, error)
	GetByIdentityID(ctx context.Context, identityID string) (*model.User, error)
	GetAllActive(ctx context.Context, limit, offset int) ([]model.User, int, error)
	UpsertFromClaims(ctx context.Context, claims model.IDTokenClaims) (*model.User, bool, error)
}

type SessionRepository interface {
	GetByID(ctx context.Context, sessionID string) (*model.Session, error)
	Exists(ctx context.Context, sessionID string) (bool, error)
	GetByUserID(ctx context.Context, userID string) ([]model.Session, error)
	Create(ctx context.Context, session model.Session) error
	UpdateTokens(ctx context.Context, sessionID string, token model.OAuth2TokenExchange, expiresAt time.Time) error
	Delete(ctx context.Context, sessionID string) error
	DeleteByUserID(ctx context.Context, userID string) error
}
