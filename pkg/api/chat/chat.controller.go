package chat

import (
	"easyflow-backend/pkg/api"
	"easyflow-backend/pkg/api/auth"
	"easyflow-backend/pkg/common"
	"easyflow-backend/pkg/enum"
	"easyflow-backend/pkg/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterChatEndpoints(r *gin.RouterGroup) {
	r.Use(auth.AuthGuard())
	r.Use(middleware.LoggerMiddleware("Chat"))
	r.Use(middleware.RateLimiter(1, 5))
	r.POST("", CreateChatController)
	r.GET("/preview", GetChatPreviewsController)
	r.GET("/:chatId", GetChatByIdController)
}

func CreateChatController(c *gin.Context) {
	payload, logger, db, _, errors := common.SetupEndpoint[CreateChatRequest](c)
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

	chat, err := CreateChat(db, payload, user.(*auth.JWTAccessTokenPayload), logger)
	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(http.StatusCreated, chat)
}

func GetChatPreviewsController(c *gin.Context) {
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

	chats, err := GetChatPreviews(db, user.(*auth.JWTAccessTokenPayload), logger)
	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(http.StatusOK, chats)
}

func GetChatByIdController(c *gin.Context) {
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

	chatId := c.Param("chatId")

	chat, err := GetChatById(db, chatId, user.(*auth.JWTAccessTokenPayload), logger)
	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.JSON(http.StatusOK, chat)
}
