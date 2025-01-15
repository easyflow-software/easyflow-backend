package middleware

import (
	"easyflow-backend/pkg/api"
	"easyflow-backend/pkg/common"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func LoggerMiddleware(module_name string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg, ok := c.Get("config")
		if !ok {
			c.JSON(http.StatusInternalServerError, api.ApiError{
				Code:    http.StatusInternalServerError,
				Error:   "ConfigError",
				Details: "Config not found in context",
			})
			c.Abort()
			return
		}

		config, ok := cfg.(*common.Config)
		if !ok {
			c.JSON(http.StatusInternalServerError, api.ApiError{
				Code:    http.StatusInternalServerError,
				Error:   "ConfigError",
				Details: "Config is not of type *common.Config",
			})
			c.Abort()
			return
		}

		c.Set("logger", common.NewLogger(os.Stdout, module_name, c, common.LogLevel(config.LogLevel)))
		c.Next()
	}
}
