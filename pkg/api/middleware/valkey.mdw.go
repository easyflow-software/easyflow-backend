package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/valkey-io/valkey-go"
)

func ValkeyMiddleware(valkeyClient valkey.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("valkey", valkeyClient)
		c.Next()
	}
}
