package gin_server

import (
	"log"
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
				log.Printf("[ERROR] %d | %13v | %s %s | %s", status, latency, method, path, e.Err)
			}
		} else {
			log.Printf("[INFO] %d | %13v | %s %s", status, latency, method, path)
		}
	}
}
