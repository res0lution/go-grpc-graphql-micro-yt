package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type healthPinger interface {
	Ping(ctx context.Context) error
}

type HealthHandler struct {
	db healthPinger
}

func NewHealthHandler(db healthPinger) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		writeErrorWithFields(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "database is unavailable", gin.H{
			"status": "error",
			"db":     "unavailable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"db":     "up",
	})
}

func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
