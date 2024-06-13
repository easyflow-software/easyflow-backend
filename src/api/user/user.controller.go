package user

import (
	"easyflow-backend/src/api"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterUserEndpoints(r *gin.RouterGroup) {
	r.POST("/signup", CreateUserController)
}

func CreateUserController(c *gin.Context) {
	var payload CreateUserRequest
	if err := c.ShouldBindJSON(&payload); err != nil {
		decrypted := Validator.DecryptErrors(err)
		c.JSON(http.StatusBadRequest, api.ApiError{
			Code:    http.StatusBadRequest,
			Message: "Invalid request payload",
			Details: &decrypted,
		})
		return
	}

	db, ok := c.Get("db")
	if !ok {
		c.JSON(http.StatusInternalServerError, api.ErrDatabaseConnection)
		return
	}

	user, err := CreateUser(db.(*gorm.DB), &payload)
	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(200, user)
}
