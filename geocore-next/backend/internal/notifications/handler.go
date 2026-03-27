package notifications

  import (
  	"fmt"
  	"net/http"
  	"time"

  	"github.com/geocore-next/backend/pkg/middleware"
  	"github.com/geocore-next/backend/pkg/response"
  	"github.com/gin-gonic/gin"
  	"github.com/google/uuid"
  	"github.com/gorilla/websocket"
  	"gorm.io/gorm"
  )

  var upgrader = websocket.Upgrader{
  	ReadBufferSize:  1024,
  	WriteBufferSize: 4096,
  	CheckOrigin:     func(r *http.Request) bool { return true },
  }

  type Handler struct {
  	db  *gorm.DB
  	hub *Hub
  	svc *Service
  }

  func NewHandler(db *gorm.DB, hub *Hub, svc *Service) *Handler {
  	return &Handler{db: db, hub: hub, svc: svc}
  }

  // ════════════════════════════════════════════════════════════════════════════
  // GET /api/v1/notifications
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) List(c *gin.Context) {
  	userID := c.GetString("user_id")
  	page, perPage := paginationParams(c)

  	q := h.db.Where("user_id = ? AND deleted_at IS NULL", userID).
  		Order("created_at DESC")

  	if filter := c.Query("read"); filter == "false" {
  		q = q.Where("read = false")
  	} else if filter == "true" {
  		q = q.Where("read = true")
  	}

  	var total int64
  	q.Model(&Notification{}).Count(&total)

  	var notifs []Notification
  	q.Offset((page - 1) * perPage).Limit(perPage).Find(&notifs)

  	response.OKMeta(c, notifs, response.Meta{
  		Total:   total,
  		Page:    page,
  		PerPage: perPage,
  		Pages:   (total + int64(perPage) - 1) / int64(perPage),
  	})
  }

  // ════════════════════════════════════════════════════════════════════════════
  // GET /api/v1/notifications/unread-count
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) UnreadCount(c *gin.Context) {
  	userID := c.GetString("user_id")
  	var count int64
  	h.db.Model(&Notification{}).
  		Where("user_id = ? AND read = false AND deleted_at IS NULL", userID).
  		Count(&count)
  	response.OK(c, gin.H{"count": count})
  }

  // ════════════════════════════════════════════════════════════════════════════
  // PUT /api/v1/notifications/:id/read
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) MarkRead(c *gin.Context) {
  	userID := c.GetString("user_id")
  	idParam := c.Param("id")

  	parsedID, err := uuid.Parse(idParam)
  	if err != nil {
  		response.BadRequest(c, "invalid notification id")
  		return
  	}

  	now := time.Now()
  	result := h.db.Model(&Notification{}).
  		Where("id = ? AND user_id = ?", parsedID, userID).
  		Updates(map[string]any{"read": true, "read_at": now})

  	if result.RowsAffected == 0 {
  		response.NotFound(c, "notification")
  		return
  	}
  	response.OK(c, gin.H{"message": "Marked as read."})
  }

  // ════════════════════════════════════════════════════════════════════════════
  // PUT /api/v1/notifications/mark-all-read
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) MarkAllRead(c *gin.Context) {
  	userID := c.GetString("user_id")
  	now := time.Now()
  	h.db.Model(&Notification{}).
  		Where("user_id = ? AND read = false AND deleted_at IS NULL", userID).
  		Updates(map[string]any{"read": true, "read_at": now})
  	response.OK(c, gin.H{"message": "All notifications marked as read."})
  }

  // ════════════════════════════════════════════════════════════════════════════
  // DELETE /api/v1/notifications/:id
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) Delete(c *gin.Context) {
  	userID := c.GetString("user_id")
  	idParam := c.Param("id")

  	parsedID, err := uuid.Parse(idParam)
  	if err != nil {
  		response.BadRequest(c, "invalid notification id")
  		return
  	}

  	result := h.db.Where("id = ? AND user_id = ?", parsedID, userID).Delete(&Notification{})
  	if result.RowsAffected == 0 {
  		response.NotFound(c, "notification")
  		return
  	}
  	response.OK(c, gin.H{"message": "Notification deleted."})
  }

  // ════════════════════════════════════════════════════════════════════════════
  // POST /api/v1/notifications/register-push-token
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) RegisterPushToken(c *gin.Context) {
  	var req struct {
  		Token    string `json:"token"    binding:"required"`
  		Platform string `json:"platform" binding:"required"`
  	}
  	if err := c.ShouldBindJSON(&req); err != nil {
  		response.BadRequest(c, err.Error())
  		return
  	}

  	userID := c.GetString("user_id")
  	userUUID, _ := uuid.Parse(userID)

  	pt := PushToken{
  		UserID:   userUUID,
  		Token:    req.Token,
  		Platform: req.Platform,
  	}
  	// Upsert — token may already exist from a previous session
  	h.db.Where("token = ?", req.Token).Assign(PushToken{UserID: userUUID, Platform: req.Platform}).
  		FirstOrCreate(&pt)

  	response.Created(c, gin.H{"id": pt.ID, "message": "Push token registered."})
  }

  // ════════════════════════════════════════════════════════════════════════════
  // DELETE /api/v1/notifications/push-tokens/:id
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) DeletePushToken(c *gin.Context) {
  	userID := c.GetString("user_id")
  	tokenID := c.Param("id")

  	parsedID, err := uuid.Parse(tokenID)
  	if err != nil {
  		response.BadRequest(c, "invalid token id")
  		return
  	}

  	result := h.db.Where("id = ? AND user_id = ?", parsedID, userID).Delete(&PushToken{})
  	if result.RowsAffected == 0 {
  		response.NotFound(c, "push token")
  		return
  	}
  	response.OK(c, gin.H{"message": "Push token removed."})
  }

  // ════════════════════════════════════════════════════════════════════════════
  // GET /api/v1/notifications/preferences
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) GetPreferences(c *gin.Context) {
  	userID := c.GetString("user_id")
  	userUUID, _ := uuid.Parse(userID)

  	var prefs NotificationPreference
  	if err := h.db.First(&prefs, "user_id = ?", userUUID).Error; err != nil {
  		// Return defaults if not yet set
  		prefs = NotificationPreference{
  			UserID:               userUUID,
  			EmailNewBid:          true,
  			EmailOutbid:          true,
  			EmailMessage:         true,
  			EmailListingApproved: true,
  			PushNewBid:           true,
  			PushOutbid:           true,
  			PushMessage:          true,
  			InAppEnabled:         true,
  		}
  	}
  	response.OK(c, prefs)
  }

  // ════════════════════════════════════════════════════════════════════════════
  // PUT /api/v1/notifications/preferences
  // ════════════════════════════════════════════════════════════════════════════

  func (h *Handler) UpdatePreferences(c *gin.Context) {
  	userID := c.GetString("user_id")
  	userUUID, _ := uuid.Parse(userID)

  	var req NotificationPreference
  	if err := c.ShouldBindJSON(&req); err != nil {
  		response.BadRequest(c, err.Error())
  		return
  	}
  	req.UserID = userUUID

  	h.db.Where("user_id = ?", userUUID).Assign(req).FirstOrCreate(&req)
  	h.db.Model(&req).Updates(req)

  	response.OK(c, req)
  }

  // ════════════════════════════════════════════════════════════════════════════
  // WS /ws/notifications — real-time WebSocket
  // ════════════════════════════════════════════════════════════════════════════

  // ServeWS upgrades an HTTP connection to WebSocket and registers the client.
  // The user must pass a valid JWT as ?token= query parameter.
  func ServeWS(hub *Hub, c *gin.Context) {
  	// Validate JWT from query string (browser WS API can't set headers)
  	token := c.Query("token")
  	if token == "" {
  		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
  		return
  	}

  	userID, err := middleware.ValidateToken(token)
  	if err != nil {
  		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
  		return
  	}

  	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
  	if err != nil {
  		return
  	}

  	client := &WSClient{
  		userID: userID,
  		conn:   conn,
  		send:   make(chan []byte, 64),
  		hub:    hub,
  	}
  	hub.reg <- client

  	go client.writePump()
  	go client.readPump()
  }

  // ════════════════════════════════════════════════════════════════════════════
  // Helpers
  // ════════════════════════════════════════════════════════════════════════════

  func paginationParams(c *gin.Context) (page, perPage int) {
  	page, perPage = 1, 20
  	fmt.Sscan(c.DefaultQuery("page", "1"), &page)
  	fmt.Sscan(c.DefaultQuery("per_page", "20"), &perPage)
  	if page < 1 { page = 1 }
  	if perPage < 1 || perPage > 100 { perPage = 20 }
  	return
  }
  