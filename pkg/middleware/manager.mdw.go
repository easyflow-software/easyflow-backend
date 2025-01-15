package middleware

import (
	"easyflow-backend/pkg/websocket"

	"github.com/gin-gonic/gin"
)

func ClientManagerMiddleware(cm *websocket.ClientManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("clientManager", cm)
		c.Next()
	}
}
