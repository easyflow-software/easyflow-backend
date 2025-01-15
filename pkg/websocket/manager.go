package websocket

import "sync"

type ClientManager struct {
	Clients    map[*Client]bool
	Register   chan *Client
	Unregister chan *Client
	mu         sync.Mutex
}

func NewClientManager() *ClientManager {
	return &ClientManager{
		Clients:    make(map[*Client]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (manager *ClientManager) Start() {
	for {
		select {
		case client := <-manager.Register:
			manager.mu.Lock()
			manager.Clients[client] = true
			manager.mu.Unlock()
		case client := <-manager.Unregister:
			manager.mu.Lock()
			if _, ok := manager.Clients[client]; ok {
				delete(manager.Clients, client)
				close(client.Outbound) 
			}
			manager.mu.Unlock()
		}
	}
}

func (manager *ClientManager) Broadcast(message []byte) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	for client := range manager.Clients {
		client.Outbound <- message
	}
}
