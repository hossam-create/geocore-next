package chat

import (
	"log"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type WSClient struct {
	hub            *Hub
	conn           *websocket.Conn
	conversationID string
	userID         string
	send           chan []byte
}

type Hub struct {
	clients    map[string]map[*WSClient]bool
	broadcast  chan *WSBroadcast
	register   chan *WSClient
	unregister chan *WSClient
	rdb        *redis.Client
	mu         sync.RWMutex
}

type WSBroadcast struct {
	ConversationID string
	Data           []byte
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		clients:    make(map[string]map[*WSClient]bool),
		broadcast:  make(chan *WSBroadcast, 512),
		register:   make(chan *WSClient, 256),
		unregister: make(chan *WSClient, 256),
		rdb:        rdb,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if _, ok := h.clients[client.conversationID]; !ok {
				h.clients[client.conversationID] = make(map[*WSClient]bool)
			}
			h.clients[client.conversationID][client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.conversationID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.send)
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients[msg.ConversationID] {
				select {
				case client.send <- msg.Data:
				default:
					close(client.send)
					delete(h.clients[msg.ConversationID], client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func ServeWS(hub *Hub, c *gin.Context, db *gorm.DB) {
	convID := c.Param("conversationId")
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WS upgrade error: %v", err)
		return
	}
	client := &WSClient{hub: hub, conn: conn, conversationID: convID, send: make(chan []byte, 256)}
	hub.register <- client
	go clientWritePump(client)
	go clientReadPump(client, hub, db)
}

func clientWritePump(c *WSClient) {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}

func clientReadPump(c *WSClient, hub *Hub, db *gorm.DB) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		hub.broadcast <- &WSBroadcast{ConversationID: c.conversationID, Data: data}
	}
}
