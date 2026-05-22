package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func AccessLog(log *logrus.Entry) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		requestID := c.GetString("request_id")
		if requestID == "" {
			requestID = c.GetHeader(requestIDHeader)
		}
		if requestID == "" {
			requestID = "-"
		}

		log.WithFields(logrus.Fields{
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"status_code": c.Writer.Status(),
			"latency_ms":  time.Since(start).Milliseconds(),
			"request_id":  requestID,
		}).Info("http request")
	}
}
