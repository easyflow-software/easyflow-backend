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

	logger, _ := c.Get("logger")
	log, _ := logger.(*common.Logger)

	cm, _ := c.Get("clientManager")
	clientManager, _ := cm.(*ClientManager)

	log.PrintfInfo("Upgrading connection to websocket")

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.PrintfError("Failed to upgrade connection to websocket: %s", err.Error())
		c.AbortWithStatusJSON(http.StatusInternalServerError, api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.UpgradeFailed,
			Details: err.Error(),
		})
		return
	}

	client := NewClient(conn)
	done := make(chan struct{})

	go func() {
		client.Read()
		done <- struct{}{}
	}()
	go client.Write()

	clientManager.Register <- client
	defer func() {
		clientManager.Unregister <- client
	}()

	<-done
}
