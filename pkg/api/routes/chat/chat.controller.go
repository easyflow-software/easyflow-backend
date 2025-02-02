package chat

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

func RegisterChatEndpoints(r *gin.RouterGroup) {
	r.Use(middleware.LoggerMiddleware("Chat"))
	r.Use(auth.AuthGuard())
	r.Use(middleware.RateLimiterMiddleware(250, 10*time.Minute))
	r.POST("/", createChatController)
	r.GET("/preview", getChatPreviewsController)
	r.GET("/:chatId", getChatByIdController)
}

func createChatController(c *gin.Context) {
	payload, logger, db, _, _, errs := endpoint.SetupEndpoint[CreateChatRequest](c)
	if len(errs) > 0 {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errs,
		})
		return
	}

	logger.PrintfDebug("Payload: %s", payload.Name)

	user, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		})
		return
	}

	chat, err := createChat(db, payload, user.(*jwt.JWTTokenPayload), logger)
	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(http.StatusCreated, chat)
}

func getChatPreviewsController(c *gin.Context) {
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

	chats, err := getChatPreviews(db, user.(*jwt.JWTTokenPayload), logger)
	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(http.StatusOK, chats)
}

func getChatByIdController(c *gin.Context) {
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

	chatId := c.Param("chatId")

	chat, err := getChatById(db, chatId, user.(*jwt.JWTTokenPayload), logger)
	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(http.StatusOK, chat)
}
