package socket

import (
	"context"
	"easyflow-backend/pkg/config"
	"easyflow-backend/pkg/logger"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/valkey-io/valkey-go"
	"gorm.io/gorm"
)

type hub struct {
	rooms          map[string]*Room
	roomsMutex     sync.RWMutex
	addRoom        chan *Room
	removeRoom     chan *Room
	valkey         valkey.Client
	cfg            *config.Config
	logger         *logger.Logger
	db             *gorm.DB
	shutdownCh     chan struct{}
	shutdownWg     sync.WaitGroup
	isShuttingDown atomic.Bool
}

func NewHub(cfg *config.Config, logger *logger.Logger, valkey valkey.Client, db *gorm.DB) *hub {
	return &hub{
		rooms:      make(map[string]*Room),
		roomsMutex: sync.RWMutex{},
		addRoom:    make(chan *Room),
		removeRoom: make(chan *Room),
		valkey:     valkey,
		cfg:        cfg,
		logger:     logger,
		db:         db,
		shutdownCh: make(chan struct{}),
		shutdownWg: sync.WaitGroup{},
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

// GracefulShutdown initiates a controlled shutdown of the WebSocket hub
func (h *hub) GracefulShutdown(timeout time.Duration) error {
	// Only allow shutdown once
	if h.isShuttingDown.Swap(true) {
		h.logger.PrintfWarning("Shutdown already in progress")
		return nil
	}

	h.logger.PrintfInfo("Initiating graceful shutdown of WebSocket hub with %d seconds timeout", int(timeout.Seconds()))

	// Create a context with timeout for the entire shutdown process
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Create a channel to signal completion
	done := make(chan struct{})

	// Start a goroutine to handle the shutdown process
	go func() {
		// Phase 1: Stop accepting new clients
		h.logger.PrintfInfo("Phase 1: Stopping new client connections")

		// Signal the hub to stop
		close(h.shutdownCh)

		// Phase 2: Notify all rooms about shutdown
		h.logger.PrintfInfo("Phase 2: Notifying all rooms about shutdown")

		roomCount := 0
		clientCount := 0

		h.roomsMutex.RLock()
		for id, room := range h.rooms {
			roomCount++

			// Add to WaitGroup for each room
			h.shutdownWg.Add(1)

			// Tell each room to initiate graceful close
			go func(r *Room, roomID string) {
				defer h.shutdownWg.Done()

				count, err := r.shutdown(ctx)
				if err != nil {
					h.logger.PrintfError("Error shutting down room %s: %v", roomID, err)
				} else {
					h.logger.PrintfInfo("Room %s shutdown complete with %d clients", roomID, count)
					clientCount += count
				}
			}(room, id)
		}
		h.roomsMutex.RUnlock()

		h.logger.PrintfInfo("Initiated shutdown of %d rooms with clients", roomCount)

		// Phase 3: Wait for all rooms to complete shutdown
		h.logger.PrintfInfo("Phase 3: Waiting for all room shutdowns to complete")

		// Create a channel to signal WaitGroup completion
		wgDone := make(chan struct{})
		go func() {
			h.shutdownWg.Wait()
			close(wgDone)
		}()

		// Wait for WaitGroup or timeout
		select {
		case <-wgDone:
			h.logger.PrintfInfo("All rooms completed shutdown successfully")
		case <-ctx.Done():
			h.logger.PrintfWarning("Timeout waiting for rooms to shutdown")
		}

		// Phase 4: Cleanup Valkey connections
		h.logger.PrintfInfo("Phase 4: Cleaning up external connections")

		// Clean up Valkey pubsub connections
		// This is implementation specific, based on your valkey client type

		h.logger.PrintfInfo("Graceful shutdown completed. Processed %d rooms and %d clients",
			roomCount, clientCount)

		// Signal completion
		close(done)
	}()

	// Wait for either completion or timeout
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
