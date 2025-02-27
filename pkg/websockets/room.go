package socket

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type Room struct {
	id              string
	clients         map[string]*Client
	clientsMutex    sync.RWMutex
	clientCount     atomic.Int32
	hub             *hub
	shutdownStarted atomic.Bool
}

func newRoom(id string, hub *hub) *Room {
	room := &Room{
		id:           id,
		clients:      make(map[string]*Client),
		clientsMutex: sync.RWMutex{},
		clientCount:  atomic.Int32{},
		hub:          hub,
	}
	hub.addRoom <- room
	go room.watchClients()
	return room
}

func (r *Room) watchClients() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		if r.clientCount.Load() < 1 {
			r.hub.removeRoom <- r
			break
		}
	}
}

func (r *Room) addClient(client *Client) {
	r.clientsMutex.Lock()
	defer r.clientsMutex.Unlock()
	r.clients[client.payload.ID] = client
	client.rooms[r.id] = r
	r.clientCount.Add(1)
}

func (r *Room) removeClient(client *Client) {
	r.clientsMutex.RLock()
	defer r.clientsMutex.RUnlock()
	delete(r.clients, client.payload.ID)
	r.clientCount.Add(-1)

	client.roomsMutex.Lock()
	defer client.roomsMutex.Unlock()
	delete(client.rooms, r.id)
}

func (r *Room) broadcast(message message) {
	semaphore := make(chan struct{}, 100)
	var wg sync.WaitGroup

	r.clientsMutex.RLock()
	defer r.clientsMutex.RUnlock()
	for _, c := range r.clients {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(client *Client) {
			defer func() {
				wg.Done()
				<-semaphore
			}()
			select {
			case client.send <- message:
			default:
				r.removeClient(client)
			}
		}(c)
	}
	wg.Wait()
}

// shutdown gracefully closes all clients in this room
func (r *Room) shutdown(ctx context.Context) (int, error) {
	// Only allow shutdown once
	if r.shutdownStarted.Swap(true) {
		return 0, nil
	}

	// Create a WaitGroup to track client shutdowns
	var wg sync.WaitGroup

	// Copy client references to avoid long lock
	r.clientsMutex.RLock()
	clients := make([]*Client, 0, len(r.clients))
	for _, client := range r.clients {
		clients = append(clients, client)
	}
	r.clientsMutex.RUnlock()

	// Count of clients processed
	clientCount := len(clients)

	// Create a context with deadline for all clients
	// Using a slightly shorter timeout than the parent context
	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(10 * time.Second)
	} else {
		// Subtract 500ms to ensure we complete before parent times out
		deadline = deadline.Add(-500 * time.Millisecond)
	}
	clientCtx, clientCancel := context.WithDeadline(ctx, deadline)
	defer clientCancel()

	// Semaphore to limit concurrent shutdowns
	sem := make(chan struct{}, 50) // Max 50 concurrent client shutdowns

	// Start client shutdowns
	for _, client := range clients {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(c *Client) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			// Send close frame with status code
			c.initiateGracefulClose(clientCtx, websocket.CloseGoingAway, "Server is shutting down")
		}(client)
	}

	// Wait for client shutdowns or timeout
	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		// All clients completed shutdown
		return clientCount, nil
	case <-ctx.Done():
		// Context deadline exceeded
		return clientCount, ctx.Err()
	}
}
