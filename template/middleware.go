package template

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"strings"
)

type Logger interface {
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

type Middleware struct {
	cfg    Config
	idm    Resolver
	legacy Resolver
	logger Logger
}

func NewMiddleware(cfg Config, idm Resolver, legacy Resolver, logger Logger) *Middleware {
	cfg.Normalize()
	return &Middleware{
		cfg:    cfg,
		idm:    idm,
		legacy: legacy,
		logger: logger,
	}
}

func (m *Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID := readSessionID(r, m.cfg.SessionCookieName)
		if sessionID == "" {
			writeUnauthorized(w)
			return
		}

		identity, err := m.resolveIdentity(r.Context(), sessionID)
		if err != nil {
			writeUnauthorized(w)
			return
		}

		next.ServeHTTP(w, AttachIdentity(r, identity))
	})
}

func (m *Middleware) resolveIdentity(ctx context.Context, sessionID string) (*Identity, error) {
	if !m.cfg.Enabled {
		return m.resolveLegacy(ctx, sessionID)
	}

	switch strings.ToLower(strings.TrimSpace(m.cfg.PrimarySource)) {
	case PrimarySourceIDM:
		return m.resolvePrimaryIDM(ctx, sessionID)
	default:
		return m.resolvePrimaryLegacy(ctx, sessionID)
	}
}

func (m *Middleware) resolvePrimaryIDM(ctx context.Context, sessionID string) (*Identity, error) {
	idmIdentity, idmErr := m.resolveIDM(ctx, sessionID)

	if m.cfg.DualReadEnabled {
		legacyIdentity, legacyErr := m.resolveLegacy(ctx, sessionID)
		m.logDiff("idm", idmIdentity, idmErr, legacyIdentity, legacyErr)
	}

	if idmErr == nil {
		return idmIdentity, nil
	}

	if m.cfg.FailOpen {
		legacyIdentity, legacyErr := m.resolveLegacy(ctx, sessionID)
		if legacyErr == nil {
			m.logWarn("idm failed, fallback to legacy", "session_id", sessionID, "error", idmErr.Error())
			return legacyIdentity, nil
		}
	}

	return nil, idmErr
}

func (m *Middleware) resolvePrimaryLegacy(ctx context.Context, sessionID string) (*Identity, error) {
	legacyIdentity, legacyErr := m.resolveLegacy(ctx, sessionID)

	if m.cfg.DualReadEnabled {
		idmIdentity, idmErr := m.resolveIDM(ctx, sessionID)
		m.logDiff("legacy", legacyIdentity, legacyErr, idmIdentity, idmErr)
	}

	if legacyErr == nil {
		return legacyIdentity, nil
	}

	if m.cfg.FailOpen {
		idmIdentity, idmErr := m.resolveIDM(ctx, sessionID)
		if idmErr == nil {
			m.logWarn("legacy failed, fallback to idm", "session_id", sessionID, "error", legacyErr.Error())
			return idmIdentity, nil
		}
	}

	return nil, legacyErr
}

func (m *Middleware) resolveIDM(ctx context.Context, sessionID string) (*Identity, error) {
	if m.idm == nil {
		return nil, errResolverNotConfigured("idm")
	}
	return m.idm.Resolve(ctx, sessionID)
}

func (m *Middleware) resolveLegacy(ctx context.Context, sessionID string) (*Identity, error) {
	if m.legacy == nil {
		return nil, errResolverNotConfigured("legacy")
	}
	return m.legacy.Resolve(ctx, sessionID)
}

func (m *Middleware) logDiff(primary string, primaryID *Identity, primaryErr error, secondaryID *Identity, secondaryErr error) {
	if primaryErr != nil || secondaryErr != nil {
		m.logInfo(
			"dual-read completed with errors",
			"primary", primary,
			"primary_error", errorString(primaryErr),
			"secondary_error", errorString(secondaryErr),
		)
		return
	}

	if identitiesEqual(primaryID, secondaryID) {
		m.logInfo("dual-read match", "primary", primary, "user_id", primaryID.UserID, "session_id", primaryID.SessionID)
		return
	}

	primaryJSON, _ := json.Marshal(primaryID)
	secondaryJSON, _ := json.Marshal(secondaryID)
	m.logWarn(
		"dual-read mismatch",
		"primary", primary,
		"primary_identity", string(primaryJSON),
		"secondary_identity", string(secondaryJSON),
	)
}

func identitiesEqual(a, b *Identity) bool {
	if a == nil || b == nil {
		return a == b
	}

	ag := slices.Clone(a.Groups)
	bg := slices.Clone(b.Groups)
	slices.Sort(ag)
	slices.Sort(bg)

	return a.SessionID == b.SessionID &&
		a.UserID == b.UserID &&
		a.IdentityID == b.IdentityID &&
		a.Login == b.Login &&
		reflect.DeepEqual(ag, bg)
}

func readSessionID(r *http.Request, cookieName string) string {
	if cookie, err := r.Cookie(cookieName); err == nil && strings.TrimSpace(cookie.Value) != "" {
		return strings.TrimSpace(cookie.Value)
	}

	headerValue := strings.TrimSpace(r.Header.Get("X-Session-ID"))
	if headerValue != "" {
		return headerValue
	}

	return ""
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"code":"UNAUTHORIZED","message":"session is invalid or missing"}`))
}

func (m *Middleware) logInfo(msg string, args ...any) {
	if m.logger != nil {
		m.logger.Info(msg, args...)
	}
}

func (m *Middleware) logWarn(msg string, args ...any) {
	if m.logger != nil {
		m.logger.Warn(msg, args...)
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
