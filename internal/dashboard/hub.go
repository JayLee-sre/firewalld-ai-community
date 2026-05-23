package dashboard

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"zhiyuwaf/internal/model"
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan model.AttackLog
}

type Hub struct {
	mu         sync.RWMutex
	clients    map[*Client]bool
	broadcast  chan model.AttackLog
	register   chan *Client
	unregister chan *Client
	done       chan struct{}
	upgrader   websocket.Upgrader
}

func NewHub(allowedOrigins []string) *Hub {
	checkOrigin := func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		for _, o := range allowedOrigins {
			if o == "*" || o == origin {
				return true
			}
		}
		return false
	}

	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan model.AttackLog, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		done:       make(chan struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: checkOrigin,
		},
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("ws client connected, total: %d", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("ws client disconnected, total: %d", len(h.clients))

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()

		case <-h.done:
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
				client.conn.Close()
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return
		}
	}
}

func (h *Hub) Stop() {
	close(h.done)
}

func (h *Hub) Broadcast(log model.AttackLog) {
	select {
	case h.broadcast <- log:
	default:
	}
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan model.AttackLog, 64),
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteJSON(msg); err != nil {
			break
		}
	}
}
