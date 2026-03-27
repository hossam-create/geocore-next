package auctions

import (
        "time"

        "github.com/google/uuid"
        "gorm.io/gorm"
)

type Auction struct {
        ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
        ListingID      uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"listing_id"`
        SellerID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"seller_id"`
        StartPrice     float64        `gorm:"not null" json:"start_price"`
        ReservePrice   *float64       `json:"reserve_price,omitempty"`
        BuyNowPrice    *float64       `json:"buy_now_price,omitempty"`
        CurrentBid     float64        `gorm:"default:0" json:"current_bid"`
        BidCount       int            `gorm:"default:0" json:"bid_count"`
        WinnerID       *uuid.UUID     `gorm:"type:uuid" json:"winner_id,omitempty"`
        Status         string         `gorm:"default:active;index" json:"status"` // active | ended | cancelled | sold
        StartsAt       time.Time      `json:"starts_at"`
        EndsAt         time.Time      `gorm:"index" json:"ends_at"`
        ExtensionCount int            `gorm:"default:0" json:"extension_count"`
        Currency       string         `gorm:"default:USD" json:"currency"`
        CreatedAt      time.Time      `json:"created_at"`
        UpdatedAt      time.Time      `json:"updated_at"`
        DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
        Bids           []Bid          `gorm:"foreignKey:AuctionID" json:"bids,omitempty"`
}

type Bid struct {
        ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
        AuctionID uuid.UUID `gorm:"type:uuid;not null;index" json:"auction_id"`
        UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
        Amount    float64   `gorm:"not null" json:"amount"`
        IsAuto    bool      `gorm:"default:false" json:"is_auto"`
        MaxAmount *float64  `json:"max_amount,omitempty"` // for auto-bidding
        PlacedAt  time.Time `gorm:"index" json:"placed_at"`
}
