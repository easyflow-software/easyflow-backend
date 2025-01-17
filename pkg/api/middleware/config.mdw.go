package middleware

import (
	"easyflow-backend/pkg/config"
	"github.com/gin-gonic/gin"
)

// Adds the application configuration to the Gin context.
// It stores the config in the context for access by subsequent handlers.
func ConfigMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("config", cfg)
		c.Next()
	}
}
