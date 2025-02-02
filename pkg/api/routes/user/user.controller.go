package user

import (
	"easyflow-backend/pkg/api/endpoint"
	"easyflow-backend/pkg/api/errors"
	"easyflow-backend/pkg/api/middleware"
	"easyflow-backend/pkg/api/routes/auth"
	"easyflow-backend/pkg/enum"
	"easyflow-backend/pkg/jwt"
	"time"

	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterUserEndpoints(r *gin.RouterGroup) {
	r.Use(middleware.LoggerMiddleware("User"))
	r.Use(middleware.RateLimiterMiddleware(100, 10*time.Minute))
	r.POST("/signup", middleware.RateLimiterMiddleware(10, 10*time.Minute), createUserController)
	r.GET("/", auth.AuthGuard(), getUserController)
	r.GET("/exists/:email", userExists)
	r.GET("/profile-picture", auth.AuthGuard(), getProfilePictureController)
	r.GET("/upload-profile-picture", auth.AuthGuard(), uploadProfilePictureController)
	r.PUT("/", auth.AuthGuard(), updateUserController)
	r.DELETE("/", auth.AuthGuard(), deleteUserController)
}

func createUserController(c *gin.Context) {
	payload, logger, db, cfg, _, errs := endpoint.SetupEndpoint[CreateUserRequest](c)
	if len(errs) > 0 {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errs,
		})
		return
	}

	user, err := createUser(db, payload, cfg, logger, c.ClientIP())
	if err != nil {
		c.JSON(err.Code, err)
		return
	}
	c.JSON(200, user)
}

func getUserController(c *gin.Context) {
	_, logger, db, _, _, errs := endpoint.SetupEndpoint[any](c)
	if len(errs) > 0 {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errs,
		})
		return
	}

	user, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		})
		return
	}

	userFromDb, err := getUserById(db, user.(*jwt.JWTTokenPayload), logger)

	if err != nil {
		c.JSON(err.Code, err)
		return
	}
	c.JSON(200, userFromDb)
}

func getProfilePictureController(c *gin.Context) {
	_, logger, db, _, _, errs := endpoint.SetupEndpoint[any](c)
	if len(errs) > 0 {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errs,
		})
		return
	}

	user, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		})
		return
	}

	imageURL, err := getProfilePictureURL(db, user.(*jwt.JWTTokenPayload), logger)

	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(200, imageURL)
}

func userExists(c *gin.Context) {
	_, logger, db, _, _, errs := endpoint.SetupEndpoint[any](c)
	if len(errs) > 0 {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errs,
		})
		return
	}

	email := c.Param("email")
	if email == ":email" {
		c.JSON(http.StatusBadRequest, errors.ApiError{
			Code:  http.StatusBadRequest,
			Error: enum.MalformedRequest,
		})
		return
	}

	userInDb, err := getUserByEmail(db, email, logger)

	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(200, userInDb)
}

func updateUserController(c *gin.Context) {
	payload, logger, db, _, _, errs := endpoint.SetupEndpoint[UpdateUserRequest](c)
	if len(errs) > 0 {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errs,
		})
		return
	}

	user, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		})
	}

	updatedUser, err := updateUser(db, user.(*jwt.JWTTokenPayload), payload, logger)

	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(200, updatedUser)
}

func uploadProfilePictureController(c *gin.Context) {
	_, logger, db, cfg, _, errs := endpoint.SetupEndpoint[any](c)
	if len(errs) > 0 {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errs,
		})
		return
	}

	user, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		})
	}

	uploadURL, err := generateUploadProfilePictureURL(db, user.(*jwt.JWTTokenPayload), logger, cfg)

	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(200, uploadURL)
}

func deleteUserController(c *gin.Context) {
	_, logger, db, _, _, errs := endpoint.SetupEndpoint[CreateUserRequest](c)
	if len(errs) > 0 {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errs,
		})
		return
	}

	user, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		})
		return
	}

	err := deleteUser(db, user.(*jwt.JWTTokenPayload), logger)

	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(200, gin.H{})
}
