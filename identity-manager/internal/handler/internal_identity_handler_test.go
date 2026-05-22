package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"identity-manager/internal/model"
	"identity-manager/internal/testutil"
)

func TestInternalIdentityHandler_Resolve_FromHeader(t *testing.T) {
	h := NewInternalIdentityHandler(&sessionSvcMock{
		resolveIDFn: func(context.Context, string) (*model.ResolvedIdentity, error) {
			return &model.ResolvedIdentity{
				Identity: model.IdentityContext{SessionID: "s1", UserID: "u1"},
				UserInfo: model.UserInfo{Sub: "u1", Group: []string{"G1"}},
			}, nil
		},
	})

	ctx, rec := testutil.NewGinContext(http.MethodPost, "/v1/internal/identity/resolve", nil)
	ctx.Request.Header.Set("X-Session-ID", "s1")
	h.Resolve(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestInternalIdentityHandler_Resolve_FromContext(t *testing.T) {
	h := NewInternalIdentityHandler(&sessionSvcMock{
		resolveIDFn: func(context.Context, string) (*model.ResolvedIdentity, error) {
			return &model.ResolvedIdentity{
				Identity: model.IdentityContext{SessionID: "s1", UserID: "u1"},
				UserInfo: model.UserInfo{Sub: "u1", Group: []string{"G1"}},
			}, nil
		},
	})
	ctx, rec := testutil.NewGinContext(http.MethodPost, "/v1/internal/identity/resolve", nil)
	ctx.Set("session_id", "s1")
	h.Resolve(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestInternalIdentityHandler_Resolve_BadJSON(t *testing.T) {
	h := NewInternalIdentityHandler(&sessionSvcMock{})
	ctx, rec := testutil.NewGinContext(http.MethodPost, "/v1/internal/identity/resolve", []byte("{bad"))
	h.Resolve(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestInternalIdentityHandler_Resolve_NotFound(t *testing.T) {
	h := NewInternalIdentityHandler(&sessionSvcMock{
		resolveIDFn: func(context.Context, string) (*model.ResolvedIdentity, error) {
			return nil, errors.New("not found")
		},
	})
	ctx, rec := testutil.NewGinContext(http.MethodPost, "/v1/internal/identity/resolve", []byte(`{"session_id":"s1"}`))
	h.Resolve(ctx)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
