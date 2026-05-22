package handler

import (
	"context"
	"net/http"
	"testing"

	"identity-manager/internal/model"
	"identity-manager/internal/testutil"
)

type userSvcMock struct {
	getCurrentFn      func(context.Context, string) (*model.User, error)
	getByIDFn         func(context.Context, string) (*model.User, error)
	getByIdentityFn   func(context.Context, string) (*model.User, error)
	getAllActiveFn    func(context.Context, int, int) ([]model.User, int, error)
}

func (m *userSvcMock) GetCurrentUser(ctx context.Context, sessionID string) (*model.User, error) {
	return m.getCurrentFn(ctx, sessionID)
}
func (m *userSvcMock) GetByID(ctx context.Context, userID string) (*model.User, error) {
	return m.getByIDFn(ctx, userID)
}
func (m *userSvcMock) GetByIdentityID(ctx context.Context, identityID string) (*model.User, error) {
	return m.getByIdentityFn(ctx, identityID)
}
func (m *userSvcMock) GetAllActive(ctx context.Context, limit, offset int) ([]model.User, int, error) {
	return m.getAllActiveFn(ctx, limit, offset)
}

func TestUserHandler_Me(t *testing.T) {
	h := NewUserHandler(&userSvcMock{
		getCurrentFn: func(context.Context, string) (*model.User, error) {
			return &model.User{ID: "u1", Login: "alice"}, nil
		},
	})
	ctx, rec := testutil.NewGinContext(http.MethodGet, "/v1/users/me", nil)
	ctx.Set("session_id", "s1")
	h.Me(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}
