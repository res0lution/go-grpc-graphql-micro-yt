package handler

import (
	"errors"
	"io"
	"net/http"

	"identity-manager/internal/model"
	"identity-manager/internal/service"

	"github.com/gin-gonic/gin"
)

type InternalIdentityHandler struct {
	sessions service.SessionService
}

func NewInternalIdentityHandler(sessions service.SessionService) *InternalIdentityHandler {
	return &InternalIdentityHandler{sessions: sessions}
}

func (h *InternalIdentityHandler) Resolve(c *gin.Context) {
	sessionID := c.GetString("session_id")
	if sessionID == "" {
		sessionID = c.GetHeader("X-Session-ID")
	}
	if sessionID == "" {
		var req model.ResolveIdentityRequest
		if err := c.ShouldBindJSON(&req); err == nil {
			sessionID = req.SessionID
		} else if !errors.Is(err, io.EOF) {
			writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "invalid json body")
			return
		}
	}
	if sessionID == "" {
		writeError(c, http.StatusBadRequest, "SESSION_REQUIRED", "session is required")
		return
	}

	resolved, err := h.sessions.ResolveIdentity(c.Request.Context(), sessionID)
	if err != nil {
		writeError(c, http.StatusNotFound, "RESOLVE_FAILED", "failed to resolve identity")
		return
	}

	c.JSON(http.StatusOK, model.ResolveIdentityResponse{
		Success:  true,
		Identity: resolved.Identity,
		UserInfo: resolved.UserInfo,
	})
}
