package ws

import (
	"sync"

	"github.com/gorilla/websocket"
)

type clientConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *clientConn) writeJSON(value any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.conn.WriteJSON(value)
}

func (c *clientConn) writeText(message []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.conn.WriteMessage(websocket.TextMessage, message)
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*clientConn]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*clientConn]struct{}),
	}
}

// Adding clients to the hub and removing them when they disconnect.
func (h *Hub) register(conn *websocket.Conn) *clientConn {
	client := &clientConn{conn: conn}

	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()

	return client
}

func (h *Hub) unregister(client *clientConn) {
	h.mu.Lock()
	delete(h.clients, client)
	h.mu.Unlock()

	_ = client.conn.Close()
}

func (h *Hub) Broadcast(message []byte) {
	h.BroadcastExcept(message, nil)
}

func (h *Hub) BroadcastExcept(message []byte, excluded *clientConn) {
	h.mu.RLock()
	clients := make([]*clientConn, 0, len(h.clients))
	for client := range h.clients {
		if client == excluded {
			continue
		}
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		if err := client.writeText(message); err != nil {
			h.unregister(client)
		}
	}
}
