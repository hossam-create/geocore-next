package watchlist

import (
	"time"

	"github.com/google/uuid"
)

type WatchlistItem struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	ListingID uuid.UUID `gorm:"type:uuid;primaryKey" json:"listing_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (WatchlistItem) TableName() string {
	return "watchlist_items"
}
