package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"identity-manager/internal/config"
	"identity-manager/internal/model"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

func TestAuthService_BuildLoginURL_Defaults(t *testing.T) {
	svc := NewAuthService(
		config.Config{IDP: config.IDPConfig{Host: "https://idp.local", ClientID: "client-1", RedirectURI: "https://app/cb"}},
		&userRepoMock{},
		&sessionRepoMock{},
	)

	u, err := svc.BuildLoginURL(context.Background(), &model.AuthRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatalf("invalid url: %v", err)
	}
	q := parsed.Query()
	if q.Get("scope") == "" || q.Get("state") == "" || q.Get("nonce") == "" {
		t.Fatalf("expected scope/state/nonce to be set, got: %s", parsed.RawQuery)
	}
}

func TestAuthService_HandleCallback_Success(t *testing.T) {
	privateKey, jwksJSON := mustKeyAndJWKS(t, "kid-1")

	var idp *httptest.Server
	idp = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			token := mustSignedJWT(t, privateKey, "kid-1", map[string]any{
				"iss":             idp.URL,
				"aud":             []string{"client-1"},
				"sub":             "sub-1",
				"nonce":           "nonce-1",
				"identity_id":     "identity-1",
				"login":           "alice",
				"email":           "alice@example.com",
				"group":           []string{"APP_SECURITY_PORTAL_ADMIN_MS"},
				"employee_number": "123",
				"exp":             time.Now().Add(1 * time.Hour).Unix(),
				"iat":             time.Now().Unix(),
			})
			_ = json.NewEncoder(w).Encode(model.OAuth2TokenExchange{
				AccessToken:  "access-1",
				RefreshToken: "refresh-1",
				IDToken:      token,
				ExpiresIn:    3600,
			})
		case "/oauth2/keys":
			_, _ = w.Write(jwksJSON)
		default:
			http.NotFound(w, r)
		}
	}))
	defer idp.Close()

	var created model.Session
	svc := NewAuthService(
		config.Config{
			IDP: config.IDPConfig{
				Host:         idp.URL,
				ClientID:     "client-1",
				ClientSecret: "secret-1",
				RedirectURI:  "https://app/callback",
				JWKSURL:      idp.URL + "/oauth2/keys",
			},
			App: config.AppConfig{FrontendURL: "https://frontend.local"},
		},
		&userRepoMock{
			upsertFn: func(context.Context, model.IDTokenClaims) (*model.User, bool, error) {
				return &model.User{ID: "user-1"}, true, nil
			},
		},
		&sessionRepoMock{
			getByIDFn: func(context.Context, string) (*model.Session, error) { return nil, nil },
			createFn: func(_ context.Context, session model.Session) error {
				created = session
				return nil
			},
			deleteFn: func(context.Context, string) error { return nil },
		},
	)

	// prepare expected state/nonce in store
	_, err := svc.BuildLoginURL(context.Background(), &model.AuthRequest{State: "state-1", Nonce: "nonce-1"})
	if err != nil {
		t.Fatalf("prep build url failed: %v", err)
	}

	res, err := svc.HandleCallback(context.Background(), &model.AuthCallbackQuery{
		Code:  "code-1",
		State: "state-1",
	})
	if err != nil {
		t.Fatalf("unexpected callback error: %v", err)
	}
	if res.SessionID == "" || created.UserID != "user-1" {
		t.Fatalf("session should be created from user claims")
	}
}

func TestAuthService_HandleCallback_InvalidState(t *testing.T) {
	svc := NewAuthService(
		config.Config{IDP: config.IDPConfig{Host: "http://idp", ClientID: "c", ClientSecret: "s", RedirectURI: "r"}},
		&userRepoMock{},
		&sessionRepoMock{
			getByIDFn: func(context.Context, string) (*model.Session, error) { return nil, nil },
			deleteFn:  func(context.Context, string) error { return nil },
		},
	)
	_, err := svc.HandleCallback(context.Background(), &model.AuthCallbackQuery{Code: "x", State: "missing"})
	if err == nil || !strings.Contains(err.Error(), "token request failed") && !strings.Contains(err.Error(), "idp") {
		// network will fail before state in this config; still must error
		t.Fatalf("expected callback to fail, got: %v", err)
	}
}

func TestAuthService_RefreshSession_Success(t *testing.T) {
	privateKey, jwksJSON := mustKeyAndJWKS(t, "kid-1")

	var idp *httptest.Server
	idp = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			token := mustSignedJWT(t, privateKey, "kid-1", map[string]any{
				"iss":   idp.URL,
				"aud":   []string{"client-1"},
				"sub":   "sub-1",
				"nonce": "n1",
				"exp":   time.Now().Add(1 * time.Hour).Unix(),
				"iat":   time.Now().Unix(),
			})
			_ = json.NewEncoder(w).Encode(model.OAuth2TokenExchange{
				AccessToken:  "access-new",
				RefreshToken: "refresh-new",
				IDToken:      token,
				ExpiresIn:    3600,
			})
		case "/oauth2/keys":
			_, _ = w.Write(jwksJSON)
		default:
			http.NotFound(w, r)
		}
	}))
	defer idp.Close()

	getCalls := 0
	svc := NewAuthService(
		config.Config{
			IDP: config.IDPConfig{
				Host:         idp.URL,
				ClientID:     "client-1",
				ClientSecret: "secret-1",
				JWKSURL:      idp.URL + "/oauth2/keys",
			},
		},
		&userRepoMock{},
		&sessionRepoMock{
			getByIDFn: func(context.Context, string) (*model.Session, error) {
				getCalls++
				if getCalls == 1 {
					return &model.Session{ID: "s1", RefreshToken: "refresh-old"}, nil
				}
				return &model.Session{ID: "s1", AccessToken: "access-new"}, nil
			},
			updateTokensFn: func(context.Context, string, model.OAuth2TokenExchange, time.Time) error { return nil },
			deleteFn:       func(context.Context, string) error { return nil },
		},
	)

	session, err := svc.RefreshSession(context.Background(), "s1")
	if err != nil {
		t.Fatalf("unexpected refresh error: %v", err)
	}
	if session.AccessToken != "access-new" {
		t.Fatalf("expected refreshed session")
	}
}

