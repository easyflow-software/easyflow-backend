package socket

import (
	"context"
	"easyflow-backend/pkg/config"
	"easyflow-backend/pkg/logger"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/valkey-io/valkey-go"
	"gorm.io/gorm"
)

type hub struct {
	rooms      map[string]*room
	roomsMutex sync.RWMutex
	addRoom    chan *room
	removeRoom chan *room
	valkey     valkey.Client
	cfg        *config.Config
	logger     *logger.Logger
	db         *gorm.DB
}

func NewHub(cfg *config.Config, logger *logger.Logger, valkey valkey.Client, db *gorm.DB) *hub {
	return &hub{
		rooms:      make(map[string]*room),
		roomsMutex: sync.RWMutex{},
		addRoom:    make(chan *room),
		removeRoom: make(chan *room),
		valkey:     valkey,
		cfg:        cfg,
		logger:     logger,
		db:         db,
	}
}

func (h *hub) Run() {
	c, cancel := h.valkey.Dedicate()
	defer cancel()

	wait := c.SetPubSubHooks(valkey.PubSubHooks{
		OnMessage: func(msg valkey.PubSubMessage) {
			var message message
			err := json.Unmarshal([]byte(msg.Message), &message)
			if err != nil {
				h.logger.Printf("Failed to unmarshal message from valkey: %v", err)
				return
			}
			h.roomsMutex.RLock()
			defer h.roomsMutex.RUnlock()
			if room, ok := h.rooms[message.Room]; ok {
				room.broadcast(message)
			} else {
				h.logger.Printf("Received message for unknown room %s", message.Room)
			}
		},
	})

	h.logger.PrintfInfo("Started listening for multi instance communication")

	for {
		select {
		case room := <-h.addRoom:
			h.roomsMutex.Lock()
			h.rooms[room.id] = room
			h.roomsMutex.Unlock()
			c.Do(context.Background(), c.B().Subscribe().Channel(fmt.Sprintf("room-%s", room.id)).Build())
			h.logger.PrintfInfo("Subscribed to room %s", room.id)
		case room := <-h.removeRoom:
			h.roomsMutex.Lock()
			delete(h.rooms, room.id)
			h.roomsMutex.Unlock()
			c.Do(context.Background(), c.B().Unsubscribe().Channel(fmt.Sprintf("room-%s", room.id)).Build())
			h.logger.PrintfInfo("Unsubscribed from room %s", room.id)
		case err := <-wait:
			h.logger.PrintfError("Failed to handle multi instance pub sub stream")
			panic(err)
		}
	}
}
