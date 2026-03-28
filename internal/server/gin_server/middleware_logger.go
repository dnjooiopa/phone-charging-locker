package gin_server

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path

		if len(c.Errors) > 0 {
			for _, e := range c.Errors {
				slog.Error("request error", "status", status, "latency", latency, "method", method, "path", path, "error", e.Err)
			}
		} else {
			slog.Info("request", "status", status, "latency", latency, "method", method, "path", path)
		}
	}
}
