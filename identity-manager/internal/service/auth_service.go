package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"identity-manager/internal/config"
	"identity-manager/internal/model"
	"identity-manager/internal/repository"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jws"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type DefaultAuthService struct {
	cfg      config.Config
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	http     *http.Client
	states   *authStateStore
	jwksMu   sync.RWMutex
	jwks     jwk.Set
	jwksAt   time.Time
	jwksTTL  time.Duration
}

func NewAuthService(cfg config.Config, userRepo repository.UserRepository, sessionRepo repository.SessionRepository) *DefaultAuthService {
	return &DefaultAuthService{
		cfg:      cfg,
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
		states:  newAuthStateStore(5 * time.Minute),
		jwksTTL: 10 * time.Minute,
	}
}

func (s *DefaultAuthService) BuildLoginURL(_ context.Context, req *model.AuthRequest) (string, error) {
	if s.cfg.IDP.Host == "" || s.cfg.IDP.ClientID == "" || s.cfg.IDP.RedirectURI == "" {
		return "", fmt.Errorf("idp config is incomplete")
	}

	state := req.State
	if state == "" {
		var err error
		state, err = randomToken(16)
		if err != nil {
			return "", fmt.Errorf("failed to generate state: %w", err)
		}
	}

	nonce := req.Nonce
	if nonce == "" {
		var err error
		nonce, err = randomToken(16)
		if err != nil {
			return "", fmt.Errorf("failed to generate nonce: %w", err)
		}
	}

	scope := strings.TrimSpace(req.Scope)
	if scope == "" {
		scope = "openid profile email offline_access"
	}

	values := url.Values{}
	values.Set("response_type", "code")
	values.Set("client_id", s.cfg.IDP.ClientID)
	values.Set("redirect_uri", s.cfg.IDP.RedirectURI)
	values.Set("scope", scope)
	values.Set("state", state)
	values.Set("nonce", nonce)
	s.states.Put(state, nonce)

	return strings.TrimRight(s.cfg.IDP.Host, "/") + "/oauth2/auth?" + values.Encode(), nil
}

func (s *DefaultAuthService) HandleCallback(ctx context.Context, q *model.AuthCallbackQuery) (*model.LoginResult, error) {
	if q.Error != "" {
		return nil, fmt.Errorf("idp returned error: %s", q.Error)
	}
	if q.Code == "" || q.State == "" {
		return nil, fmt.Errorf("missing code or state")
	}

	token, err := s.exchangeCode(ctx, q.Code)
	if err != nil {
		return nil, err
	}

	if err := s.validateJWTWithJWKS(ctx, token.IDToken, true); err != nil {
		return nil, fmt.Errorf("failed to verify id token signature: %w", err)
	}

	claims, err := parseIDTokenClaims(token.IDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse id token: %w", err)
	}
	expectedNonce, ok := s.states.Consume(q.State)
	if !ok {
		return nil, fmt.Errorf("invalid or expired state")
	}
	if claims.Nonce == "" || claims.Nonce != expectedNonce {
		return nil, fmt.Errorf("nonce validation failed")
	}

	user, _, err := s.userRepo.UpsertFromClaims(ctx, *claims)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert user from claims: %w", err)
	}

	now := time.Now()
	tokenExpiry := now.Add(time.Duration(token.ExpiresIn) * time.Second)
	if token.ExpiresIn <= 0 {
		tokenExpiry = now.Add(8 * time.Hour)
	}

	session := model.Session{
		ID:           uuid.NewString(),
		UserID:       user.ID,
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		IDToken:      token.IDToken,
		TokenExpiry:  tokenExpiry,
		ExpiresAt:    tokenExpiry,
	}

	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	redirect := s.cfg.App.FrontendURL
	if redirect == "" {
		redirect = "/"
	}
	return &model.LoginResult{
		SessionID: session.ID,
		ExpiresAt: session.ExpiresAt,
		Redirect:  redirect,
	}, nil
}

func (s *DefaultAuthService) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	return s.sessionRepo.Delete(ctx, sessionID)
}

