package context_helper

import (
	"portal-core/internal/model"

	"github.com/gin-gonic/gin"
)

func GetSession(ctx *gin.Context) (*model.Session, bool) {
	sessionInterface, exists := ctx.Get("session")

	if !exists {
		return nil, false
	}

	session, ok := sessionInterface.(*model.Session)

	if !ok {
		return nil, false
	}

	return session, true
}

func GetUserID(ctx *gin.Context) (string, bool) {
	userID, exists := ctx.Get("session_user_id")

	if !exists {
		return "", false
	}

	userIDStr, ok := userID.(string)

	if !ok {
		return "", false
	}

	return userIDStr, true
}

func GetUserInfo(ctx *gin.Context) (model.UserInfo, bool) {
	userInfoInterface, exists := ctx.Get("session_user_info")

	if !exists {
		return model.UserInfo{}, false
	}

	userInfo, ok := userInfoInterface.(model.UserInfo)

	if !ok {
		return model.UserInfo{}, false
	}

	return userInfo, true
}

func MustGetUserInfo(ctx *gin.Context) model.UserInfo {
	userInfo, exists := GetUserInfo(ctx)

	if !exists {
		panic("user info not found in context, did you forget AuthRequired middleware?")
	}

	return userInfo
}
