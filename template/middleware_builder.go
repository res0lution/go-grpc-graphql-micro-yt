package api
import (
	"portal-core/internal/config"
	"portal-core/internal/middleware"
	"portal-core/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)
type IdentityResolver interface {
	Resolve(ctx context.Context, sessionID string) (*Identity, error)
}
// buildAuthMiddleware returns:
// - legacy middleware.AuthRequired(...) when feature is off
// - new identity-manager middleware when feature is on
func buildAuthMiddleware(
	log *logrus.Entry,
	cfg *config.Config,
	authSvc *service.AuthService,
	idmResolver IdentityResolver, // nil allowed when flag off
) gin.HandlerFunc {
	if cfg == nil || authSvc == nil {
		panic("buildAuthMiddleware: cfg and authSvc are required")
	}
	// default path: old middleware
	if !cfg.IdentityManager.Enabled {
		log.Info("auth middleware: using legacy AuthRequired")
		return middleware.AuthRequired(authSvc)
	}
	// feature enabled: use new middleware via identity-manager client
	if idmResolver == nil {
		// fail-safe: don't kill app, fallback to legacy
		log.Warn("auth middleware: idm enabled but resolver is nil, fallback to legacy")
		return middleware.AuthRequired(authSvc)
	}
	log.WithFields(logrus.Fields{
		"dual_read":      cfg.IdentityManager.DualReadEnabled,
		"primary_source": cfg.IdentityManager.PrimarySource,
		"fail_open":      cfg.IdentityManager.FailOpen,
	}).Info("auth middleware: using identity-manager resolver")
	return middleware.NewIdentityAuthMiddleware(middleware.IdentityAuthOptions{
		Logger:          log,
		AuthService:     authSvc,      // legacy resolver for fallback/dual-read
		IDMResolver:     idmResolver,  // new resolver client
		DualReadEnabled: cfg.IdentityManager.DualReadEnabled,
		PrimarySource:   cfg.IdentityManager.PrimarySource, // "legacy" | "idm"
		FailOpen:        cfg.IdentityManager.FailOpen,
		CookieName:      cfg.IdentityManager.SessionCookieName,
	}).Handle
}

authMW := buildAuthMiddleware(log, cfg, h.Auth.AuthService, idmResolver)
protected := r.Group("")
protected.Use(authMW)
v1 := protected.Group("/api/v1")