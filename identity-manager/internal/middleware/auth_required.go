package middleware

import (
	"net/http"

	"identity-manager/internal/model"
	"identity-manager/internal/service"

	"github.com/gin-gonic/gin"
)

func AuthRequired(auth service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionCookie, err := c.Request.Cookie("session_id")
		if err != nil {
			abortUnauthorized(c, "UNAUTHORIZED", "Session cookie not found")
			return
		}

		session, err := auth.ValidateSession(c.Request.Context(), sessionCookie.Value)
		if err != nil {
			abortUnauthorized(c, "INVALID_SESSION", "Session is invalid or expired")
			return
		}

		userInfo, err := auth.GetUserInfoBySessionID(c.Request.Context(), sessionCookie.Value)
		if err != nil {
			abortUnauthorized(c, "USER_INFO_UNAVAILABLE", "Could not retrieve user information")
			return
		}

		setAuthContext(c, session, userInfo)
		c.Next()
	}
}

func OptionalAuth(auth service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionCookie, err := c.Request.Cookie("session_id")
		if err != nil {
			c.Next()
			return
		}

		session, err := auth.ValidateSession(c.Request.Context(), sessionCookie.Value)
		if err != nil {
			c.Next()
			return
		}

		userInfo, err := auth.GetUserInfoBySessionID(c.Request.Context(), sessionCookie.Value)
		if err != nil {
			c.Next()
			return
		}

		setAuthContext(c, session, userInfo)
		c.Next()
	}
}

func RequireAnyGroup(auth service.AuthService, allowedGroups ...string) gin.HandlerFunc {
	_ = auth
	return func(c *gin.Context) {
		raw, exists := c.Get("session_user_info")
		if !exists {
			abortUnauthorized(c, "UNAUTHORIZED", "Authentication required")
			return
		}
		userInfo, ok := raw.(model.UserInfo)
		if !ok {
			abortError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Invalid user information")
			return
		}
		if !userInfo.HasAnyGroup(allowedGroups) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":            "ACCESS_DENIED",
				"message":         "You do not have permission to access this resource",
				"required_groups": allowedGroups,
			})
			return
		}
		c.Next()
	}
}

func RequireAllGroups(auth service.AuthService, requiredGroups ...string) gin.HandlerFunc {
	_ = auth
	return func(c *gin.Context) {
		raw, exists := c.Get("session_user_info")
		if !exists {
			abortUnauthorized(c, "UNAUTHORIZED", "Authentication required")
			return
		}
		userInfo, ok := raw.(model.UserInfo)
		if !ok {
			abortError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Invalid user information")
			return
		}
		if !userInfo.HasAllGroups(requiredGroups) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":            "ACCESS_DENIED",
				"message":         "You do not have all required permissions",
				"required_groups": requiredGroups,
			})
			return
		}
		c.Next()
	}
}

func MustGetUserInfo(c *gin.Context) model.UserInfo {
	raw, ok := c.Get("session_user_info")
	if !ok {
		panic("user info not found in context")
	}
	info, ok := raw.(model.UserInfo)
	if !ok {
		panic("invalid user info type in context")
	}
	return info
}
