package socket

import (
	"sync"
	"sync/atomic"
	"time"
)

type room struct {
	id           string
	clients      map[string]*client
	clientsMutex sync.RWMutex
	clientCount  atomic.Int32
	hub          *hub
}

func newRoom(id string, hub *hub) *room {
	room := &room{
		id:           id,
		clients:      make(map[string]*client),
		clientsMutex: sync.RWMutex{},
		clientCount:  atomic.Int32{},
		hub:          hub,
	}
	hub.addRoom <- room
	go room.watchClients()
	return room
}

func (r *room) watchClients() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		if r.clientCount.Load() < 1 {
			r.hub.removeRoom <- r
			break
		}
	}
}

func (r *room) addClient(client *client) {
	r.clientsMutex.Lock()
	defer r.clientsMutex.Unlock()
	r.clients[client.payload.ID] = client
	client.rooms[r.id] = r
	r.clientCount.Add(1)
}

func (r *room) removeClient(client *client) {
	r.clientsMutex.RLock()
	defer r.clientsMutex.RUnlock()
	delete(r.clients, client.payload.ID)
	r.clientCount.Add(-1)

	client.roomsMutex.Lock()
	defer client.roomsMutex.Unlock()
	delete(client.rooms, r.id)
}

func (r *room) broadcast(message message) {
	semaphore := make(chan struct{}, 100)
	var wg sync.WaitGroup

	r.clientsMutex.RLock()
	defer r.clientsMutex.RUnlock()
	for _, c := range r.clients {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(client *client) {
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