func (s *DefaultAuthService) RefreshSession(ctx context.Context, sessionID string) (*model.Session, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session.RefreshToken == "" {
		return nil, fmt.Errorf("refresh token is missing")
	}

	token, err := s.exchangeRefreshToken(ctx, session.RefreshToken)
	if err != nil {
		return nil, err
	}
	if err := s.validateJWTWithJWKS(ctx, token.IDToken, true); err != nil {
		return nil, fmt.Errorf("failed to verify refreshed id token signature: %w", err)
	}

	expiresAt := time.Now().Add(time.Duration(token.ExpiresIn) * time.Second)
	if token.ExpiresIn <= 0 {
		expiresAt = time.Now().Add(8 * time.Hour)
	}

	if err := s.sessionRepo.UpdateTokens(ctx, sessionID, *token, expiresAt); err != nil {
		return nil, err
	}
	return s.sessionRepo.GetByID(ctx, sessionID)
}

func (s *DefaultAuthService) ValidateSession(ctx context.Context, sessionID string) (*model.Session, error) {
	return s.sessionRepo.GetByID(ctx, sessionID)
}

func (s *DefaultAuthService) GetUserInfoBySessionID(ctx context.Context, sessionID string) (*model.UserInfo, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	claims, claimsErr := parseIDTokenClaims(session.IDToken)

	user, userErr := s.userRepo.GetByID(ctx, session.UserID)
	if userErr != nil && claimsErr != nil {
		return nil, userErr
	}

	if claimsErr == nil {
		return &model.UserInfo{
			Sub:            claims.Sub,
			Iss:            claims.Iss,
			Aud:            claims.Aud,
			ACR:            claims.ACR,
			AMR:            claims.AMR,
			Iat:            claims.Iat,
			IdentityID:     claims.IdentityID,
			Login:          claims.Login,
			Email:          claims.Email,
			GivenName:      claims.GivenName,
			FamilyName:     claims.FamilyName,
			Name:           claims.Name,
			Group:          claims.Group,
			WinAccountName: claims.WinAccountName,
			EmployeeNumber: claims.EmployeeNumber,
		}, nil
	}

	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	return &model.UserInfo{
		Sub:            user.Sub,
		IdentityID:     user.IdentityID,
		Login:          user.Login,
		Email:          user.Email,
		GivenName:      user.GivenName,
		FamilyName:     user.FamilyName,
		Name:           user.Name,
		Group:          user.Groups,
		WinAccountName: user.WinAccountName,
		EmployeeNumber: user.EmployeeNumber,
	}, nil
}

func (s *DefaultAuthService) HandleBackChannelLogout(ctx context.Context, logoutToken string) error {
	if err := s.validateJWTWithJWKS(ctx, logoutToken, false); err != nil {
		return fmt.Errorf("failed to verify logout token signature: %w", err)
	}

	claims, err := parseLogoutTokenClaims(logoutToken)
	if err != nil {
		return fmt.Errorf("invalid logout token: %w", err)
	}

	const logoutEventKey = "http://schemas.openid.net/event/backchannel-logout"
	if claims.Events == nil {
		return fmt.Errorf("logout token missing events claim")
	}
	if _, ok := claims.Events[logoutEventKey]; !ok {
		return fmt.Errorf("logout token is not back-channel logout")
	}

	if claims.Sid != "" {
		if err := s.sessionRepo.Delete(ctx, claims.Sid); err != nil && !errors.Is(err, repository.ErrSessionNotFound) {
			return err
		}
		return nil
	}

	if claims.Sub != "" {
		user, err := s.userRepo.GetByIdentityID(ctx, claims.Sub)
		if err != nil {
			if errors.Is(err, repository.ErrUserNotFound) {
				return nil
			}
			return err
		}
		return s.sessionRepo.DeleteByUserID(ctx, user.ID)
	}

	return fmt.Errorf("logout token has neither sid nor sub")
}

func (s *DefaultAuthService) GetJWKSStatus(_ context.Context) map[string]any {
	s.jwksMu.RLock()
	defer s.jwksMu.RUnlock()

	lastUpdate := ""
	if !s.jwksAt.IsZero() {
		lastUpdate = s.jwksAt.Format(time.RFC3339)
	}

	keysCount := 0
	if s.jwks != nil {
		keysCount = s.jwks.Len()
	}

	return map[string]any{
		"jwks_url":        s.jwksURL(),
		"cache_exists":    s.jwks != nil,
		"keys_count":      keysCount,
		"last_updated_at": lastUpdate,
		"cache_ttl_sec":   int(s.jwksTTL.Seconds()),
	}
}

