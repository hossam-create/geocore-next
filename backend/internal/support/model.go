package support

import (
	"time"

	"github.com/google/uuid"
)

// ContactMessage represents a contact form submission
type ContactMessage struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name      string     `gorm:"not null" json:"name"`
	Email     string     `gorm:"not null" json:"email"`
	Subject   string     `gorm:"not null" json:"subject"`
	Message   string     `gorm:"type:text;not null" json:"message"`
	Status    string     `gorm:"type:varchar(20);default:'new'" json:"status"` // new, read, resolved
	UserID    *uuid.UUID `gorm:"type:uuid" json:"user_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// TableName sets the table name
func (ContactMessage) TableName() string {
	return "contact_messages"
}

// ContactRequest is the payload for contact form submission
type ContactRequest struct {
	Name    string `json:"name" binding:"required,min=2,max=100"`
	Email   string `json:"email" binding:"required,email"`
	Subject string `json:"subject" binding:"required"`
	Message string `json:"message" binding:"required,min=20,max=5000"`
}

// SubjectOptions returns available subject options
var SubjectOptions = []struct {
	Value string `json:"value"`
	Label string `json:"label"`
}{
	{"general", "General Inquiry"},
	{"order", "Order Issue"},
	{"payment", "Payment Problem"},
	{"account", "Account Help"},
	{"selling", "Selling Question"},
	{"technical", "Technical Support"},
	{"feedback", "Feedback"},
	{"other", "Other"},
}

// TicketStatus represents the status of a support ticket
type TicketStatus string

const (
	TicketStatusOpen       TicketStatus = "open"
	TicketStatusInProgress TicketStatus = "in_progress"
	TicketStatusResolved   TicketStatus = "resolved"
	TicketStatusClosed     TicketStatus = "closed"
)

// TicketPriority represents the priority level
type TicketPriority string

const (
	PriorityLow    TicketPriority = "low"
	PriorityNormal TicketPriority = "normal"
	PriorityHigh   TicketPriority = "high"
	PriorityUrgent TicketPriority = "urgent"
)

// SupportTicket represents a support ticket
type SupportTicket struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"user_id"`
	Subject   string         `gorm:"not null" json:"subject"`
	Status    TicketStatus   `gorm:"type:varchar(20);default:'open'" json:"status"`
	Priority  TicketPriority `gorm:"type:varchar(20);default:'normal'" json:"priority"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`

	// Relations
	Messages []TicketMessage `gorm:"foreignKey:TicketID" json:"messages,omitempty"`
	User     *UserInfo       `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TicketMessage represents a message in a support ticket
type TicketMessage struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	TicketID  uuid.UUID `gorm:"type:uuid;not null;index" json:"ticket_id"`
	SenderID  uuid.UUID `gorm:"type:uuid;not null" json:"sender_id"`
	IsAdmin   bool      `gorm:"default:false" json:"is_admin"`
	Message   string    `gorm:"type:text;not null" json:"message"`
	CreatedAt time.Time `json:"created_at"`

	// Relations
	Sender *UserInfo `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
}

// UserInfo contains basic user details for ticket responses
type UserInfo struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Name string    `gorm:"not null" json:"name"`
}

// TableName sets the table name for SupportTicket
func (SupportTicket) TableName() string {
	return "support_tickets"
}

// TableName sets the table name for TicketMessage
func (TicketMessage) TableName() string {
	return "ticket_messages"
}

// TicketCreateRequest is the payload for creating a new ticket
type TicketCreateRequest struct {
	Subject string `json:"subject" binding:"required,min=5,max=200"`
	Message string `json:"message" binding:"required,min=20,max=5000"`
}

// TicketMessageRequest is the payload for adding a message
type TicketMessageRequest struct {
	Message string `json:"message" binding:"required,min=10,max=5000"`
}

// TicketUpdateRequest is the payload for updating ticket status
type TicketUpdateRequest struct {
	Status   *TicketStatus   `json:"status,omitempty"`
	Priority *TicketPriority `json:"priority,omitempty"`
}
