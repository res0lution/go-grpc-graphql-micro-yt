package handler

import (
	"net/http"

	"identity-manager/internal/service"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	users service.UserService
}

func NewUserHandler(users service.UserService) *UserHandler {
	return &UserHandler{users: users}
}

func (h *UserHandler) Me(c *gin.Context) {
	user, err := h.users.GetCurrentUser(c.Request.Context(), c.GetString("session_id"))
	if err != nil {
		writeUnauthorized(c, "UNAUTHORIZED", "User not authenticated")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "User received successfully",
		"user":    user,
	})
}
