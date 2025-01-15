package user

import (
	"easyflow-backend/pkg/api"
	"easyflow-backend/pkg/api/auth"
	"easyflow-backend/pkg/common"
	"easyflow-backend/pkg/enum"
	"easyflow-backend/pkg/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterUserEndpoints(r *gin.RouterGroup) {
	r.Use(middleware.LoggerMiddleware("User"))
	r.Use(middleware.RateLimiter(1, 4))
	r.POST("/signup", middleware.RateLimiter(1, 0), CreateUserController)
	r.GET("/", auth.AuthGuard(), GetUserController)
	r.GET("/exists/:email", UserExists)
	r.GET("/profile-picture", auth.AuthGuard(), GetProfilePictureController)
	r.GET("/upload-profile-picture", auth.AuthGuard(), GenerateUploadProfilePictureURLController)
	r.PUT("/", auth.AuthGuard(), UpdateUserController)
	r.DELETE("/", auth.AuthGuard(), DeleteUserController)
}

func CreateUserController(c *gin.Context) {
	payload, logger, db, cfg, errors := common.SetupEndpoint[CreateUserRequest](c)
	if errors != nil {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errors,
		})
		return
	}

	user, err := CreateUser(db, payload, cfg, logger)
	if err != nil {
		c.JSON(err.Code, err)
		return
	}
	c.JSON(200, user)
}

func GetUserController(c *gin.Context) {
	_, logger, db, _, errors := common.SetupEndpoint[any](c)
	if errors != nil {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errors,
		})
		return
	}

	user, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		})
		return
	}

	userFromDb, err := GetUserById(db, user.(*auth.JWTAccessTokenPayload), logger)

	if err != nil {
		c.JSON(err.Code, err)
		return
	}
	c.JSON(200, userFromDb)
}

func GetProfilePictureController(c *gin.Context) {
	_, logger, db, cfg, errors := common.SetupEndpoint[any](c)
	if errors != nil {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errors,
		})
		return
	}

	user, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		})
		return
	}

	imageURL, err := GenerateGetProfilePictureURL(db, user.(*auth.JWTAccessTokenPayload), logger, cfg)

	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(200, imageURL)
}

func UserExists(c *gin.Context) {
	_, logger, db, _, errors := common.SetupEndpoint[any](c)
	if errors != nil {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errors,
		})
		return
	}

	email := c.Param("email")
	if email == ":email" {
		c.JSON(http.StatusBadRequest, api.ApiError{
			Code:  http.StatusBadRequest,
			Error: enum.MalformedRequest,
		})
		return
	}

	userInDb, err := GetUserByEmail(db, email, logger)

	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(200, userInDb)
}

func UpdateUserController(c *gin.Context) {
	payload, logger, db, _, errors := common.SetupEndpoint[UpdateUserRequest](c)
	if errors != nil {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errors,
		})
		return
	}

	user, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		})
	}

	updatedUser, err := UpdateUser(db, user.(*auth.JWTAccessTokenPayload), payload, logger)

	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(200, updatedUser)
}

func GenerateUploadProfilePictureURLController(c *gin.Context) {
	_, logger, db, cfg, errors := common.SetupEndpoint[any](c)
	if errors != nil {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errors,
		})
		return
	}

	user, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		})
	}

	uploadURL, err := GenerateUploadProfilePictureURL(db, user.(*auth.JWTAccessTokenPayload), logger, cfg)

	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(200, uploadURL)
}

func DeleteUserController(c *gin.Context) {
	_, logger, db, _, errors := common.SetupEndpoint[CreateUserRequest](c)
	if errors != nil {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errors,
		})
		return
	}

	user, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		})
		return
	}

	err := DeleteUser(db, user.(*auth.JWTAccessTokenPayload), logger)

	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(200, gin.H{})
}
