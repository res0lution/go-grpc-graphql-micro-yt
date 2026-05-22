package handler

import (
	"context"
	"net/http"
	"testing"
	"time"

	"identity-manager/internal/model"
	"identity-manager/internal/testutil"
)

type sessionSvcMock struct {
	getCurrentFn   func(context.Context, string) (*model.Session, error)
	refreshFn      func(context.Context, string) (*model.Session, error)
	revokeFn       func(context.Context, string) error
	resolveIDFn    func(context.Context, string) (*model.ResolvedIdentity, error)
}

func (m *sessionSvcMock) GetCurrentSession(ctx context.Context, sessionID string) (*model.Session, error) {
	return m.getCurrentFn(ctx, sessionID)
}
func (m *sessionSvcMock) RefreshSession(ctx context.Context, sessionID string) (*model.Session, error) {
	return m.refreshFn(ctx, sessionID)
}
func (m *sessionSvcMock) RevokeSession(ctx context.Context, sessionID string) error {
	return m.revokeFn(ctx, sessionID)
}
func (m *sessionSvcMock) ResolveIdentity(ctx context.Context, sessionID string) (*model.ResolvedIdentity, error) {
	return m.resolveIDFn(ctx, sessionID)
}

func TestSessionHandler_Refresh(t *testing.T) {
	h := NewSessionHandler(&sessionSvcMock{
		refreshFn: func(context.Context, string) (*model.Session, error) {
			return &model.Session{ID: "s1", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
	})
	ctx, rec := testutil.NewGinContext(http.MethodPost, "/v1/sessions/refresh", nil)
	ctx.Set("session_id", "s1")
	h.Refresh(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestSessionHandler_Me(t *testing.T) {
	h := NewSessionHandler(&sessionSvcMock{
		getCurrentFn: func(context.Context, string) (*model.Session, error) {
			return &model.Session{ID: "s1"}, nil
		},
	})
	ctx, rec := testutil.NewGinContext(http.MethodGet, "/v1/sessions/me", nil)
	ctx.Set("session_id", "s1")
	h.Me(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestSessionHandler_Delete(t *testing.T) {
	h := NewSessionHandler(&sessionSvcMock{
		revokeFn: func(context.Context, string) error { return nil },
	})
	ctx, rec := testutil.NewGinContext(http.MethodDelete, "/v1/sessions/me", nil)
	ctx.Set("session_id", "s1")
	h.Delete(ctx)
	if ctx.Writer.Status() != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}
