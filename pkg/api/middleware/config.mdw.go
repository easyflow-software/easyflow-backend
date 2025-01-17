package middleware

import (
	"easyflow-backend/pkg/config"

	"github.com/gin-gonic/gin"
)

func ConfigMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("config", cfg)
		c.Next()
	}
}
