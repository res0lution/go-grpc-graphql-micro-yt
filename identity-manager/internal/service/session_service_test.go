package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"identity-manager/internal/model"
)

type sessionRepoMock struct {
	getByIDFn      func(ctx context.Context, sessionID string) (*model.Session, error)
	existsFn       func(ctx context.Context, sessionID string) (bool, error)
	getByUserIDFn  func(ctx context.Context, userID string) ([]model.Session, error)
	createFn       func(ctx context.Context, session model.Session) error
	updateTokensFn func(ctx context.Context, sessionID string, token model.OAuth2TokenExchange, expiresAt time.Time) error
	deleteFn       func(ctx context.Context, sessionID string) error
	deleteByUserFn func(ctx context.Context, userID string) error
}

func (m *sessionRepoMock) GetByID(ctx context.Context, sessionID string) (*model.Session, error) {
	return m.getByIDFn(ctx, sessionID)
}
func (m *sessionRepoMock) Exists(ctx context.Context, sessionID string) (bool, error) {
	if m.existsFn != nil {
		return m.existsFn(ctx, sessionID)
	}
	return false, nil
}
func (m *sessionRepoMock) GetByUserID(ctx context.Context, userID string) ([]model.Session, error) {
	if m.getByUserIDFn != nil {
		return m.getByUserIDFn(ctx, userID)
	}
	return nil, nil
}
func (m *sessionRepoMock) Create(ctx context.Context, session model.Session) error {
	if m.createFn != nil {
		return m.createFn(ctx, session)
	}
	return nil
}
func (m *sessionRepoMock) UpdateTokens(ctx context.Context, sessionID string, token model.OAuth2TokenExchange, expiresAt time.Time) error {
	if m.updateTokensFn != nil {
		return m.updateTokensFn(ctx, sessionID, token, expiresAt)
	}
	return nil
}
func (m *sessionRepoMock) Delete(ctx context.Context, sessionID string) error {
	return m.deleteFn(ctx, sessionID)
}
func (m *sessionRepoMock) DeleteByUserID(ctx context.Context, userID string) error {
	if m.deleteByUserFn != nil {
		return m.deleteByUserFn(ctx, userID)
	}
	return nil
}

type userRepoMock struct {
	getByIDFn         func(ctx context.Context, userID string) (*model.User, error)
	getByIdentityIDFn func(ctx context.Context, identityID string) (*model.User, error)
	getAllActiveFn    func(ctx context.Context, limit, offset int) ([]model.User, int, error)
	upsertFn          func(ctx context.Context, claims model.IDTokenClaims) (*model.User, bool, error)
}

func (m *userRepoMock) GetByID(ctx context.Context, userID string) (*model.User, error) {
	return m.getByIDFn(ctx, userID)
}
func (m *userRepoMock) GetByIdentityID(ctx context.Context, identityID string) (*model.User, error) {
	if m.getByIdentityIDFn != nil {
		return m.getByIdentityIDFn(ctx, identityID)
	}
	return nil, errors.New("not implemented")
}
func (m *userRepoMock) GetAllActive(ctx context.Context, limit, offset int) ([]model.User, int, error) {
	if m.getAllActiveFn != nil {
		return m.getAllActiveFn(ctx, limit, offset)
	}
	return nil, 0, nil
}
func (m *userRepoMock) UpsertFromClaims(ctx context.Context, claims model.IDTokenClaims) (*model.User, bool, error) {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, claims)
	}
	return nil, false, nil
}

type authServiceMock struct {
	buildLoginURLFn          func(context.Context, *model.AuthRequest) (string, error)
	handleCallbackFn         func(context.Context, *model.AuthCallbackQuery) (*model.LoginResult, error)
	refreshSessionFn         func(context.Context, string) (*model.Session, error)
	validateSessionFn        func(context.Context, string) (*model.Session, error)
	getUserInfoBySessionIDFn func(context.Context, string) (*model.UserInfo, error)
	logoutFn                 func(context.Context, string) error
	handleBackChannelFn      func(context.Context, string) error
	getJWKSStatusFn          func(context.Context) map[string]any
}

