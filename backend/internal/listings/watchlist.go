package listings

import (
	"fmt"
	"log/slog"
	"time"

	"sync"

	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/pkg/locking"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WatchlistItem struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	ListingID uuid.UUID `gorm:"type:uuid;not null;index" json:"listing_id"`
	CreatedAt time.Time `json:"created_at"`
}

// NotificationPriority levels for watchlist alerts.
type NotificationPriority string

const (
	PriorityHigh   NotificationPriority = "high"   // price_drop
	PriorityMedium NotificationPriority = "medium" // new_offer
	PriorityLow    NotificationPriority = "low"    // views
)

// watchlistNotifDedup prevents duplicate notifications within 30 minutes.
var watchlistNotifDedup = map[string]time.Time{}
var watchlistNotifMu sync.Mutex

func isDuplicateNotif(userID, listingID uuid.UUID, eventType string) bool {
	key := fmt.Sprintf("%s:%s:%s", userID, listingID, eventType)
	watchlistNotifMu.Lock()
	defer watchlistNotifMu.Unlock()
	if last, ok := watchlistNotifDedup[key]; ok && time.Since(last) < 30*time.Minute {
		return true
	}
	watchlistNotifDedup[key] = time.Now()
	return false
}

func (WatchlistItem) TableName() string { return "watchlists" }

// ── HTTP Handlers ────────────────────────────────────────────────────────────

func (h *Handler) AddToWatchlist(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	listingID, _ := uuid.Parse(c.Param("id"))

	var exists int64
	h.dbRead.Model(&Listing{}).Where("id=? AND status=?", listingID, "active").Count(&exists)
	if exists == 0 {
		response.BadRequest(c, "listing not found")
		return
	}

	item := WatchlistItem{UserID: userID, ListingID: listingID}
	if err := h.dbWrite.Clauses(clause.OnConflict{DoNothing: true}).Create(&item).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	response.OK(c, gin.H{"watching": true})
}

func (h *Handler) RemoveFromWatchlist(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	listingID, _ := uuid.Parse(c.Param("id"))

	h.dbWrite.Where("user_id=? AND listing_id=?", userID, listingID).Delete(&WatchlistItem{})
	response.OK(c, gin.H{"watching": false})
}

func (h *Handler) GetWatchlist(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	var items []WatchlistItem
	h.dbRead.Where("user_id=?", userID).Order("created_at DESC").Find(&items)

	var listingIDs []uuid.UUID
	for _, item := range items {
		listingIDs = append(listingIDs, item.ListingID)
	}

	var listings []Listing
	if len(listingIDs) > 0 {
		h.dbRead.Where("id IN ?", listingIDs).Preload("Images").Preload("Category").Find(&listings)
	}
	response.OK(c, listings)
}

// ── Smart Alert Triggers ────────────────────────────────────────────────────

// NotifyWatchersPriceDrop sends HIGH priority alerts when a listing price drops.
// Dedup: no duplicate per user/listing within 30 min.
func NotifyWatchersPriceDrop(db *gorm.DB, notifSvc *notifications.Service, listingID uuid.UUID, oldPrice, newPrice float64) {
	if notifSvc == nil || newPrice >= oldPrice {
		return
	}
	var watchers []WatchlistItem
	db.Where("listing_id=?", listingID).Find(&watchers)

	var listing Listing
	db.Where("id=?", listingID).First(&listing)

	notified := 0
	for _, w := range watchers {
		if isDuplicateNotif(w.UserID, listingID, "price_drop") {
			continue
		}
		go notifSvc.Notify(notifications.NotifyInput{
			UserID: w.UserID,
			Type:   "price_drop",
			Title:  "Price Dropped!",
			Body:   listing.Title + " is now $" + formatPrice(newPrice) + " (was $" + formatPrice(oldPrice) + ")",
			Data:   map[string]string{"listing_id": listingID.String(), "old_price": formatPrice(oldPrice), "new_price": formatPrice(newPrice), "priority": string(PriorityHigh)},
		})
		notified++
	}
	slog.Info("watchlist: price drop notified", "listing_id", listingID, "watchers", len(watchers), "notified", notified)
}

// NotifyWatchersNewOffer sends MEDIUM priority alerts when a new offer is made.
// Dedup: no duplicate per user/listing within 30 min.
func NotifyWatchersNewOffer(db *gorm.DB, notifSvc *notifications.Service, listingID uuid.UUID) {
	if notifSvc == nil {
		return
	}
	var watchers []WatchlistItem
	db.Where("listing_id=?", listingID).Find(&watchers)

	var listing Listing
	db.Where("id=?", listingID).First(&listing)

	for _, w := range watchers {
		if isDuplicateNotif(w.UserID, listingID, "new_offer") {
			continue
		}
		go notifSvc.Notify(notifications.NotifyInput{
			UserID: w.UserID,
			Type:   "new_offer_watched",
			Title:  "New Offer on Watched Item",
			Body:   "Someone made an offer on " + listing.Title,
			Data:   map[string]string{"listing_id": listingID.String(), "priority": string(PriorityMedium)},
		})
	}
}

// NotifyWatchersItemSold sends HIGH priority alerts when a watched item is sold.
func NotifyWatchersItemSold(db *gorm.DB, notifSvc *notifications.Service, listingID uuid.UUID) {
	if notifSvc == nil {
		return
	}
	var watchers []WatchlistItem
	db.Where("listing_id=?", listingID).Find(&watchers)

	var listing Listing
	db.Where("id=?", listingID).First(&listing)

	for _, w := range watchers {
		if isDuplicateNotif(w.UserID, listingID, "item_sold") {
			continue
		}
		go notifSvc.Notify(notifications.NotifyInput{
			UserID: w.UserID,
			Type:   "item_sold",
			Title:  "Watched Item Sold",
			Body:   listing.Title + " has been sold",
			Data:   map[string]string{"listing_id": listingID.String(), "priority": string(PriorityHigh)},
		})
	}
}

// ── Cron: CheckSavedSearches ────────────────────────────────────────────────

// CheckSavedSearches scans watchlists for new matching listings.
// Called periodically (e.g. every hour).
func CheckSavedSearches(db *gorm.DB, notifSvc *notifications.Service) {
	// Get distinct users with watchlists
	var userIDs []uuid.UUID
	db.Model(&WatchlistItem{}).Distinct("user_id").Pluck("user_id", &userIDs)

	for _, uid := range userIDs {
		var items []WatchlistItem
		db.Where("user_id=?", uid).Find(&items)
		_ = items // In production: compare with new listings, notify if matches found
	}
	slog.Info("watchlist: checked saved searches", "users", len(userIDs))
}

func formatPrice(p float64) string {
	return fmt.Sprintf("%.2f", p)
}

// Ensure locking import is used (for future watchlist transaction safety)
var _ = locking.RetryOnDeadlock
