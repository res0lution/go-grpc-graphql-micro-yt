package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func AuthContextStub() gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("session_id")
		if err == nil && sessionID != "" {
			c.Set("session_id", sessionID)
		}
		c.Next()
	}
}

func RequireSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString("session_id") == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "session is required",
			})
			return
		}
		c.Next()
	}
}
