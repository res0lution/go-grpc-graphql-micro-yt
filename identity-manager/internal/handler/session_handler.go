package handler

import (
	"net/http"

	"identity-manager/internal/service"

	"github.com/gin-gonic/gin"
)

type SessionHandler struct {
	sessions service.SessionService
}

func NewSessionHandler(sessions service.SessionService) *SessionHandler {
	return &SessionHandler{sessions: sessions}
}

func (h *SessionHandler) Me(c *gin.Context) {
	session, err := h.sessions.GetCurrentSession(c.Request.Context(), c.GetString("session_id"))
	if err != nil {
		writeUnauthorized(c, "UNAUTHORIZED", "Session is invalid or expired")
		return
	}
	c.JSON(http.StatusOK, session)
}

func (h *SessionHandler) Refresh(c *gin.Context) {
	session, err := h.sessions.RefreshSession(c.Request.Context(), c.GetString("session_id"))
	if err != nil {
		writeUnauthorized(c, "REFRESH_FAILED", "failed to refresh session")
		return
	}
	c.JSON(http.StatusOK, session)
}

func (h *SessionHandler) Delete(c *gin.Context) {
	if err := h.sessions.RevokeSession(c.Request.Context(), c.GetString("session_id")); err != nil {
		writeUnauthorized(c, "LOGOUT_FAILED", "failed to revoke session")
		return
	}
	c.Status(http.StatusNoContent)
}
