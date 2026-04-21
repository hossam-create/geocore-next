package growth

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/crowdshipping"
	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ════════════════════════════════════════════════════════════════════════════
// Liquidity Bootstrap Engine
// Seeds initial supply/demand for marketplace launch.
// ════════════════════════════════════════════════════════════════════════════

// GhostListing is an admin-controlled placeholder listing to seed supply.
type GhostListing struct {
	ID                 uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Title              string          `gorm:"size:255;not null" json:"title"`
	Description        string          `gorm:"type:text" json:"description"`
	Price              decimal.Decimal `gorm:"type:decimal(12,2);not null" json:"price"`
	Currency           string          `gorm:"size:10;not null;default:'USD'" json:"currency"`
	Category           string          `gorm:"size:100" json:"category"`
	OriginCountry      string          `gorm:"size:100;not null" json:"origin_country"`
	DestCountry        string          `gorm:"size:100;not null" json:"dest_country"`
	IsPlatformAssisted bool            `gorm:"not null;default:true" json:"is_platform_assisted"`
	IsActive           bool            `gorm:"not null;default:true" json:"is_active"`
	CreatedBy          uuid.UUID       `gorm:"type:uuid" json:"created_by"`
	CreatedAt          time.Time       `json:"created_at"`
}

func (GhostListing) TableName() string { return "ghost_listings" }

// PlatformTraveler is an internal traveler used for early-stage matching.
type PlatformTraveler struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID        uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	Name          string    `gorm:"size:255;not null" json:"name"`
	Reputation    int       `gorm:"not null;default:80" json:"reputation"`
	OriginCountry string    `gorm:"size:100;not null" json:"origin_country"`
	DestCountry   string    `gorm:"size:100;not null" json:"dest_country"`
	IsActive      bool      `gorm:"not null;default:true" json:"is_active"`
	IsInternal    bool      `gorm:"not null;default:true" json:"is_internal"`
	CreatedAt     time.Time `json:"created_at"`
}

func (PlatformTraveler) TableName() string { return "platform_travelers" }

// SeedGhostListings creates admin-controlled placeholder listings.
func SeedGhostListings(db *gorm.DB, adminID uuid.UUID, listings []GhostListing) error {
	return locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		for i := range listings {
			l := &listings[i]
			l.ID = uuid.New()
			l.CreatedBy = adminID
			l.IsPlatformAssisted = true
			l.IsActive = true
			if err := tx.Create(l).Error; err != nil {
				slog.Error("bootstrap: failed to seed ghost listing", "title", l.Title, "error", err)
				return err
			}
		}
		slog.Info("bootstrap: seeded ghost listings", "count", len(listings))
		return nil
	})
}

// SeedPlatformTravelers creates internal travelers with high reputation for early matching.
func SeedPlatformTravelers(db *gorm.DB, travelers []PlatformTraveler) error {
	return locking.RetryOnDeadlock(db, func(tx *gorm.DB) error {
		for i := range travelers {
			t := &travelers[i]
			t.ID = uuid.New()
			t.IsActive = true
			t.IsInternal = true
			t.Reputation = 80
			if err := tx.Create(t).Error; err != nil {
				slog.Error("bootstrap: failed to seed platform traveler", "name", t.Name, "error", err)
				return err
			}
		}
		slog.Info("bootstrap: seeded platform travelers", "count", len(travelers))
		return nil
	})
}

// SmartAutoFill checks for delivery requests with no offers and triggers auto-generation.
// Called periodically (e.g. every 2 minutes).
func SmartAutoFill(db *gorm.DB, notifSvc *notifications.Service) error {
	cutoff := time.Now().Add(-2 * time.Minute)

	// Find delivery requests with no offers created in the last window
	var unattended []crowdshipping.DeliveryRequest
	db.Where("status = ? AND created_at <= ?", crowdshipping.DeliveryPending, cutoff).
		Find(&unattended)

	var autoFilled int
	for _, dr := range unattended {
		var offerCount int64
		db.Table("traveler_offers").
			Where("delivery_request_id = ? AND deleted_at IS NULL", dr.ID).
			Count(&offerCount)

		if offerCount == 0 {
			// Trigger auto-offer generation
			if err := crowdshipping.GenerateAutoOffers(db, notifSvc, dr.ID); err != nil {
				slog.Warn("bootstrap: auto-fill failed", "request_id", dr.ID, "error", err)
				continue
			}
			autoFilled++

			// Fallback: if still no offers, use platform traveler
			if notifSvc != nil {
				go notifSvc.Notify(notifications.NotifyInput{
					UserID: dr.BuyerID,
					Type:   "auto_fill_triggered",
					Title:  "Finding Travelers",
					Body:   fmt.Sprintf("We're finding travelers for your request: %s", dr.ItemName),
					Data:   map[string]string{"request_id": dr.ID.String()},
				})
			}
		}
	}

	if autoFilled > 0 {
		slog.Info("bootstrap: auto-filled requests", "count", autoFilled)
	}
	return nil
}

// FindPlatformTravelerFallback finds a platform traveler for a route if no real travelers exist.
func FindPlatformTravelerFallback(db *gorm.DB, originCountry, destCountry string) *PlatformTraveler {
	var pt PlatformTraveler
	if db.Where("origin_country = ? AND dest_country = ? AND is_active = ? AND is_internal = ?",
		originCountry, destCountry, true, true).First(&pt).Error != nil {
		return nil
	}
	return &pt
}

// GetBootstrapStats returns current bootstrap state metrics.
func GetBootstrapStats(db *gorm.DB) map[string]interface{} {
	var ghostCount int64
	var travelerCount int64
	var activeGhosts int64

	db.Model(&GhostListing{}).Count(&ghostCount)
	db.Model(&GhostListing{}).Where("is_active = ?", true).Count(&activeGhosts)
	db.Model(&PlatformTraveler{}).Where("is_active = ? AND is_internal = ?", true, true).Count(&travelerCount)

	return map[string]interface{}{
		"ghost_listings":        ghostCount,
		"active_ghost_listings": activeGhosts,
		"platform_travelers":    travelerCount,
	}
}

// CleanupGhostData marks all ghost listings and platform travelers as inactive
// once real marketplace activity reaches sufficient levels.
func CleanupGhostData(db *gorm.DB) error {
	var realCount int64
	db.Table("listings").Where("status = ? AND deleted_at IS NULL", "active").Count(&realCount)

	var ghostCount int64
	db.Model(&GhostListing{}).Where("is_active = ?", true).Count(&ghostCount)

	if realCount > ghostCount && realCount >= 50 {
		db.Model(&GhostListing{}).Where("is_active = ?", true).Update("is_active", false)
		db.Model(&PlatformTraveler{}).Where("is_internal = ? AND is_active = ?", true, true).Update("is_active", false)
		slog.Info("bootstrap: ghost data deactivated — real marketplace active", "real_listings", realCount)
	}

	return nil
}
