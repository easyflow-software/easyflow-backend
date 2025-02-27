package socket

import (
	"easyflow-backend/pkg/jwt"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for simplicity
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func ServeWs(hub *hub, payload *jwt.JWTTokenPayload, w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			hub.logger.PrintfError("Panic in ServeWs: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := newClient(conn, payload, hub)

	go client.readMessages()
	go client.writeMessages()
	hub.logger.PrintfInfo("Client with id: %s connected", client.payload.UserID)
}
