package auctions

import (
        "context"
        "log"
        "net/http"
        "sync"
        "time"

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

type Client struct {
        conn      *websocket.Conn
        auctionID string
        send      chan []byte
}

type Hub struct {
        clients    map[string]map[*Client]bool
        broadcast  chan *BroadcastMsg
        register   chan *Client
        unregister chan *Client
        rdb        *redis.Client
        mu         sync.RWMutex
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

// Run processes register/unregister/broadcast events.
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
                        h.mu.Lock()
                        for client := range h.clients[msg.AuctionID] {
                                select {
                                case client.send <- msg.Data:
                                default:
                                        close(client.send)
                                        delete(h.clients[msg.AuctionID], client)
                                }
                        }
                        h.mu.Unlock()
                }
        }
}

// SubscribeRedis listens to all auction bid events published to Redis
// (channel pattern "auction:*") and forwards them to the in-process hub
// so every connected WebSocket client receives live bid updates.
// It auto-reconnects with exponential backoff if the Redis channel closes.
func (h *Hub) SubscribeRedis(ctx context.Context) {
        if h.rdb == nil {
                log.Println("[auction-hub] no Redis client — cross-client bid broadcast disabled")
                return
        }

        backoff := time.Second
        const maxBackoff = 30 * time.Second

        for {
                if ctx.Err() != nil {
                        return
                }

                pubsub := h.rdb.PSubscribe(ctx, "auction:*")
                ch := pubsub.Channel()
                log.Println("[auction-hub] Redis PSubscribe auction:* — listening for bid events")
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
                                // Channel format is "auction:<auctionID>"
                                auctionID := ""
                                if len(msg.Channel) > len("auction:") {
                                        auctionID = msg.Channel[len("auction:"):]
                                }
                                if auctionID == "" {
                                        continue
                                }
                                h.broadcast <- &BroadcastMsg{
                                        AuctionID: auctionID,
                                        Data:      []byte(msg.Payload),
                                }
                        }
                }
                pubsub.Close()

                log.Printf("[auction-hub] Redis PubSub channel closed — reconnecting in %s", backoff)
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