func (m *authServiceMock) BuildLoginURL(ctx context.Context, req *model.AuthRequest) (string, error) {
	return m.buildLoginURLFn(ctx, req)
}
func (m *authServiceMock) HandleCallback(ctx context.Context, q *model.AuthCallbackQuery) (*model.LoginResult, error) {
	return m.handleCallbackFn(ctx, q)
}
func (m *authServiceMock) RefreshSession(ctx context.Context, sessionID string) (*model.Session, error) {
	return m.refreshSessionFn(ctx, sessionID)
}
func (m *authServiceMock) ValidateSession(ctx context.Context, sessionID string) (*model.Session, error) {
	return m.validateSessionFn(ctx, sessionID)
}
func (m *authServiceMock) GetUserInfoBySessionID(ctx context.Context, sessionID string) (*model.UserInfo, error) {
	return m.getUserInfoBySessionIDFn(ctx, sessionID)
}
func (m *authServiceMock) Logout(ctx context.Context, sessionID string) error {
	return m.logoutFn(ctx, sessionID)
}
func (m *authServiceMock) HandleBackChannelLogout(ctx context.Context, token string) error {
	return m.handleBackChannelFn(ctx, token)
}
func (m *authServiceMock) GetJWKSStatus(ctx context.Context) map[string]any {
	return m.getJWKSStatusFn(ctx)
}

func TestSessionService_ResolveIdentity(t *testing.T) {
	svc := NewSessionService(
		&sessionRepoMock{
			getByIDFn: func(context.Context, string) (*model.Session, error) {
				return &model.Session{ID: "s1", UserID: "u1"}, nil
			},
			deleteFn: func(context.Context, string) error { return nil },
		},
		&userRepoMock{
			getByIDFn: func(context.Context, string) (*model.User, error) {
				return &model.User{IdentityID: "idp-1", Login: "alice", Groups: []string{"g1"}}, nil
			},
		},
		nil,
	)

	got, err := svc.ResolveIdentity(context.Background(), "s1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Identity.IdentityID != "idp-1" || got.Identity.Login != "alice" {
		t.Fatalf("unexpected identity context: %+v", got.Identity)
	}
	if got.UserInfo.IdentityID != "idp-1" || got.UserInfo.Login != "alice" {
		t.Fatalf("unexpected user info: %+v", got.UserInfo)
	}
}

func TestSessionService_RefreshDelegatesToAuth(t *testing.T) {
	called := false
	svc := NewSessionService(
		&sessionRepoMock{
			getByIDFn: func(context.Context, string) (*model.Session, error) {
				return &model.Session{ID: "fallback"}, nil
			},
			deleteFn: func(context.Context, string) error { return nil },
		},
		&userRepoMock{getByIDFn: func(context.Context, string) (*model.User, error) { return nil, nil }},
		&authServiceMock{
			refreshSessionFn: func(context.Context, string) (*model.Session, error) {
				called = true
				return &model.Session{ID: "from-auth"}, nil
			},
			buildLoginURLFn:          func(context.Context, *model.AuthRequest) (string, error) { return "", nil },
			handleCallbackFn:         func(context.Context, *model.AuthCallbackQuery) (*model.LoginResult, error) { return nil, nil },
			validateSessionFn:        func(context.Context, string) (*model.Session, error) { return nil, nil },
			getUserInfoBySessionIDFn: func(context.Context, string) (*model.UserInfo, error) { return nil, nil },
			logoutFn:                 func(context.Context, string) error { return nil },
			handleBackChannelFn:      func(context.Context, string) error { return nil },
			getJWKSStatusFn:          func(context.Context) map[string]any { return nil },
		},
	)

	session, err := svc.RefreshSession(context.Background(), "s1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called || session.ID != "from-auth" {
		t.Fatalf("expected refresh delegation, got %+v", session)
	}
}
