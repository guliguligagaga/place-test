package logging

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"slices"
	"time"
)

var ingnoredLogPath = []string{"/healthz", "/probes", "/metrics"}

func Ginrus() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// some evil middlewares modify this values
		path := c.Request.URL.Path
		lg := logger.WithField("request_id", c.Request.Header.Get("x-request-id"))
		c.Set("logger", lg)
		c.Set("request_id", c.Request.Header.Get("x-request-id"))
		c.Next()
		if slices.Contains(ingnoredLogPath, path) {
			return
		}
		fields := logrus.Fields{
			"status":  c.Writer.Status(),
			"method":  c.Request.Method,
			"path":    c.Request.URL.String(),
			"ip":      c.ClientIP(),
			"latency": time.Since(start),
		}
		entry := lg.WithFields(fields)

		if len(c.Errors) > 0 {
			// Append error field if this is an erroneous request.
			entry.Error(c.Errors.String())
		} else {
			entry.Debug()
		}
	}
}
