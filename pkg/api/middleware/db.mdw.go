package middleware

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Adds the database connection to the Gin context.
// It stores the GORM DB instance in the context for access by subsequent handlers.
func DatabaseMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	}
}
