package api

import (
	"identity-manager/internal/constants"
	"identity-manager/internal/handler"
	"identity-manager/internal/middleware"
	"identity-manager/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	Health            *handler.HealthHandler
	Auth              *handler.AuthHandler
	AuthSvc           service.AuthService
	CoreInternalToken string
	User              *handler.UserHandler
	Session           *handler.SessionHandler
	Internal          *handler.InternalIdentityHandler
}

const (
	v1Prefix     = "/v1"
	legacyPrefix = "/api/v1"
)

func NewRouter(log *logrus.Entry, h *Handlers) *gin.Engine {
	mustValidateHandlers(h)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.AccessLog(log))
	r.Use(middleware.AuthContextStub())

	registerPublicRoutes(r, h)
	registerVersionedRoutes(r, h)
	registerLegacyRoutes(r, h)

	return r
}

func mustValidateHandlers(h *Handlers) {
	if h == nil {
		panic("api handlers must not be nil")
	}
	if h.Health == nil {
		panic("api health handler must not be nil")
	}
	if h.Auth == nil {
		panic("api auth handler must not be nil")
	}
	if h.AuthSvc == nil {
		panic("api auth service must not be nil")
	}
	if h.User == nil {
		panic("api user handler must not be nil")
	}
	if h.Session == nil {
		panic("api session handler must not be nil")
	}
	if h.Internal == nil {
		panic("api internal identity handler must not be nil")
	}
}

func registerPublicRoutes(r *gin.Engine, h *Handlers) {
	r.GET("/live", h.Health.Live)
	r.GET("/ready", h.Health.Health)
	r.GET("/health", h.Health.Health)
	r.GET("/oidc/auth", h.Auth.Login)
	r.GET("/oidc/callback", h.Auth.Callback)
	r.GET("/oidc/logout", h.Auth.Logout)
}

func registerVersionedRoutes(r *gin.Engine, h *Handlers) {
	v1 := r.Group(v1Prefix)
	{
		v1.GET("/auth/login", h.Auth.Login)
		v1.GET("/auth/callback", h.Auth.Callback)
		v1.POST("/auth/logout", h.Auth.Logout)
	}

	registerVersionedAdminRoutes(v1, h)
	registerVersionedAuthedRoutes(v1, h)
	registerInternalRoutes(v1, h)
}

func registerVersionedAdminRoutes(v1 *gin.RouterGroup, h *Handlers) {
	admin := v1.Group("/admin")
	admin.Use(middleware.AuthRequired(h.AuthSvc))
	admin.Use(middleware.RequireAllGroups(h.AuthSvc, constants.GroupAdmin))
	admin.GET("/jwks/status", h.Auth.JWKSStatus)
}

func registerVersionedAuthedRoutes(v1 *gin.RouterGroup, h *Handlers) {
	authed := v1.Group("")
	authed.Use(middleware.AuthRequired(h.AuthSvc))
	authed.GET("/users/me", h.User.Me)
	authed.GET("/sessions/me", h.Session.Me)
	authed.POST("/sessions/refresh", h.Session.Refresh)
	authed.DELETE("/sessions/me", h.Session.Delete)
}

func registerInternalRoutes(v1 *gin.RouterGroup, h *Handlers) {
	internal := v1.Group("/internal")
	internal.Use(middleware.RequireInternalToken(h.CoreInternalToken))
	internal.POST("/identity/resolve", h.Internal.Resolve)
}

func registerLegacyRoutes(r *gin.Engine, h *Handlers) {
	legacy := r.Group(legacyPrefix)
	legacy.Use(middleware.AuthRequired(h.AuthSvc))
	legacy.GET("/auth/user", h.Auth.GetCurrentUser)
	legacy.POST("/auth/refresh", h.Auth.RefreshToken)
	legacy.GET("/health", h.Health.Health)

	admin := legacy.Group("/admin")
	admin.Use(middleware.RequireAllGroups(h.AuthSvc, constants.GroupAdmin))
	admin.GET("/jwks/status", h.Auth.JWKSStatus)
}
