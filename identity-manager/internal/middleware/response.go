package middleware

import (
	"net/http"

	"identity-manager/internal/model"

	"github.com/gin-gonic/gin"
)

func abortError(c *gin.Context, status int, code, message string) {
	body := gin.H{
		"code":    code,
		"message": message,
	}
	c.AbortWithStatusJSON(status, body)
}

func abortUnauthorized(c *gin.Context, code, message string) {
	abortError(c, http.StatusUnauthorized, code, message)
}

func setAuthContext(c *gin.Context, session *model.Session, userInfo *model.UserInfo) {
	c.Set("session", session)
	c.Set("session_id", session.ID)
	c.Set("session_user_id", session.UserID)
	c.Set("session_user_info", *userInfo)
}
