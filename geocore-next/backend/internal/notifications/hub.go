package notifications

  import (
  	"encoding/json"
  	"log/slog"
  	"sync"

  	"github.com/gorilla/websocket"
  )

  // Hub manages per-user WebSocket connections for real-time notifications.
  // A user may have multiple connections (multiple browser tabs / devices).
  type Hub struct {
  	mu      sync.RWMutex
  	clients map[string]map[*WSClient]struct{} // userID → set of clients
  	reg     chan *WSClient
  	unreg   chan *WSClient
  }

  // WSClient represents a single WebSocket connection from a user.
  type WSClient struct {
  	userID string
  	conn   *websocket.Conn
  	send   chan []byte
  	hub    *Hub
  }

  // WSMessage is the envelope sent over the WebSocket.
  type WSMessage struct {
  	Type    string      `json:"type"`
  	Payload interface{} `json:"payload"`
  }

  func NewHub() *Hub {
  	return &Hub{
  		clients: make(map[string]map[*WSClient]struct{}),
  		reg:     make(chan *WSClient, 32),
  		unreg:   make(chan *WSClient, 32),
  	}
  }

  // Run starts the hub event loop. Call in a goroutine.
  func (h *Hub) Run() {
  	for {
  		select {
  		case c := <-h.reg:
  			h.mu.Lock()
  			if h.clients[c.userID] == nil {
  				h.clients[c.userID] = make(map[*WSClient]struct{})
  			}
  			h.clients[c.userID][c] = struct{}{}
  			h.mu.Unlock()
  			slog.Debug("ws: notification client connected", "user_id", c.userID)

  		case c := <-h.unreg:
  			h.mu.Lock()
  			if set, ok := h.clients[c.userID]; ok {
  				delete(set, c)
  				if len(set) == 0 {
  					delete(h.clients, c.userID)
  				}
  			}
  			h.mu.Unlock()
  			close(c.send)
  			slog.Debug("ws: notification client disconnected", "user_id", c.userID)
  		}
  	}
  }

  // BroadcastToUser delivers a notification to all active WebSocket connections
  // for a specific user. Silently skips if the user has no active connections.
  func (h *Hub) BroadcastToUser(userID string, notif *Notification) {
  	msg := WSMessage{Type: "notification", Payload: notif}
  	data, err := json.Marshal(msg)
  	if err != nil {
  		slog.Error("ws: marshal notification", "error", err.Error())
  		return
  	}

  	h.mu.RLock()
  	clients := h.clients[userID]
  	h.mu.RUnlock()

  	for c := range clients {
  		select {
  		case c.send <- data:
  		default:
  			// Client send buffer full — disconnect
  			h.unreg <- c
  		}
  	}
  }

  // writePump pumps messages from the hub to the WebSocket connection.
  func (c *WSClient) writePump() {
  	defer c.conn.Close()
  	for msg := range c.send {
  		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
  			return
  		}
  	}
  }

  // readPump keeps the read side alive (discards client messages, handles pong/close).
  func (c *WSClient) readPump() {
  	defer func() {
  		c.hub.unreg <- c
  		c.conn.Close()
  	}()
  	c.conn.SetReadLimit(512)
  	for {
  		_, _, err := c.conn.ReadMessage()
  		if err != nil {
  			return
  		}
  	}
  }
  