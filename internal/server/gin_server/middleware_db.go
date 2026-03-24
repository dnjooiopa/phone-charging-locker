package gin_server

import (
	"database/sql"

	"github.com/gin-gonic/gin"

	"github.com/dnjooiopa/phone-charging-locker/pkg/dbctx"
)

// DatabaseMiddleware injects the database object into the request context
func DatabaseMiddleware(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		ctx = dbctx.NewContext(ctx, db)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
