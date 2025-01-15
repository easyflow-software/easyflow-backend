package websocket

import (
	"easyflow-backend/pkg/api"
	"easyflow-backend/pkg/common"
	"easyflow-backend/pkg/enum"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func WebsocketEndpoint(c *gin.Context) {
	w := c.Writer
	r := c.Request

	logger, err := c.Get("logger")
	if !err {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: "Logger not found in context",
		})
		c.Abort()
		return
	}

	log, ok := logger.(*common.Logger)
	if !ok {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: "Logger is not of type *common.Logger",
		})
		c.Abort()
		return
	}

	log.PrintfInfo("Upgrading connection to websocket")

	conn, sock_err := upgrader.Upgrade(w, r, nil)
	if sock_err != nil {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.UpgradeFailed,
			Details: sock_err.Error(),
		})
		c.Abort()
		log.PrintfError("Failed to upgrade connection to websocket: %s", sock_err.Error())
		return
	}

	log.PrintfDebug("Closing connection")
	defer conn.Close()
}