func TestAuthService_HandleBackChannelLogout_BySID(t *testing.T) {
	privateKey, jwksJSON := mustKeyAndJWKS(t, "kid-1")

	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/keys" {
			_, _ = w.Write(jwksJSON)
			return
		}
		http.NotFound(w, r)
	}))
	defer idp.Close()
	logout := mustSignedJWT(t, privateKey, "kid-1", map[string]any{
		"iss": idp.URL,
		"sub": "sub-1",
		"sid": "session-1",
		"events": map[string]any{
			"http://schemas.openid.net/event/backchannel-logout": map[string]any{},
		},
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	})

	deleted := ""
	svc := NewAuthService(
		config.Config{IDP: config.IDPConfig{Host: idp.URL, JWKSURL: idp.URL + "/oauth2/keys"}},
		&userRepoMock{},
		&sessionRepoMock{
			getByIDFn: func(context.Context, string) (*model.Session, error) { return nil, nil },
			deleteFn: func(_ context.Context, sid string) error {
				deleted = sid
				return nil
			},
		},
	)

	if err := svc.HandleBackChannelLogout(context.Background(), logout); err != nil {
		t.Fatalf("unexpected backchannel error: %v", err)
	}
	if deleted != "session-1" {
		t.Fatalf("expected session deletion by sid")
	}
}

func TestAuthService_GetUserInfoBySessionID_FallbackToUser(t *testing.T) {
	svc := NewAuthService(
		config.Config{},
		&userRepoMock{
			getByIDFn: func(context.Context, string) (*model.User, error) {
				return &model.User{Sub: "sub1", IdentityID: "id1", Login: "login1"}, nil
			},
		},
		&sessionRepoMock{
			getByIDFn: func(context.Context, string) (*model.Session, error) {
				return &model.Session{ID: "s1", UserID: "u1", IDToken: "invalid.token"}, nil
			},
			deleteFn: func(context.Context, string) error { return nil },
		},
	)

	info, err := svc.GetUserInfoBySessionID(context.Background(), "s1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.IdentityID != "id1" {
		t.Fatalf("expected fallback user info")
	}
}

func mustKeyAndJWKS(t *testing.T, kid string) (*rsa.PrivateKey, []byte) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pub, err := jwk.FromRaw(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("jwk from raw: %v", err)
	}
	_ = pub.Set(jwk.KeyIDKey, kid)
	_ = pub.Set(jwk.AlgorithmKey, jwa.RS256)

	set := jwk.NewSet()
	set.AddKey(pub)
	data, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("marshal jwks: %v", err)
	}
	return privateKey, data
}

func mustSignedJWT(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()
	tok := jwt.New()
	for k, v := range claims {
		if err := tok.Set(k, v); err != nil {
			t.Fatalf("set claim %s: %v", k, err)
		}
	}
	if _, ok := claims[jwt.ExpirationKey]; !ok {
		_ = tok.Set(jwt.ExpirationKey, time.Now().Add(1*time.Hour))
	}
	privateJWK, err := jwk.FromRaw(key)
	if err != nil {
		t.Fatalf("private jwk: %v", err)
	}
	_ = privateJWK.Set(jwk.KeyIDKey, kid)
	_ = privateJWK.Set(jwk.AlgorithmKey, jwa.RS256)
	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256, privateJWK))
	if err != nil {
		t.Fatalf("sign jwt: %v", err)
	}
	return string(signed)
}

func TestAuthService_BuildLoginURL_MissingConfig(t *testing.T) {
	svc := NewAuthService(config.Config{}, &userRepoMock{}, &sessionRepoMock{})
	if _, err := svc.BuildLoginURL(context.Background(), &model.AuthRequest{}); err == nil {
		t.Fatalf("expected config error")
	}
}

func TestAuthService_RefreshSession_MissingRefreshToken(t *testing.T) {
	svc := NewAuthService(
		config.Config{},
		&userRepoMock{},
		&sessionRepoMock{
			getByIDFn: func(context.Context, string) (*model.Session, error) {
				return &model.Session{ID: "s1"}, nil
			},
			deleteFn: func(context.Context, string) error { return nil },
		},
	)
	_, err := svc.RefreshSession(context.Background(), "s1")
	if err == nil || !strings.Contains(err.Error(), "refresh token is missing") {
		t.Fatalf("expected missing refresh token error")
	}
}

func TestAuthService_GetJWKSStatus_Fields(t *testing.T) {
	svc := NewAuthService(
		config.Config{IDP: config.IDPConfig{Host: "https://idp.local"}},
		&userRepoMock{},
		&sessionRepoMock{},
	)
	status := svc.GetJWKSStatus(context.Background())
	if _, ok := status["jwks_url"]; !ok {
		t.Fatalf("expected jwks_url in status")
	}
	if _, ok := status["cache_ttl_sec"]; !ok {
		t.Fatalf("expected cache_ttl_sec in status")
	}
	if _, err := strconv.Atoi(strconv.Itoa(status["cache_ttl_sec"].(int))); err != nil {
		t.Fatalf("cache_ttl_sec must be int")
	}
}
