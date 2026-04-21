package support

import (
	"os"
	"time"

	"github.com/geocore-next/backend/pkg/email"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// SubmitContact handles contact form submission
func (h *Handler) SubmitContact(c *gin.Context) {
	var req ContactRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Get user ID if authenticated
	var userID *uuid.UUID
	if uid := c.GetString("user_id"); uid != "" {
		if parsed, err := uuid.Parse(uid); err == nil {
			userID = &parsed
		}
	}

	message := ContactMessage{
		ID:        uuid.New(),
		Name:      req.Name,
		Email:     req.Email,
		Subject:   req.Subject,
		Message:   req.Message,
		Status:    "new",
		UserID:    userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.db.Create(&message).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	adminEmail := os.Getenv("SUPPORT_ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = os.Getenv("SMTP_FROM")
	}
	if adminEmail != "" {
		go email.SendSupportContactEmail(adminEmail, req.Name, req.Email, req.Subject, req.Message)
	}

	response.Created(c, gin.H{
		"message": "Your message has been submitted successfully. We'll get back to you soon.",
		"id":      message.ID,
	})
}

// GetSubjects returns available subject options
func (h *Handler) GetSubjects(c *gin.Context) {
	response.OK(c, SubjectOptions)
}

// GetMessages returns contact messages (admin only - for future use)
func (h *Handler) GetMessages(c *gin.Context) {
	var messages []ContactMessage
	if err := h.db.Order("created_at DESC").Limit(50).Find(&messages).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, messages)
}

// CreateTicket creates a new support ticket
func (h *Handler) CreateTicket(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Unauthorized(c)
		return
	}

	var req TicketCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ticket := SupportTicket{
		ID:        uuid.New(),
		UserID:    userID,
		Subject:   req.Subject,
		Status:    TicketStatusOpen,
		Priority:  PriorityNormal,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.db.Create(&ticket).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Create initial message
	message := TicketMessage{
		ID:        uuid.New(),
		TicketID:  ticket.ID,
		SenderID:  userID,
		IsAdmin:   false,
		Message:   req.Message,
		CreatedAt: time.Now(),
	}

	if err := h.db.Create(&message).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Fetch with messages
	h.db.Preload("Messages").First(&ticket, "id = ?", ticket.ID)
	response.Created(c, ticket)
}

// GetTickets returns user's support tickets
func (h *Handler) GetTickets(c *gin.Context) {
	userID := c.GetString("user_id")

	var tickets []SupportTicket
	if err := h.db.Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&tickets).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, tickets)
}

// GetTicket returns a single ticket with messages
func (h *Handler) GetTicket(c *gin.Context) {
	userID := c.GetString("user_id")
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ticket ID")
		return
	}

	var ticket SupportTicket
	err = h.db.Where("id = ? AND user_id = ?", ticketID, userID).
		Preload("Messages", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC")
		}).
		Preload("Messages.Sender").
		First(&ticket).Error

	if err != nil {
		response.NotFound(c, "Ticket")
		return
	}

	response.OK(c, ticket)
}

// AddTicketMessage adds a reply to a ticket
func (h *Handler) AddTicketMessage(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Unauthorized(c)
		return
	}

	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ticket ID")
		return
	}

	var req TicketMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Verify ticket ownership
	var ticket SupportTicket
	if err := h.db.Where("id = ? AND user_id = ?", ticketID, userID).First(&ticket).Error; err != nil {
		response.NotFound(c, "Ticket")
		return
	}

	if ticket.Status == TicketStatusClosed {
		response.BadRequest(c, "Cannot add message to closed ticket")
		return
	}

	message := TicketMessage{
		ID:        uuid.New(),
		TicketID:  ticketID,
		SenderID:  userID,
		IsAdmin:   false,
		Message:   req.Message,
		CreatedAt: time.Now(),
	}

	if err := h.db.Create(&message).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Update ticket timestamp
	h.db.Model(&ticket).Update("updated_at", time.Now())

	response.Created(c, message)
}

// UpdateTicket updates ticket status (admin only)
func (h *Handler) UpdateTicket(c *gin.Context) {
	ticketID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ticket ID")
		return
	}

	var req TicketUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	updates := map[string]interface{}{"updated_at": time.Now()}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}

	result := h.db.Model(&SupportTicket{}).Where("id = ?", ticketID).Updates(updates)
	if result.Error != nil {
		response.InternalError(c, result.Error)
		return
	}

	if result.RowsAffected == 0 {
		response.NotFound(c, "Ticket")
		return
	}

	response.OK(c, gin.H{"message": "Ticket updated"})
}