func (s *DefaultAuthService) exchangeCode(ctx context.Context, code string) (*model.OAuth2TokenExchange, error) {
	if s.cfg.IDP.Host == "" || s.cfg.IDP.ClientID == "" || s.cfg.IDP.ClientSecret == "" || s.cfg.IDP.RedirectURI == "" {
		return nil, fmt.Errorf("idp config is incomplete")
	}

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("client_id", s.cfg.IDP.ClientID)
	form.Set("client_secret", s.cfg.IDP.ClientSecret)
	form.Set("redirect_uri", s.cfg.IDP.RedirectURI)

	tokenURL := strings.TrimRight(s.cfg.IDP.Host, "/") + "/oauth2/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("idp returned status %d", resp.StatusCode)
	}

	var token model.OAuth2TokenExchange
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("invalid token response: %w", err)
	}
	return &token, nil
}

func (s *DefaultAuthService) exchangeRefreshToken(ctx context.Context, refreshToken string) (*model.OAuth2TokenExchange, error) {
	if s.cfg.IDP.Host == "" || s.cfg.IDP.ClientID == "" || s.cfg.IDP.ClientSecret == "" {
		return nil, fmt.Errorf("idp config is incomplete")
	}

	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", s.cfg.IDP.ClientID)
	form.Set("client_secret", s.cfg.IDP.ClientSecret)

	tokenURL := strings.TrimRight(s.cfg.IDP.Host, "/") + "/oauth2/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read refresh response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("idp refresh returned status %d", resp.StatusCode)
	}

	var token model.OAuth2TokenExchange
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("invalid refresh token response: %w", err)
	}
	return &token, nil
}

func parseIDTokenClaims(token string) (*model.IDTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid jwt format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode jwt payload: %w", err)
	}

	var claims model.IDTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}
	return &claims, nil
}

func (s *DefaultAuthService) validateJWTWithJWKS(ctx context.Context, rawToken string, requireAudience bool) error {
	set, err := s.getJWKS(ctx)
	if err != nil {
		return err
	}

	opts := []jwt.ParseOption{
		jwt.WithKeySet(set, jws.WithInferAlgorithmFromKey(true)),
		jwt.WithValidate(true),
	}

	if s.cfg.IDP.Host != "" {
		opts = append(opts, jwt.WithIssuer(strings.TrimRight(s.cfg.IDP.Host, "/")))
	}
	if requireAudience && s.cfg.IDP.ClientID != "" {
		opts = append(opts, jwt.WithAudience(s.cfg.IDP.ClientID))
	}

	if _, err := jwt.ParseString(rawToken, opts...); err != nil {
		return fmt.Errorf("jwt validation failed: %w", err)
	}

	return nil
}

func (s *DefaultAuthService) getJWKS(ctx context.Context) (jwk.Set, error) {
	now := time.Now()

	s.jwksMu.RLock()
	if s.jwks != nil && now.Sub(s.jwksAt) < s.jwksTTL {
		cached := s.jwks
		s.jwksMu.RUnlock()
		return cached, nil
	}
	s.jwksMu.RUnlock()

	s.jwksMu.Lock()
	defer s.jwksMu.Unlock()

	if s.jwks != nil && now.Sub(s.jwksAt) < s.jwksTTL {
		return s.jwks, nil
	}

	url := s.jwksURL()
	if url == "" {
		return nil, fmt.Errorf("jwks url is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create jwks request: %w", err)
	}

	resp, err := s.http.Do(req)
	if err != nil {
		if s.jwks != nil {
			return s.jwks, nil
		}
		return nil, fmt.Errorf("failed to fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if s.jwks != nil {
			return s.jwks, nil
		}
		return nil, fmt.Errorf("jwks endpoint returned status %d", resp.StatusCode)
	}

	set, err := jwk.ParseReader(resp.Body)
	if err != nil {
		if s.jwks != nil {
			return s.jwks, nil
		}
		return nil, fmt.Errorf("failed to parse jwks: %w", err)
	}

	s.jwks = set
	s.jwksAt = time.Now()
	return s.jwks, nil
}

func (s *DefaultAuthService) jwksURL() string {
	if strings.TrimSpace(s.cfg.IDP.JWKSURL) != "" {
		return s.cfg.IDP.JWKSURL
	}
	if strings.TrimSpace(s.cfg.IDP.Host) == "" {
		return ""
	}
	return strings.TrimRight(s.cfg.IDP.Host, "/") + "/oauth2/keys"
}

func parseLogoutTokenClaims(token string) (*model.LogoutTokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid jwt format")
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to decode jwt payload: %w", err)
	}

	var claims model.LogoutTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}
	return &claims, nil
}

func randomToken(bytesLen int) (string, error) {
	buf := make([]byte, bytesLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
