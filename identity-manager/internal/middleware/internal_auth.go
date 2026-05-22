package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RequireInternalToken(expectedToken string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.TrimSpace(expectedToken) == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"code":    "INTERNAL_AUTH_NOT_CONFIGURED",
				"message": "internal auth token is not configured",
			})
			return
		}

		raw := c.GetHeader("Authorization")
		if !strings.HasPrefix(raw, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "bearer token is required",
			})
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
		if subtle.ConstantTimeCompare([]byte(token), []byte(expectedToken)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "invalid internal auth token",
			})
			return
		}

		c.Next()
	}
}
