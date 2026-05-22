package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"identity-manager/internal/config"
	"identity-manager/internal/handler"
	"identity-manager/internal/model"

	"github.com/sirupsen/logrus"
)

type authSvcMock struct {
	groups []string
}

func (m *authSvcMock) BuildLoginURL(context.Context, *model.AuthRequest) (string, error) {
	return "https://idp/auth", nil
}
func (m *authSvcMock) HandleCallback(context.Context, *model.AuthCallbackQuery) (*model.LoginResult, error) {
	return &model.LoginResult{SessionID: "s1"}, nil
}
func (m *authSvcMock) RefreshSession(context.Context, string) (*model.Session, error) {
	return &model.Session{ID: "s1"}, nil
}
func (m *authSvcMock) ValidateSession(context.Context, string) (*model.Session, error) {
	return &model.Session{ID: "s1", UserID: "u1"}, nil
}
func (m *authSvcMock) GetUserInfoBySessionID(context.Context, string) (*model.UserInfo, error) {
	return &model.UserInfo{Group: m.groups}, nil
}
func (m *authSvcMock) Logout(context.Context, string) error { return nil }
func (m *authSvcMock) HandleBackChannelLogout(context.Context, string) error { return nil }
func (m *authSvcMock) GetJWKSStatus(context.Context) map[string]any { return map[string]any{"ok": true} }

type userSvcMock struct{}

func (m *userSvcMock) GetCurrentUser(context.Context, string) (*model.User, error) {
	return &model.User{ID: "u1"}, nil
}
func (m *userSvcMock) GetByID(context.Context, string) (*model.User, error) { return &model.User{}, nil }
func (m *userSvcMock) GetByIdentityID(context.Context, string) (*model.User, error) {
	return &model.User{}, nil
}
func (m *userSvcMock) GetAllActive(context.Context, int, int) ([]model.User, int, error) {
	return nil, 0, nil
}

type sessionSvcMock struct{}

func (m *sessionSvcMock) GetCurrentSession(context.Context, string) (*model.Session, error) {
	return &model.Session{ID: "s1"}, nil
}
func (m *sessionSvcMock) RefreshSession(context.Context, string) (*model.Session, error) {
	return &model.Session{ID: "s1"}, nil
}
func (m *sessionSvcMock) RevokeSession(context.Context, string) error { return nil }
func (m *sessionSvcMock) ResolveIdentity(context.Context, string) (*model.ResolvedIdentity, error) {
	return &model.ResolvedIdentity{
		Identity: model.IdentityContext{SessionID: "s1", UserID: "u1"},
		UserInfo: model.UserInfo{Sub: "u1"},
	}, nil
}

type pingerMock struct{}

func (m *pingerMock) Ping(context.Context) error { return nil }

func buildTestRouter(groups []string) *http.ServeMux {
	authSvc := &authSvcMock{groups: groups}
	sessionSvc := &sessionSvcMock{}
	h := &Handlers{
		Health:            handler.NewHealthHandler(&pingerMock{}),
		Auth:              handler.NewAuthHandler(authSvc, config.CookieConfig{Name: "session_id", Secure: true, HTTPOnly: true, SameSite: "Lax"}),
		AuthSvc:           authSvc,
		CoreInternalToken: "core-token",
		User:              handler.NewUserHandler(&userSvcMock{}),
		Session:           handler.NewSessionHandler(sessionSvc),
		Internal:          handler.NewInternalIdentityHandler(sessionSvc),
	}
	r := NewRouter(logrus.NewEntry(logrus.New()), h)
	mux := http.NewServeMux()
	mux.Handle("/", r)
	return mux
}

