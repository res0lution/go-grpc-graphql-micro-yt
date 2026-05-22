package middleware

import (
	"net/http"

	"portal-core/internal/logger"
	"portal-core/internal/model"
	"portal-core/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var log = logger.L().WithField("component", "auth_middleware")

func AuthRequired(authService *service.AuthService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sessionCookie, err := ctx.Request.Cookie("session_id")
		if err != nil {
			log.Debug("No session cookie found")

			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Not authenticated",
				"message": "Session cookie not found",
			})

			return
		}

		session, err := authService.ValidateSession(
			ctx.Request.Context(),
			sessionCookie.Value,
		)

		if err != nil {
			log.WithError(err).Debug("Invalid session")

			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Invalid session",
				"message": "Session is invalid or expired",
			})

			return
		}

		userInfo, err := authService.GetUserInfoBySessionID(
			ctx.Request.Context(),
			sessionCookie.Value,
		)

		if err != nil {
			log.WithError(err).Error("Failed to get user info")

			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Failed to get user info",
				"message": "Could not retrieve user information",
			})

			return
		}

		ctx.Set("session", session)
		ctx.Set("session_id", session.ID)
		ctx.Set("session_user_id", session.UserID)
		ctx.Set("session_user_info", userInfo)

		log.WithFields(logrus.Fields{
			"user_id": session.UserID,
			"email":   userInfo.Email,
		}).Debug("User authenticated successfully")

		ctx.Next()
	}
}

func RequireAnyGroup(
	authService *service.AuthService,
	allowedGroups ...string,
) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userInfoInterface, exists := ctx.Get("session_user_info")

		if !exists {
			log.Warn("RequireAnyGroup called without AuthRequired middleware")

			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Not authenticated",
				"message": "Authentication required",
			})

			return
		}

		userInfo, ok := userInfoInterface.(model.UserInfo)

		if !ok {
			log.Error("Invalid user_info type in context")

			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Internal error",
				"message": "Invalid user information",
			})

			return
		}

		if !userInfo.HasAnyGroup(allowedGroups) {
			log.WithFields(logrus.Fields{
				"user_id":         userInfo.Sub,
				"user_groups":     userInfo.Group,
				"required_groups": allowedGroups,
			}).Warn("Access denied: user not in required groups")

			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success":         false,
				"error":           "Access denied",
				"message":         "You do not have permission to access this resource",
				"required_groups": allowedGroups,
			})

			return
		}

		log.WithFields(logrus.Fields{
			"user_id": userInfo.Sub,
			"groups":  userInfo.Group,
		}).Debug("Group authorization successful")

		ctx.Next()
	}
}

func RequireAllGroups(
	authService *service.AuthService,
	requiredGroups ...string,
) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userInfoInterface, exists := ctx.Get("session_user_info")

		if !exists {
			log.Warn("RequireAllGroups called without AuthRequired middleware")

			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"success": false,
				"error":   "Not authenticated",
				"message": "Authentication required",
			})

			return
		}

		userInfo, ok := userInfoInterface.(model.UserInfo)

		if !ok {
			log.Error("Invalid user info type in context")

			ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Internal error",
				"message": "Invalid user information",
			})

			return
		}
		if !userInfo.HasAllGroups(requiredGroups) {
			log.WithFields(logrus.Fields{
				"user_id":         userInfo.Sub,
				"user_groups":     userInfo.Group,
				"required_groups": requiredGroups,
			}).Warn("Access denied: user missing required groups")

			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success":         false,
				"error":           "Access denied",
				"message":         "You do not have all required permissions",
				"required_groups": requiredGroups,
			})

			return
		}

		log.WithFields(logrus.Fields{
			"user_id": userInfo.Sub,
			"groups":  userInfo.Group,
		}).Debug("All groups authorization successful")

		ctx.Next()
	}
}

func OptionalAuth(authService *service.AuthService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		sessionCookie, err := ctx.Request.Cookie("session_id")

		if err != nil {
			log.Debug("No session cookie found (optional auth)")
			ctx.Next()
			return
		}

		session, err := authService.ValidateSession(
			ctx.Request.Context(),
			sessionCookie.Value,
		)

		if err != nil {
			log.WithError(err).Debug("Invalid session (optional auth)")
			ctx.Next()
			return
		}

		userInfo, err := authService.GetUserInfoBySessionID(
			ctx.Request.Context(),
			sessionCookie.Value,
		)

		if err != nil {
			log.WithError(err).Debug("Failed to get user info (optional auth)")
			ctx.Next()
			return
		}

		ctx.Set("session", session)
		ctx.Set("session_id", session.ID)
		ctx.Set("user_id", session.UserID)
		ctx.Set("user_info", userInfo)

		log.WithFields(logrus.Fields{
			"user_id": session.UserID,
			"email":   userInfo.Email,
		}).Debug("User authenticated (optional)")

		ctx.Next()
	}
}
