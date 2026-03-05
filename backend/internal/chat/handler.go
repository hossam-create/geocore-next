package chat

import (
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"time"
)

type Handler struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
	return &Handler{db, rdb}
}

func (h *Handler) GetConversations(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	var members []ConversationMember
	h.db.Where("user_id = ?", userID).
		Preload("Conversation").
		Order("joined_at DESC").
		Find(&members)
	// Extract conversation IDs
	convIDs := make([]uuid.UUID, len(members))
	for i, m := range members {
		convIDs[i] = m.ConversationID
	}
	var convos []Conversation
	h.db.Where("id IN ?", convIDs).
		Preload("Members").
		Order("last_msg_at DESC NULLS LAST").
		Find(&convos)
	response.OK(c, convos)
}

func (h *Handler) CreateOrGetConversation(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	var req struct {
		OtherUserID string  `json:"other_user_id" binding:"required"`
		ListingID   *string `json:"listing_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	otherID, _ := uuid.Parse(req.OtherUserID)

	// Check if conversation already exists
	var existingMember ConversationMember
	subQuery := h.db.Model(&ConversationMember{}).
		Select("conversation_id").
		Where("user_id = ?", otherID)

	if err := h.db.Where("user_id = ? AND conversation_id IN (?)", userID, subQuery).
		First(&existingMember).Error; err == nil {
		var convo Conversation
		h.db.Preload("Members").First(&convo, "id = ?", existingMember.ConversationID)
		response.OK(c, convo)
		return
	}

	// Create new conversation
	convo := Conversation{ID: uuid.New()}
	if req.ListingID != nil {
		lid, _ := uuid.Parse(*req.ListingID)
		convo.ListingID = &lid
	}
	h.db.Create(&convo)
	now := time.Now()
	h.db.Create(&ConversationMember{ID: uuid.New(), ConversationID: convo.ID, UserID: userID, JoinedAt: now})
	h.db.Create(&ConversationMember{ID: uuid.New(), ConversationID: convo.ID, UserID: otherID, JoinedAt: now})
	h.db.Preload("Members").First(&convo, "id = ?", convo.ID)
	response.Created(c, convo)
}

func (h *Handler) GetMessages(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	convID, _ := uuid.Parse(c.Param("id"))

	// Verify user is a member
	var member ConversationMember
	if err := h.db.Where("conversation_id = ? AND user_id = ?", convID, userID).
		First(&member).Error; err != nil {
		response.Forbidden(c)
		return
	}

	var messages []Message
	h.db.Where("conversation_id = ?", convID).
		Order("created_at ASC").
		Limit(100).
		Find(&messages)

	// Mark as read
	go h.db.Model(&ConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, userID).
		Updates(map[string]interface{}{"unread_count": 0, "last_read_at": time.Now()})

	response.OK(c, messages)
}

func (h *Handler) SendMessage(c *gin.Context) {
	userID, _ := uuid.Parse(c.MustGet("user_id").(string))
	convID, _ := uuid.Parse(c.Param("id"))

	var req struct {
		Content string `json:"content" binding:"required"`
		Type    string `json:"type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Verify membership
	var member ConversationMember
	if err := h.db.Where("conversation_id = ? AND user_id = ?", convID, userID).
		First(&member).Error; err != nil {
		response.Forbidden(c)
		return
	}

	msg := Message{
		ID:             uuid.New(),
		ConversationID: convID,
		SenderID:       userID,
		Content:        req.Content,
		Type:           defaultStr(req.Type, "text"),
		CreatedAt:      time.Now(),
	}
	h.db.Create(&msg)

	now := time.Now()
	h.db.Model(&Conversation{}).Where("id = ?", convID).
		Update("last_msg_at", now)
	h.db.Model(&ConversationMember{}).
		Where("conversation_id = ? AND user_id != ?", convID, userID).
		UpdateColumn("unread_count", gorm.Expr("unread_count + 1"))

	response.Created(c, msg)
}

func defaultStr(s, d string) string {
	if s == "" { return d }
	return s
}
