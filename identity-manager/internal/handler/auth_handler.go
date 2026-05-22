package handler

import (
	"net/http"
	"strings"
	"time"

	"identity-manager/internal/config"
	"identity-manager/internal/model"
	"identity-manager/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth   service.AuthService
	cookie config.CookieConfig
}

func NewAuthHandler(auth service.AuthService, cookie config.CookieConfig) *AuthHandler {
	return &AuthHandler{auth: auth, cookie: cookie}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req model.AuthRequest
	// Login query params are optional and validated downstream by auth service.
	_ = c.ShouldBindQuery(&req)

	url, err := h.auth.BuildLoginURL(c.Request.Context(), &req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "AUTH_LOGIN_FAILED", "failed to build login url")
		return
	}
	c.Redirect(http.StatusFound, url)
}

func (h *AuthHandler) Callback(c *gin.Context) {
	var query model.AuthCallbackQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, "BAD_REQUEST", "invalid callback query")
		return
	}

	login, err := h.auth.HandleCallback(c.Request.Context(), &query)
	if err != nil {
		writeUnauthorized(c, "AUTH_CALLBACK_FAILED", "authentication callback failed")
		return
	}

	h.setSessionCookie(c, login.SessionID, login.ExpiresAt)
	c.Redirect(http.StatusFound, login.Redirect)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	if err := c.Request.ParseForm(); err == nil {
		logoutToken := c.Request.Form.Get("logout_token")
		if logoutToken != "" {
			if err := h.auth.HandleBackChannelLogout(c.Request.Context(), logoutToken); err != nil {
				writeUnauthorized(c, "INVALID_LOGOUT_TOKEN", "logout token validation failed")
				return
			}
			c.Status(http.StatusOK)
			return
		}
	}

	sessionID := c.GetString("session_id")
	if sessionID == "" {
		sessionID, _ = c.Cookie(cookieName(h.cookie))
	}
	if err := h.auth.Logout(c.Request.Context(), sessionID); err != nil {
		writeUnauthorized(c, "LOGOUT_FAILED", "failed to logout session")
		return
	}
	c.SetCookie(cookieName(h.cookie), "", -1, "/", "", h.cookie.Secure, h.cookie.HTTPOnly)
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func (h *AuthHandler) JWKSStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"jwks": h.auth.GetJWKSStatus(c.Request.Context()),
	})
}

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userInfo, err := h.auth.GetUserInfoBySessionID(c.Request.Context(), c.GetString("session_id"))
	if err != nil {
		writeUnauthorized(c, "FAILED_TO_GET_USER", "failed to retrieve current user")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "user retrieved",
		"user":    userInfo,
	})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	session, err := h.auth.RefreshSession(c.Request.Context(), c.GetString("session_id"))
	if err != nil {
		writeUnauthorized(c, "REFRESH_FAILED", "failed to refresh session")
		return
	}

	h.setSessionCookie(c, session.ID, session.ExpiresAt)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "token refreshed",
		"expires": session.ExpiresAt,
	})
}

func cookieName(cfg config.CookieConfig) string {
	if strings.TrimSpace(cfg.Name) == "" {
		return "session_id"
	}
	return cfg.Name
}

func parseSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	case "lax":
		fallthrough
	default:
		return http.SameSiteLaxMode
	}
}

func (h *AuthHandler) setSessionCookie(c *gin.Context, sessionID string, expiresAt time.Time) {
	c.SetSameSite(parseSameSite(h.cookie.SameSite))

	maxAgeSeconds := int(time.Until(expiresAt).Seconds())
	if maxAgeSeconds < 0 {
		maxAgeSeconds = 0
	}

	c.SetCookie(cookieName(h.cookie), sessionID, maxAgeSeconds, "/", "", h.cookie.Secure, h.cookie.HTTPOnly)
}
