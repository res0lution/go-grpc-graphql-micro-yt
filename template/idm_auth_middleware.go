package middleware

import (
	"net/http"
	"strings"
	"portal-core/internal/model"
	"portal-core/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)
const (
	PrimarySourceLegacy = "legacy"
	PrimarySourceIDM    = "idm"
)
type IDMIdentity struct {
	SessionID  string
	UserID     string
	IdentityID string
	Login      string
	Groups     []string
}
type IDMResolver interface {
	Resolve(ctx context.Context, sessionID string) (*IDMIdentity, error)
}
type IdentityAuthOptions struct {
	Logger      *logrus.Entry
	AuthService *service.AuthService // legacy path (ValidateSession/GetUserInfoBySessionID)
	IDMResolver IDMResolver          // new path via client
	DualReadEnabled bool
	PrimarySource   string // "legacy" | "idm"
	FailOpen        bool
	CookieName      string
}
type IdentityAuthMiddleware struct {
	opts IdentityAuthOptions
}
func NewIdentityAuthMiddleware(opts IdentityAuthOptions) *IdentityAuthMiddleware {
	if opts.CookieName == "" {
		opts.CookieName = "session_id"
	}
	if opts.PrimarySource == "" {
		opts.PrimarySource = PrimarySourceLegacy
	}
	return &IdentityAuthMiddleware{opts: opts}
}
func (m *IdentityAuthMiddleware) Handle(ctx *gin.Context) {
	sessionID, err := m.readSessionID(ctx)
	if err != nil {
		m.abortUnauthorized(ctx, "Not authenticated", "Session cookie not found")
		return
	}
	session, userInfo, err := m.resolve(ctx, sessionID)
	if err != nil {
		m.abortUnauthorized(ctx, "Invalid session", "Session is invalid or expired")
		return
	}
	// Backward-compatible keys for existing handlers/RBAC middleware.
	ctx.Set("session", session)
	ctx.Set("session_id", session.ID)
	ctx.Set("session_user_id", session.UserID)
	ctx.Set("session_user_info", userInfo)
	ctx.Next()
}
func (m *IdentityAuthMiddleware) resolve(ctx *gin.Context, sessionID string) (*model.Session, model.UserInfo, error) {
	primary := strings.ToLower(strings.TrimSpace(m.opts.PrimarySource))
	switch primary {
	case PrimarySourceIDM:
		return m.resolvePrimaryIDM(ctx, sessionID)
	default:
		return m.resolvePrimaryLegacy(ctx, sessionID)
	}
}
func (m *IdentityAuthMiddleware) resolvePrimaryLegacy(ctx *gin.Context, sessionID string) (*model.Session, model.UserInfo, error) {
	session, userInfo, err := m.resolveLegacy(ctx, sessionID)
	if m.opts.DualReadEnabled {
		_, idmInfo, idmErr := m.resolveIDM(ctx, sessionID)
		m.logDualRead("legacy", userInfo, err, idmInfo, idmErr)
	}
	if err == nil {
		return session, userInfo, nil
	}
	if m.opts.FailOpen {
		idmSession, idmInfo, idmErr := m.resolveIDM(ctx, sessionID)
		if idmErr == nil {
			m.logWarn("legacy failed, fallback to idm", "session_id", sessionID, "error", err.Error())
			return idmSession, idmInfo, nil
		}
	}
	return nil, model.UserInfo{}, err
}
func (m *IdentityAuthMiddleware) resolvePrimaryIDM(ctx *gin.Context, sessionID string) (*model.Session, model.UserInfo, error) {
	session, userInfo, err := m.resolveIDM(ctx, sessionID)
	if m.opts.DualReadEnabled {
		_, legacyInfo, legacyErr := m.resolveLegacy(ctx, sessionID)
		m.logDualRead("idm", userInfo, err, legacyInfo, legacyErr)
	}
	if err == nil {
		return session, userInfo, nil
	}
	if m.opts.FailOpen {
		legacySession, legacyInfo, legacyErr := m.resolveLegacy(ctx, sessionID)
		if legacyErr == nil {
			m.logWarn("idm failed, fallback to legacy", "session_id", sessionID, "error", err.Error())
			return legacySession, legacyInfo, nil
		}
	}
	return nil, model.UserInfo{}, err
}
func (m *IdentityAuthMiddleware) resolveLegacy(ctx *gin.Context, sessionID string) (*model.Session, model.UserInfo, error) {
	session, err := m.opts.AuthService.ValidateSession(ctx.Request.Context(), sessionID)
	if err != nil {
		return nil, model.UserInfo{}, err
	}
	userInfo, err := m.opts.AuthService.GetUserInfoBySessionID(ctx.Request.Context(), sessionID)
	if err != nil {
		return nil, model.UserInfo{}, err
	}
	return session, *userInfo, nil
}
func (m *IdentityAuthMiddleware) resolveIDM(ctx *gin.Context, sessionID string) (*model.Session, model.UserInfo, error) {
	id, err := m.opts.IDMResolver.Resolve(ctx.Request.Context(), sessionID)
	if err != nil {
		return nil, model.UserInfo{}, err
	}
	s := &model.Session{
		ID:     id.SessionID,
		UserID: id.UserID,
	}
	// Minimal compatible payload for existing RBAC middleware:
	// RequireAnyGroup/RequireAllGroups depend on UserInfo.Group.
	u := model.UserInfo{
		Sub:        id.UserID,
		IdentityID: id.IdentityID,
		Login:      id.Login,
		Group:      id.Groups,
	}
	return s, u, nil
}
func (m *IdentityAuthMiddleware) readSessionID(ctx *gin.Context) (string, error) {
	c, err := ctx.Request.Cookie(m.opts.CookieName)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(c.Value) == "" {
		return "", errors.New("empty session cookie")
	}
	return c.Value, nil
}
func (m *IdentityAuthMiddleware) abortUnauthorized(ctx *gin.Context, errText, msg string) {
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"success": false,
		"error":   errText,
		"message": msg,
	})
}
func (m *IdentityAuthMiddleware) logDualRead(primary string, a model.UserInfo, aErr error, b model.UserInfo, bErr error) {
	if m.opts.Logger == nil {
		return
	}
	if aErr != nil || bErr != nil {
		m.opts.Logger.WithFields(logrus.Fields{
			"primary":         primary,
			"primary_error":   errString(aErr),
			"secondary_error": errString(bErr),
		}).Info("identity dual-read completed with errors")
		return
	}
	if a.Sub == b.Sub && a.IdentityID == b.IdentityID && slices.Equal(a.Group, b.Group) {
		m.opts.Logger.WithField("primary", primary).Debug("identity dual-read match")
		return
	}
	m.opts.Logger.WithFields(logrus.Fields{
		"primary": primary,
		"a_sub":   a.Sub,
		"b_sub":   b.Sub,
		"a_gid":   a.IdentityID,
		"b_gid":   b.IdentityID,
	}).Warn("identity dual-read mismatch")
}