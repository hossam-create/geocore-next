package listings

import (
        "context"
        "log"
        "time"

        "gorm.io/gorm"
)

// StartListingExpiryWorker runs daily and sets listings whose expires_at has
// passed (and are still active) to status "expired".
func StartListingExpiryWorker(ctx context.Context, db *gorm.DB) {
        ticker := time.NewTicker(24 * time.Hour)
        defer ticker.Stop()

        log.Println("[listing-scheduler] listing-expiry worker started")

        // Run once immediately on startup, then every 24 hours
        expireListings(db)
        demoteExpiredBoosts(db)

        for {
                select {
                case <-ctx.Done():
                        log.Println("[listing-scheduler] listing-expiry worker stopped")
                        return
                case <-ticker.C:
                        expireListings(db)
                        demoteExpiredBoosts(db)
                }
        }
}

func expireListings(db *gorm.DB) {
        result := db.Model(&Listing{}).
                Where("status = ? AND expires_at IS NOT NULL AND expires_at < ?", "active", time.Now()).
                Update("status", "expired")

        if result.Error != nil {
                log.Printf("[listing-scheduler] error expiring listings: %v", result.Error)
                return
        }

        if result.RowsAffected > 0 {
                log.Printf("[listing-scheduler] expired %d listing(s)", result.RowsAffected)
        }
}

// demoteExpiredBoosts clears is_featured on listings whose featured_until has passed.
func demoteExpiredBoosts(db *gorm.DB) {
	result := db.Model(&Listing{}).
		Where("is_featured = true AND featured_until IS NOT NULL AND featured_until < ?", time.Now()).
		Updates(map[string]interface{}{"is_featured": false})

	if result.Error != nil {
		log.Printf("[listing-scheduler] error demoting expired boosts: %v", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		log.Printf("[listing-scheduler] demoted %d expired boost(s)", result.RowsAffected)
	}
}
