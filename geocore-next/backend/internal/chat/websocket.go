package chat

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Restrict cross-origin connections to the app's own domain.
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return false
		}
		allowed := os.Getenv("ALLOWED_ORIGIN")
		if allowed != "" && strings.Contains(origin, allowed) {
			return true
		}
		// Allow same-host and Replit preview domains
		host := r.Host
		return strings.Contains(origin, host) ||
			strings.HasSuffix(origin, ".replit.dev") ||
			strings.HasSuffix(origin, ".repl.co") ||
			origin == "http://localhost:22333" ||
			origin == "https://localhost:22333"
	},
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
			// Use a write lock because we may delete slow clients during broadcast.
			h.mu.Lock()
			for client := range h.clients[msg.ConversationID] {
				select {
				case client.send <- msg.Data:
				default:
					// Client send buffer full — remove it cleanly.
					close(client.send)
					delete(h.clients[msg.ConversationID], client)
				}
			}
			h.mu.Unlock()
		}
	}
}

// SubscribeRedis listens to all chat message events published to Redis
// (channel pattern "chat:*") and forwards them to connected WebSocket clients.
// This enables multi-instance deployments to broadcast across server nodes.
// It auto-reconnects with exponential backoff if the Redis channel closes.
func (h *Hub) SubscribeRedis(ctx context.Context) {
	if h.rdb == nil {
		log.Println("[chat-hub] no Redis client — cross-node chat broadcast disabled")
		return
	}

	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		if ctx.Err() != nil {
			return
		}

		pubsub := h.rdb.PSubscribe(ctx, "chat:*")
		ch := pubsub.Channel()
		log.Println("[chat-hub] Redis PSubscribe chat:* — listening for chat events")
		backoff = time.Second // reset on successful subscribe

		channelClosed := false
		for !channelClosed {
			select {
			case <-ctx.Done():
				pubsub.Close()
				return
			case msg, ok := <-ch:
				if !ok {
					channelClosed = true
					break
				}
				// Channel format is "chat:<conversationID>"
				convID := ""
				if len(msg.Channel) > len("chat:") {
					convID = msg.Channel[len("chat:"):]
				}
				if convID == "" {
					continue
				}
				select {
				case h.broadcast <- &WSBroadcast{
					ConversationID: convID,
					Data:           []byte(msg.Payload),
				}:
				default:
					log.Printf("[chat-hub] broadcast channel full, dropping message for %s", convID)
				}
			}
		}
		pubsub.Close()

		log.Printf("[chat-hub] Redis PubSub channel closed — reconnecting in %s", backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < maxBackoff {
			backoff *= 2
		}
	}
}

func ServeWS(hub *Hub, c *gin.Context, db *gorm.DB) {
	// Authenticate via ?token= query parameter.
	// Browsers cannot set Authorization headers on WebSocket connections,
	// so the JWT is passed as a short-lived query parameter instead.
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token query parameter"})
		return
	}
	userID, err := middleware.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	convIDStr := c.Param("id")
	convID, err := uuid.Parse(convIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid conversation ID"})
		return
	}

	// Enforce conversation membership — prevent IDOR/unauthorized subscription.
	var member ConversationMember
	if dbErr := db.Where("conversation_id = ? AND user_id = ?", convID, userID).
		First(&member).Error; dbErr != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this conversation"})
		return
	}

	conn, wsErr := upgrader.Upgrade(c.Writer, c.Request, nil)
	if wsErr != nil {
		log.Printf("WS upgrade error: %v", wsErr)
		return
	}

	client := &WSClient{
		hub:            hub,
		conn:           conn,
		conversationID: convID.String(),
		userID:         userID,
		send:           make(chan []byte, 256),
	}
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

// wsIncomingMessage is the shape expected from a client sending a chat message over WS.
type wsIncomingMessage struct {
	Content string `json:"content"`
	Type    string `json:"type"`
}

func clientReadPump(c *WSClient, hub *Hub, db *gorm.DB) {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()

	senderID, _ := uuid.Parse(c.userID)
	convID, _ := uuid.Parse(c.conversationID)

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var incoming wsIncomingMessage
		if jsonErr := json.Unmarshal(raw, &incoming); jsonErr != nil || strings.TrimSpace(incoming.Content) == "" {
			// Ignore malformed or empty frames
			continue
		}

		msgType := incoming.Type
		if msgType == "" {
			msgType = "text"
		}

		// Persist to database before broadcasting
		msg := Message{
			ID:             uuid.New(),
			ConversationID: convID,
			SenderID:       senderID,
			Content:        incoming.Content,
			Type:           msgType,
			CreatedAt:      time.Now(),
		}
		if dbErr := db.Create(&msg).Error; dbErr != nil {
			log.Printf("WS persist message error: %v", dbErr)
			continue
		}

		// Update conversation last_msg_at and unread counts
		now := time.Now()
		db.Model(&Conversation{}).Where("id = ?", convID).Update("last_msg_at", now)
		db.Model(&ConversationMember{}).
			Where("conversation_id = ? AND user_id != ?", convID, senderID).
			UpdateColumn("unread_count", gorm.Expr("unread_count + 1"))

		// Publish to Redis so other server nodes can broadcast to their clients.
		if c.hub.rdb != nil {
			if data, jsonErr := json.Marshal(msg); jsonErr == nil {
				c.hub.rdb.Publish(context.Background(),
					"chat:"+convID.String(), string(data))
			}
		} else {
			// Single-node mode: broadcast directly without Redis.
			if data, jsonErr := json.Marshal(msg); jsonErr == nil {
				hub.broadcast <- &WSBroadcast{ConversationID: c.conversationID, Data: data}
			}
		}
	}
}
