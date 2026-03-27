package chat

import (
	"time"
	"github.com/google/uuid"
)

type Conversation struct {
	ID        uuid.UUID            `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ListingID *uuid.UUID           `gorm:"type:uuid;index" json:"listing_id,omitempty"`
	LastMsgAt *time.Time           `json:"last_message_at,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
	Members   []ConversationMember `gorm:"foreignKey:ConversationID" json:"members,omitempty"`
	Messages  []Message            `gorm:"foreignKey:ConversationID" json:"messages,omitempty"`
}

type ConversationMember struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ConversationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"conversation_id"`
	UserID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	JoinedAt       time.Time  `json:"joined_at"`
	LastReadAt     *time.Time `json:"last_read_at,omitempty"`
	UnreadCount    int        `gorm:"default:0" json:"unread_count"`
}

type Message struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ConversationID uuid.UUID  `gorm:"type:uuid;not null;index" json:"conversation_id"`
	SenderID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"sender_id"`
	Content        string     `gorm:"type:text;not null" json:"content"`
	Type           string     `gorm:"default:text" json:"type"` // text | image | offer
	ReadAt         *time.Time `json:"read_at,omitempty"`
	CreatedAt      time.Time  `gorm:"index" json:"created_at"`
}
