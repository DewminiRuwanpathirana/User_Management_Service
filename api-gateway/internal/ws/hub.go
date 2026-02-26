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
	clients    map[*clientConn]struct{}
	registerCh chan *clientConn
	removeCh   chan *clientConn
	broadcast  chan []byte
}

func NewHub() *Hub {
	h := &Hub{
		clients:    make(map[*clientConn]struct{}),
		registerCh: make(chan *clientConn),
		removeCh:   make(chan *clientConn),
		broadcast:  make(chan []byte, 64), // up to 64 broadcast messages can queue without blocking sender
	}
	// go starts a new goroutine to run the hub's main loop
	go h.run()
	return h
}

// hub’s background goroutine loop that continuously listens
func (h *Hub) run() {
	for { // infinite loop to keep the hub running and processing incoming events
		select {
		case client := <-h.registerCh:
			h.clients[client] = struct{}{}
		case client := <-h.removeCh:
			if _, exists := h.clients[client]; exists {
				delete(h.clients, client)
				_ = client.conn.Close()
			}
		case message := <-h.broadcast:
			for client := range h.clients { // iterate over all connected clients and send the broadcast message
				if err := client.writeText(message); err != nil {
					delete(h.clients, client)
					_ = client.conn.Close()
				}
			}
		}
	}
}

// register a new client connection to the hub and return the clientConn instance.
func (h *Hub) register(conn *websocket.Conn) *clientConn {
	client := &clientConn{conn: conn}
	h.registerCh <- client // send the clientConn instance to the register channel to be added to the clients map.
	return client
}

func (h *Hub) unregister(client *clientConn) {
	h.removeCh <- client
}

func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}
