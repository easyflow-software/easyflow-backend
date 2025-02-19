package socket

import (
	"context"
	"easyflow-backend/pkg/database"
	"easyflow-backend/pkg/jwt"
	"easyflow-backend/pkg/logger"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gorm.io/gorm"
)

type client struct {
	conn       *websocket.Conn
	send       chan message
	payload    *jwt.JWTTokenPayload
	logger     *logger.Logger
	rooms      map[string]*room
	roomsMutex sync.RWMutex
	db         *gorm.DB
}

type clientMessage struct {
	Room string `json:"room"`
	Data string `json:"data"`
	Iv   string `json:"iv"`
}

type message struct {
	clientMessage
	SenderID string `json:"sender_id"`
}

type errorMessage struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024 * 1024 // 1 MB
)

func newClient(conn *websocket.Conn, payload *jwt.JWTTokenPayload, hub *hub) *client {
	client := &client{
		conn:       conn,
		send:       make(chan message),
		payload:    payload,
		logger:     hub.logger,
		rooms:      make(map[string]*room),
		roomsMutex: sync.RWMutex{},
		db:         hub.db,
	}

	// Initialize rooms in a separate function to handle errors
	if err := client.initializeRooms(hub); err != nil {
		client.logger.PrintfError("Failed to initialize rooms for user %s: %v", payload.UserID, err)
		return nil
	}

	return client
}

func (c *client) initializeRooms(hub *hub) error {
	var chats []database.ChatsUsers
	if err := hub.db.Find(&chats).Where("user_id = ?", c.payload.UserID).Error; err != nil {
		return err
	}

	hub.roomsMutex.RLock()
	defer hub.roomsMutex.RUnlock()

	for _, chat := range chats {
		if room, exists := hub.rooms[chat.ChatID]; exists {
			room.addClient(c)
			continue
		}
		room := newRoom(chat.ChatID, hub)
		room.addClient(c)
	}

	return nil
}

func (c *client) readMessages() {
	var err error
	defer func() {
		c.logger.PrintfInfo("Cleaning up connection for user %s", c.payload.UserID)
		if r := recover(); r != nil {
			c.logger.PrintfError("Panic recovered in readMessages for user %s: %v", c.payload.UserID, r)
		}
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				c.logger.PrintfInfo("Connection closed normally for user %s", c.payload.UserID)
			} else {
				c.logger.PrintfError("Connection error for user %s: %v", c.payload.UserID, err)
			}
		}
		c.cleanup()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		var msg clientMessage
		if err = c.conn.ReadJSON(&msg); err != nil {
			return
		}

		c.logger.PrintfInfo("Received message from user %s: %v", c.payload.UserID, msg.Room)

		if err = c.handleMessage(msg); err != nil {
			return
		}
	}
}

func (c *client) handleMessage(msg clientMessage) error {
	c.roomsMutex.RLock()
	defer c.roomsMutex.RUnlock()
	room, exists := c.rooms[msg.Room]

	if !exists {
		c.logger.PrintfWarning("Access to room: %s denied for user: %s", msg.Room, c.payload.UserID)
		return c.conn.WriteJSON(errorMessage{
			Error:   "Access Denied",
			Details: "You do not have permission to access this room or the room does not exist",
		})
	}

	var message = message{
		clientMessage: msg,
		SenderID:      c.payload.UserID,
	}

	jsonString, err := json.Marshal(message)
	if err != nil {
		fmt.Println("Error marshaling message:", err)
		return err
	}
	room.hub.valkey.Do(context.Background(), room.hub.valkey.B().Publish().Channel(fmt.Sprintf("room-%s", room.id)).Message(string(jsonString)).Build())

	return nil
}

func (c *client) writeMessages() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		if r := recover(); r != nil {
			c.logger.PrintfError("Panic recovered in writeMessages for user %s: %v", c.payload.UserID, r)
		}
		c.cleanup()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				// Channel was closed
				return
			}

			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				c.logger.PrintfError("Failed to set write deadline for user %s: %v", c.payload.UserID, err)
				return
			}
			if err := c.conn.WriteJSON(msg); err != nil {
				c.logger.PrintfError("Failed to write message for user %s: %v", c.payload.UserID, err)
				return
			}

		case <-ticker.C:
			err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				c.logger.PrintfError("Failed to set write deadline for user %s: %v", c.payload.UserID, err)
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.PrintfError("Failed to write ping for user %s: %v", c.payload.UserID, err)
				return
			}
		}
	}
}

func (c *client) cleanup() {
	c.roomsMutex.Lock()
	defer c.roomsMutex.Unlock()

	// Close the connection
	if err := c.conn.Close(); err != nil {
		c.logger.PrintfError("Error closing connection for user %s: %v", c.payload.UserID, err)
	}

	// Remove client from all rooms
	for _, room := range c.rooms {
		room.removeClient(c)
	}

	// Clear the rooms map
	c.rooms = nil

	// Close the send channel
	close(c.send)

	c.conn.Close()

	c.logger.PrintfInfo("User %s disconnected and cleaned up", c.payload.UserID)
}
