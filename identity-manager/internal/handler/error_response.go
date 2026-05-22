package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func writeError(c *gin.Context, status int, code, message string) {
	writeErrorWithFields(c, status, code, message, nil)
}

func writeErrorWithFields(c *gin.Context, status int, code, message string, fields gin.H) {
	body := gin.H{
		"code":    code,
		"message": message,
	}
	for key, value := range fields {
		body[key] = value
	}

	c.JSON(status, body)
}

func writeUnauthorized(c *gin.Context, code, message string) {
	writeError(c, http.StatusUnauthorized, code, message)
}
