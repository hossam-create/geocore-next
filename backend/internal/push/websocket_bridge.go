package push

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/gorilla/websocket"
)

// ════════════════════════════════════════════════════════════════════════════
// WebSocket Bridge — delivers push via WS when user is online
// ════════════════════════════════════════════════════════════════════════════

// PushWSHub tracks online users and delivers push notifications via WebSocket
// as a first-class fallback/alternative to FCM.
type PushWSHub struct {
	mu      sync.RWMutex
	clients map[string]map[*pushWSClient]struct{} // userID → set of clients
	reg     chan *pushWSClient
	unreg   chan *pushWSClient
}

type pushWSClient struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
	hub    *PushWSHub
}

// NewPushWSHub creates a new WebSocket hub for push delivery.
func NewPushWSHub() *PushWSHub {
	return &PushWSHub{
		clients: make(map[string]map[*pushWSClient]struct{}),
		reg:     make(chan *pushWSClient, 32),
		unreg:   make(chan *pushWSClient, 32),
	}
}

// Run starts the hub event loop. Call in a goroutine.
func (h *PushWSHub) Run() {
	for {
		select {
		case c := <-h.reg:
			h.mu.Lock()
			if h.clients[c.userID] == nil {
				h.clients[c.userID] = make(map[*pushWSClient]struct{})
			}
			h.clients[c.userID][c] = struct{}{}
			h.mu.Unlock()

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
		}
	}
}

// IsOnline returns true if the user has at least one active WS connection.
func (h *PushWSHub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

// SendWS delivers a push payload to all active WS connections for a user.
func (h *PushWSHub) SendWS(userID string, msg any) error {
	data, err := json.Marshal(map[string]any{
		"type":    "push",
		"payload": msg,
	})
	if err != nil {
		return err
	}

	h.mu.RLock()
	clients := h.clients[userID]
	h.mu.RUnlock()

	for c := range clients {
		select {
		case c.send <- data:
		default:
			// Buffer full — disconnect
			h.unreg <- c
		}
	}
	return nil
}

// OnlineUserCount returns the number of users with active WS connections.
func (h *PushWSHub) OnlineUserCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// ════════════════════════════════════════════════════════════════════════════
// Global service singleton (same pattern as pkg/email)
// ════════════════════════════════════════════════════════════════════════════

var (
	defaultService *PushService
	defaultOnce    sync.Once
)

// SetDefault sets the global PushService singleton.
func SetDefault(svc *PushService) {
	defaultService = svc
}

// Default returns the global PushService. Returns a no-op service if not initialised.
func Default() *PushService {
	if defaultService != nil {
		return defaultService
	}
	// Return a no-op service so callers don't need nil checks
	defaultOnce.Do(func() {
		slog.Warn("push: Default() called before SetDefault — using no-op service")
		defaultService = &PushService{}
	})
	return defaultService
}
