package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"identity-manager/internal/model"
	"identity-manager/internal/testutil"

	"github.com/gin-gonic/gin"
)

type authSvcMock struct {
	validateSessionFn        func(context.Context, string) (*model.Session, error)
	getUserInfoBySessionIDFn func(context.Context, string) (*model.UserInfo, error)
}

func (m *authSvcMock) BuildLoginURL(context.Context, *model.AuthRequest) (string, error) {
	panic("not used")
}
func (m *authSvcMock) HandleCallback(context.Context, *model.AuthCallbackQuery) (*model.LoginResult, error) {
	panic("not used")
}
func (m *authSvcMock) RefreshSession(context.Context, string) (*model.Session, error) {
	panic("not used")
}
func (m *authSvcMock) ValidateSession(ctx context.Context, sessionID string) (*model.Session, error) {
	return m.validateSessionFn(ctx, sessionID)
}
func (m *authSvcMock) GetUserInfoBySessionID(ctx context.Context, sessionID string) (*model.UserInfo, error) {
	return m.getUserInfoBySessionIDFn(ctx, sessionID)
}
func (m *authSvcMock) Logout(context.Context, string) error { panic("not used") }
func (m *authSvcMock) HandleBackChannelLogout(context.Context, string) error {
	panic("not used")
}
func (m *authSvcMock) GetJWKSStatus(context.Context) map[string]any { return nil }

func TestAuthRequired_NoCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, rec := testutil.NewGinContext(http.MethodGet, "/", nil)

	AuthRequired(&authSvcMock{})(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"UNAUTHORIZED"`) {
		t.Fatalf("expected UNAUTHORIZED code, got %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), `"success"`) {
		t.Fatalf("unexpected success field, got %s", rec.Body.String())
	}
}

func TestAuthRequired_SuccessSetsContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, rec := testutil.NewGinContext(http.MethodGet, "/", nil)
	testutil.AddCookie(ctx, "session_id", "s1")

	AuthRequired(&authSvcMock{
		validateSessionFn: func(context.Context, string) (*model.Session, error) {
			return &model.Session{ID: "s1", UserID: "u1"}, nil
		},
		getUserInfoBySessionIDFn: func(context.Context, string) (*model.UserInfo, error) {
			return &model.UserInfo{Sub: "sub1", Group: []string{"G1"}}, nil
		},
	})(ctx)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ctx.GetString("session_id") != "s1" {
		t.Fatalf("session_id not set")
	}
}

func TestAuthRequired_InvalidSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, rec := testutil.NewGinContext(http.MethodGet, "/", nil)
	testutil.AddCookie(ctx, "session_id", "s1")
	AuthRequired(&authSvcMock{
		validateSessionFn: func(context.Context, string) (*model.Session, error) {
			return nil, errors.New("expired")
		},
		getUserInfoBySessionIDFn: func(context.Context, string) (*model.UserInfo, error) {
			return nil, nil
		},
	})(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"INVALID_SESSION"`) {
		t.Fatalf("expected INVALID_SESSION code, got %s", rec.Body.String())
	}
}

func TestAuthRequired_InvalidUserInfo(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, rec := testutil.NewGinContext(http.MethodGet, "/", nil)
	testutil.AddCookie(ctx, "session_id", "s1")
	AuthRequired(&authSvcMock{
		validateSessionFn: func(context.Context, string) (*model.Session, error) {
			return &model.Session{ID: "s1", UserID: "u1"}, nil
		},
		getUserInfoBySessionIDFn: func(context.Context, string) (*model.UserInfo, error) {
			return nil, errors.New("bad claims")
		},
	})(ctx)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"USER_INFO_UNAVAILABLE"`) {
		t.Fatalf("expected USER_INFO_UNAVAILABLE code, got %s", rec.Body.String())
	}
}

func TestRequireAnyGroup_Forbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, rec := testutil.NewGinContext(http.MethodGet, "/", nil)
	ctx.Set("session_user_info", model.UserInfo{Group: []string{"G1"}})

	RequireAnyGroup(&authSvcMock{}, "ADMIN")(ctx)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"code":"ACCESS_DENIED"`) {
		t.Fatalf("expected ACCESS_DENIED code, got %s", rec.Body.String())
	}
}

func TestRequireAllGroup_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, rec := testutil.NewGinContext(http.MethodGet, "/", nil)
	ctx.Set("session_user_info", model.UserInfo{Group: []string{"A", "B"}})

	RequireAllGroups(&authSvcMock{}, "A", "B")(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestMustGetUserInfo_PanicWhenMissing(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic")
		}
	}()
	gin.SetMode(gin.TestMode)
	ctx, _ := testutil.NewGinContext(http.MethodGet, "/", nil)
	_ = MustGetUserInfo(ctx)
}

func TestOptionalAuth_SetsContextWhenValid(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := testutil.NewGinContext(http.MethodGet, "/", nil)
	testutil.AddCookie(ctx, "session_id", "s1")

	OptionalAuth(&authSvcMock{
		validateSessionFn: func(context.Context, string) (*model.Session, error) {
			return &model.Session{ID: "s1", UserID: "u1", ExpiresAt: time.Now().Add(time.Hour)}, nil
		},
		getUserInfoBySessionIDFn: func(context.Context, string) (*model.UserInfo, error) {
			return &model.UserInfo{Sub: "sub1"}, nil
		},
	})(ctx)

	if ctx.GetString("session_id") != "s1" {
		t.Fatalf("expected session_id context")
	}
}
