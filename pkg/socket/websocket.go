package socket

import (
	"easyflow-backend/pkg/jwt"

	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type ClientMessage struct {
	Room string `json:"room"`
	Data string `json:"data"`
	Iv   string `jsom:"iv"`
}

type ErrorMessage struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024 * 1024 // 1 MB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for simplicity
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	hub        *Hub
	conn       *websocket.Conn
	send       chan []byte
	rooms      map[string]bool
	roomsMutex sync.RWMutex
	payload    *jwt.JWTTokenPayload
}

func (c *Client) readPump() {
	defer func() {
		if err := recover(); err != nil {
			c.hub.logger.PrintfError("Panic in readPump: %v", err)
			err := c.conn.WriteJSON(ErrorMessage{Error: "Internal Server Error", Details: "An unexpected error occurred."})
			if err != nil {
				c.hub.logger.PrintfError("Failed to send error Message to client")
			}
		}
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		c.hub.logger.PrintfError("Could not set read deadline. Error: %s", err)
		panic(err)
	}
	c.conn.SetPongHandler(func(string) error { err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); return err })

	for {
		_, message, readErr := c.conn.ReadMessage()
		if readErr != nil {
			if websocket.IsCloseError(readErr, websocket.CloseAbnormalClosure, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.hub.logger.PrintfInfo("Client with id: %s disconnected", c.payload.UserID)
				break
			}
			c.hub.logger.PrintfWarning("Error while reading message from user. Error: %s", readErr)

			if err := c.conn.WriteJSON(ErrorMessage{Error: "Internal Server Error", Details: "Could not read message"}); err != nil {
				c.hub.logger.PrintfWarning("Cannot send error message to client. Disconnecting")
				break
			}
			continue
		}

		var msg ClientMessage
		unmarshalErr := json.Unmarshal(message, &msg)
		if unmarshalErr != nil {
			c.hub.logger.PrintfWarning("Could not unmarshal json. Error: %s", unmarshalErr)
			if err := c.conn.WriteJSON(ErrorMessage{Error: "Bad Request", Details: "Could not unmarshal json provided"}); err != nil {
				c.hub.logger.PrintfWarning("Cannot send error message to client. Disconnecting")
				break
			}
			continue
		}

		c.roomsMutex.RLock()
		_, exists := c.rooms[msg.Room]
		c.roomsMutex.RUnlock()

		if !exists {
			c.hub.logger.PrintfWarning("User with id: %s cannot write into room: %s", c.payload.UserID, msg.Room)
			if err := c.conn.WriteJSON(ErrorMessage{Error: "Unauthorized", Details: fmt.Sprintf("You cannot write into room: %s", msg.Room)}); err != nil {
				c.hub.logger.PrintfWarning("Cannot send error message to client. Disconnecting")
				break
			}
			continue
		}

		m := Message{
			ClientMessage: msg,
			SenderID:      c.payload.UserID,
			Client:        c,
		}
		c.hub.broadcast <- m
		c.hub.logger.PrintfDebug("User %s sent message to broadcast", c.payload.UserID)
	}
}

func (c *Client) subscribe(room string) {
	c.roomsMutex.Lock()
	defer c.roomsMutex.Unlock()

	if !c.rooms[room] {
		c.rooms[room] = true
		c.hub.addClientToRoom(c, room)
	}
}

func (c *Client) unsubscribe(room string) {
	c.roomsMutex.Lock()
	defer c.roomsMutex.Unlock()

	if c.rooms[room] {
		delete(c.rooms, room)
		c.hub.removeClientFromRoom(c, room)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		if err := recover(); err != nil {
			c.hub.logger.PrintfError("Panic in writePump: %v", err)
			err := c.conn.WriteJSON(ErrorMessage{Error: "Internal Server Error", Details: "An unexpected error occurred."})
			if err != nil {
				c.hub.logger.PrintfError("Failed to send error Message to client")
			}
		}
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.Close()
				return
			}

			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				c.hub.logger.PrintfError("Could not set write deadline. Error: %s", err)
				panic(err)
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("Write error: %v", err)
				return
			}
		case <-ticker.C:
			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				c.hub.logger.PrintfError("Could not set write deadline. Error: %s", err)
				panic(err)
			}

			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func ServeWs(hub *Hub, payload *jwt.JWTTokenPayload, w http.ResponseWriter, r *http.Request) {
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
	client := &Client{
		hub:     hub,
		conn:    conn,
		send:    make(chan []byte, 256),
		rooms:   make(map[string]bool),
		payload: payload,
	}
	client.hub.register <- client

	hub.logger.PrintfInfo("Client with id: %s connected", client.payload.UserID)
	go client.writePump()
	go client.readPump()
}
