package handler

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"identity-manager/internal/config"
	"identity-manager/internal/model"
	"identity-manager/internal/testutil"
)

type authSvcMock struct {
	buildLoginURLFn          func(context.Context, *model.AuthRequest) (string, error)
	handleCallbackFn         func(context.Context, *model.AuthCallbackQuery) (*model.LoginResult, error)
	refreshSessionFn         func(context.Context, string) (*model.Session, error)
	validateSessionFn        func(context.Context, string) (*model.Session, error)
	getUserInfoBySessionIDFn func(context.Context, string) (*model.UserInfo, error)
	logoutFn                 func(context.Context, string) error
	handleBackChannelFn      func(context.Context, string) error
	getJWKSStatusFn          func(context.Context) map[string]any
}

func (m *authSvcMock) BuildLoginURL(ctx context.Context, req *model.AuthRequest) (string, error) {
	return m.buildLoginURLFn(ctx, req)
}
func (m *authSvcMock) HandleCallback(ctx context.Context, q *model.AuthCallbackQuery) (*model.LoginResult, error) {
	return m.handleCallbackFn(ctx, q)
}
func (m *authSvcMock) RefreshSession(ctx context.Context, sessionID string) (*model.Session, error) {
	return m.refreshSessionFn(ctx, sessionID)
}
func (m *authSvcMock) ValidateSession(ctx context.Context, sessionID string) (*model.Session, error) {
	return m.validateSessionFn(ctx, sessionID)
}
func (m *authSvcMock) GetUserInfoBySessionID(ctx context.Context, sessionID string) (*model.UserInfo, error) {
	return m.getUserInfoBySessionIDFn(ctx, sessionID)
}
func (m *authSvcMock) Logout(ctx context.Context, sessionID string) error { return m.logoutFn(ctx, sessionID) }
func (m *authSvcMock) HandleBackChannelLogout(ctx context.Context, logoutToken string) error {
	return m.handleBackChannelFn(ctx, logoutToken)
}
func (m *authSvcMock) GetJWKSStatus(ctx context.Context) map[string]any { return m.getJWKSStatusFn(ctx) }

func TestAuthHandler_LoginRedirect(t *testing.T) {
	h := NewAuthHandler(&authSvcMock{
		buildLoginURLFn: func(context.Context, *model.AuthRequest) (string, error) { return "https://idp/auth", nil },
	}, config.CookieConfig{Name: "session_id", Secure: true, HTTPOnly: true, SameSite: "Lax"})
	ctx, rec := testutil.NewGinContext(http.MethodGet, "/v1/auth/login", nil)
	h.Login(ctx)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "https://idp/auth" {
		t.Fatalf("unexpected location: %s", got)
	}
}

func TestAuthHandler_CallbackSetsCookieAndRedirects(t *testing.T) {
	h := NewAuthHandler(&authSvcMock{
		handleCallbackFn: func(context.Context, *model.AuthCallbackQuery) (*model.LoginResult, error) {
			return &model.LoginResult{
				SessionID: "s1",
				ExpiresAt: time.Now().Add(time.Hour),
				Redirect:  "https://frontend",
			}, nil
		},
	}, config.CookieConfig{Name: "session_id", Secure: true, HTTPOnly: true, SameSite: "Lax"})
	ctx, rec := testutil.NewGinContext(http.MethodGet, "/v1/auth/callback?code=c1&state=s1", nil)
	h.Callback(ctx)
	if rec.Code != http.StatusFound {
		t.Fatalf("expected 302, got %d", rec.Code)
	}
	if rec.Header().Get("Location") != "https://frontend" {
		t.Fatalf("expected redirect to frontend")
	}
	cookie := rec.Header().Get("Set-Cookie")
	if !strings.Contains(cookie, "session_id=s1") {
		t.Fatalf("session cookie not set")
	}
}

func TestAuthHandler_LogoutBySessionCookie(t *testing.T) {
	h := NewAuthHandler(&authSvcMock{
		logoutFn: func(context.Context, string) error { return nil },
	}, config.CookieConfig{Name: "session_id", Secure: true, HTTPOnly: true, SameSite: "Lax"})
	ctx, rec := testutil.NewGinContext(http.MethodPost, "/v1/auth/logout", nil)
	testutil.AddCookie(ctx, "session_id", "s1")
	h.Logout(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAuthHandler_LogoutBackChannel(t *testing.T) {
	h := NewAuthHandler(&authSvcMock{
		handleBackChannelFn: func(context.Context, string) error { return nil },
	}, config.CookieConfig{Name: "session_id", Secure: true, HTTPOnly: true, SameSite: "Lax"})
	ctx, rec := testutil.NewGinContext(http.MethodPost, "/v1/auth/logout", []byte(`{}`))
	ctx.Request.PostForm = map[string][]string{"logout_token": {"token-1"}}
	ctx.Request.Form = ctx.Request.PostForm
	h.Logout(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAuthHandler_RefreshToken(t *testing.T) {
	h := NewAuthHandler(&authSvcMock{
		refreshSessionFn: func(context.Context, string) (*model.Session, error) {
			return &model.Session{ID: "s1", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
	}, config.CookieConfig{Name: "session_id", Secure: true, HTTPOnly: true, SameSite: "Lax"})
	ctx, rec := testutil.NewGinContext(http.MethodPost, "/api/v1/auth/refresh", nil)
	ctx.Set("session_id", "s1")
	h.RefreshToken(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAuthHandler_GetCurrentUser_Error(t *testing.T) {
	h := NewAuthHandler(&authSvcMock{
		getUserInfoBySessionIDFn: func(context.Context, string) (*model.UserInfo, error) {
			return nil, errors.New("not found")
		},
	}, config.CookieConfig{Name: "session_id", Secure: true, HTTPOnly: true, SameSite: "Lax"})
	ctx, rec := testutil.NewGinContext(http.MethodGet, "/api/v1/auth/user", nil)
	ctx.Set("session_id", "s1")
	h.GetCurrentUser(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"FAILED_TO_GET_USER"`) {
		t.Fatalf("expected FAILED_TO_GET_USER code in response: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"success"`) {
		t.Fatalf("unexpected success field in error response: %s", rec.Body.String())
	}
}

func TestAuthHandler_RefreshToken_Error_NoSuccessField(t *testing.T) {
	h := NewAuthHandler(&authSvcMock{
		refreshSessionFn: func(context.Context, string) (*model.Session, error) {
			return nil, errors.New("refresh failed")
		},
	}, config.CookieConfig{Name: "session_id", Secure: true, HTTPOnly: true, SameSite: "Lax"})
	ctx, rec := testutil.NewGinContext(http.MethodPost, "/api/v1/auth/refresh", nil)
	ctx.Set("session_id", "s1")

	h.RefreshToken(ctx)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"REFRESH_FAILED"`) {
		t.Fatalf("expected REFRESH_FAILED code in response: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"success"`) {
		t.Fatalf("unexpected success field in error response: %s", rec.Body.String())
	}
}
