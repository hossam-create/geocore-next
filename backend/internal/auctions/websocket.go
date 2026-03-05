package auctions

import (
	"log"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Client struct {
	conn      *websocket.Conn
	auctionID string
	send      chan []byte
}

type Hub struct {
	clients   map[string]map[*Client]bool
	broadcast chan *BroadcastMsg
	register  chan *Client
	unregister chan *Client
	rdb       *redis.Client
	mu        sync.RWMutex
}

type BroadcastMsg struct {
	AuctionID string
	Data      []byte
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		clients:    make(map[string]map[*Client]bool),
		broadcast:  make(chan *BroadcastMsg, 256),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		rdb:        rdb,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if _, ok := h.clients[client.auctionID]; !ok {
				h.clients[client.auctionID] = make(map[*Client]bool)
			}
			h.clients[client.auctionID][client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.auctionID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients[msg.AuctionID] {
				select {
				case client.send <- msg.Data:
				default:
					close(client.send)
					delete(h.clients[msg.AuctionID], client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func ServeWS(hub *Hub, c *gin.Context, db *gorm.DB) {
	auctionID := c.Param("id")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	client := &Client{conn: conn, auctionID: auctionID, send: make(chan []byte, 256)}
	hub.register <- client
	go writePump(client, hub)
	go readPump(client, hub)
}

func writePump(c *Client, hub *Hub) {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}

func readPump(c *Client, hub *Hub) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}
