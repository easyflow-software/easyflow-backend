package socket

import (
	"easyflow-backend/pkg/database"
	"easyflow-backend/pkg/logger"

	"encoding/json"
	"fmt"

	"gorm.io/gorm"
)

type Message struct {
	ClientMessage
	SenderID string  `json:"senderId"`
	Client   *Client `json:"-"`
}

type Hub struct {
	db         *gorm.DB
	logger     *logger.Logger
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	rooms      map[string]map[*Client]bool
}

func NewHub(db *gorm.DB, logger *logger.Logger) *Hub {
	return &Hub{
		db:         db,
		logger:     logger,
		broadcast:  make(chan Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
	}
}

func (h *Hub) Run() {
	defer func() {
		if err := recover(); err != nil {
			h.logger.PrintfError("Panic in readPump: %v", err)
		}
	}()
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			// Get user from database including his chats
			var user = database.User{ID: client.payload.UserID}
			if err := h.db.Preload("Chats").First(&user).Error; err != nil {
				h.logger.PrintfWarning("Could not get user with id: %s from database. Error: %s.", client.payload.UserID, err.Error())
				client.send <- []byte(fmt.Sprintf(`{"error": "User not found", "details": %s}`, err.Error()))
				delete(h.clients, client)
				close(client.send)
			}
			// Add user the respective chat rooms
			for _, chat := range user.Chats {
				client.subscribe(chat.ID)
			}
		case client := <-h.unregister:
			// Check if the client is even connected
			if h.clients[client] {
				// Close client connection and remove from rooms
				delete(h.clients, client)
				close(client.send)
				for room := range client.rooms {
					client.unsubscribe(room)
				}
			}
		case message := <-h.broadcast:
			// Check if the room exists
			if clients, ok := h.rooms[message.Room]; ok {
				var dbMsg = database.Message{
					Content:  message.Data,
					Iv:       message.Iv,
					ChatID:   message.Room,
					SenderID: message.SenderID,
				}
				if err := h.db.Create(&dbMsg).Error; err != nil {
					h.logger.PrintfError("Could not create message entry for chat: %s", message.Room)
					err := message.Client.conn.WriteJSON(ErrorMessage{Error: "Internal Server Error", Details: err.Error()})
					if err != nil {
						h.logger.PrintfError("Failed to send error message to client")
						h.unregister <- message.Client
						message.Client.conn.Close()
					}
					continue
				}
				msg, err := json.Marshal(dbMsg)
				if err != nil {
					h.logger.PrintfError("Failed to marshall message data to json string")
					err := message.Client.conn.WriteJSON(ErrorMessage{Error: "Internal Server Error", Details: err.Error()})
					if err != nil {
						h.logger.PrintfError("Failed to send error message to client")
						h.unregister <- message.Client
						message.Client.conn.Close()
					}
					continue
				}
				for client := range clients {
					select {
					case client.send <- msg:
					default:
						close(client.send)
						delete(clients, client)
					}
				}
			}
		}
	}
}

func (h *Hub) addClientToRoom(c *Client, room string) {
	if h.rooms[room] == nil {
		h.rooms[room] = make(map[*Client]bool)
	}
	h.rooms[room][c] = true
}

func (h *Hub) removeClientFromRoom(c *Client, room string) {
	if clients, ok := h.rooms[room]; ok {
		if clients[c] {
			delete(clients, c)
			if len(clients) == 0 {
				delete(h.rooms, room)
			}
		}
	}
}
