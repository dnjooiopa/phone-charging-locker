package gin_server

import (
	"github.com/acoshift/pgsql/pgctx"
	"github.com/gin-gonic/gin"
)

// DatabaseMiddleware injects the database object into the request context
func DatabaseMiddleware(db pgctx.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = pgctx.NewContext(ctx, db)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
