package service

import (
	"context"

	"identity-manager/internal/model"
)

type AuthService interface {
	BuildLoginURL(ctx context.Context, req *model.AuthRequest) (string, error)
	HandleCallback(ctx context.Context, query *model.AuthCallbackQuery) (*model.LoginResult, error)
	RefreshSession(ctx context.Context, sessionID string) (*model.Session, error)
	ValidateSession(ctx context.Context, sessionID string) (*model.Session, error)
	GetUserInfoBySessionID(ctx context.Context, sessionID string) (*model.UserInfo, error)
	Logout(ctx context.Context, sessionID string) error
	HandleBackChannelLogout(ctx context.Context, logoutToken string) error
	GetJWKSStatus(ctx context.Context) map[string]any
}

type UserService interface {
	GetCurrentUser(ctx context.Context, sessionID string) (*model.User, error)
	GetByID(ctx context.Context, userID string) (*model.User, error)
	GetByIdentityID(ctx context.Context, identityID string) (*model.User, error)
	GetAllActive(ctx context.Context, limit, offset int) ([]model.User, int, error)
}

type SessionService interface {
	GetCurrentSession(ctx context.Context, sessionID string) (*model.Session, error)
	RefreshSession(ctx context.Context, sessionID string) (*model.Session, error)
	RevokeSession(ctx context.Context, sessionID string) error
	ResolveIdentity(ctx context.Context, sessionID string) (*model.ResolvedIdentity, error)
}
