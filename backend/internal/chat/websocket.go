package chat

import (
        "log"
        "net/http"
        "os"
        "strings"
        "sync"

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

func clientReadPump(c *WSClient, hub *Hub, db *gorm.DB) {
        defer func() {
                hub.unregister <- c
                c.conn.Close()
        }()
        // Drain and discard all client-sent frames.
        // Messages must be submitted via the REST endpoint (POST /messages) which
        // validates, persists, and then broadcasts the canonical server-shaped message.
        // Accepting raw client frames would bypass validation and allow spoofing.
        for {
                _, _, err := c.conn.ReadMessage()
                if err != nil {
                        break
                }
                // intentionally ignore client frames
        }
}