func TestRouterRouteContracts(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		method         string
		path           string
		body           string
		contentType    string
		sessionCookie  string
		bearerToken    string
		groups         []string
		expectedStatus int
	}{
		{
			name:           "v1 admin requires auth",
			method:         http.MethodGet,
			path:           "/v1/admin/jwks/status",
			groups:         []string{"APP_SECURITY_PORTAL_ADMIN_MS"},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "v1 admin forbidden for non admin group",
			method:         http.MethodGet,
			path:           "/v1/admin/jwks/status",
			sessionCookie:  "s1",
			groups:         []string{"APP_SECURITY_PORTAL_USER_MS"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "v1 admin ok for admin group",
			method:         http.MethodGet,
			path:           "/v1/admin/jwks/status",
			sessionCookie:  "s1",
			groups:         []string{"APP_SECURITY_PORTAL_ADMIN_MS"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "internal resolve requires bearer",
			method:         http.MethodPost,
			path:           "/v1/internal/identity/resolve",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "internal resolve rejects invalid bearer",
			method:         http.MethodPost,
			path:           "/v1/internal/identity/resolve",
			bearerToken:    "wrong-token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "internal resolve ok with service token",
			method:         http.MethodPost,
			path:           "/v1/internal/identity/resolve",
			body:           `{"session_id":"s1"}`,
			contentType:    "application/json",
			bearerToken:    "core-token",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "legacy auth user requires auth",
			method:         http.MethodGet,
			path:           "/api/v1/auth/user",
			groups:         []string{"APP_SECURITY_PORTAL_ADMIN_MS"},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "legacy admin forbidden for non admin group",
			method:         http.MethodGet,
			path:           "/api/v1/admin/jwks/status",
			sessionCookie:  "s1",
			groups:         []string{"APP_SECURITY_PORTAL_USER_MS"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "legacy admin ok for admin group",
			method:         http.MethodGet,
			path:           "/api/v1/admin/jwks/status",
			sessionCookie:  "s1",
			groups:         []string{"APP_SECURITY_PORTAL_ADMIN_MS"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "v1 logout only allows post",
			method:         http.MethodGet,
			path:           "/v1/auth/logout",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "live route available",
			method:         http.MethodGet,
			path:           "/live",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "ready route available",
			method:         http.MethodGet,
			path:           "/ready",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			router := buildTestRouter(tc.groups)
			reqBody := strings.NewReader(tc.body)
			req := httptest.NewRequest(tc.method, tc.path, reqBody)
			if tc.contentType != "" {
				req.Header.Set("Content-Type", tc.contentType)
			}
			if tc.bearerToken != "" {
				req.Header.Set("Authorization", "Bearer "+tc.bearerToken)
			}
			if tc.sessionCookie != "" {
				req.AddCookie(&http.Cookie{Name: "session_id", Value: tc.sessionCookie})
			}
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)
			if rec.Code != tc.expectedStatus {
				t.Fatalf("expected status %d, got %d", tc.expectedStatus, rec.Code)
			}
		})
	}
}

func TestNewRouter_ValidateHandlers(t *testing.T) {
	t.Parallel()

	base := &Handlers{
		Health:            handler.NewHealthHandler(&pingerMock{}),
		Auth:              handler.NewAuthHandler(&authSvcMock{groups: []string{"APP_SECURITY_PORTAL_ADMIN_MS"}}, config.CookieConfig{Name: "session_id", Secure: true, HTTPOnly: true, SameSite: "Lax"}),
		AuthSvc:           &authSvcMock{groups: []string{"APP_SECURITY_PORTAL_ADMIN_MS"}},
		CoreInternalToken: "core-token",
		User:              handler.NewUserHandler(&userSvcMock{}),
		Session:           handler.NewSessionHandler(&sessionSvcMock{}),
		Internal:          handler.NewInternalIdentityHandler(&sessionSvcMock{}),
	}

	tests := []struct {
		name string
		make func() *Handlers
	}{
		{
			name: "nil handlers",
			make: func() *Handlers { return nil },
		},
		{
			name: "nil health handler",
			make: func() *Handlers {
				h := *base
				h.Health = nil
				return &h
			},
		},
		{
			name: "nil auth handler",
			make: func() *Handlers {
				h := *base
				h.Auth = nil
				return &h
			},
		},
		{
			name: "nil auth service",
			make: func() *Handlers {
				h := *base
				h.AuthSvc = nil
				return &h
			},
		},
		{
			name: "nil user handler",
			make: func() *Handlers {
				h := *base
				h.User = nil
				return &h
			},
		},
		{
			name: "nil session handler",
			make: func() *Handlers {
				h := *base
				h.Session = nil
				return &h
			},
		},
		{
			name: "nil internal handler",
			make: func() *Handlers {
				h := *base
				h.Internal = nil
				return &h
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if recover() == nil {
					t.Fatal("expected panic, got nil")
				}
			}()

			_ = NewRouter(logrus.NewEntry(logrus.New()), tt.make())
		})
	}
}
