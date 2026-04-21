package listings

import (
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ListingView records a single page view event.
type ListingView struct {
	ID        uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	ListingID uuid.UUID  `gorm:"type:uuid;not null;index" json:"listing_id"`
	ViewerID  *uuid.UUID `gorm:"type:uuid" json:"viewer_id,omitempty"`
	IPHash    string     `gorm:"size:64" json:"-"`
	UserAgent string     `gorm:"size:500" json:"-"`
	ViewedAt  time.Time  `gorm:"not null;default:now();index" json:"viewed_at"`
}

func (ListingView) TableName() string { return "listing_views" }

// DailyViewRow is used to read from the listing_daily_views materialized view.
type DailyViewRow struct {
	ListingID   uuid.UUID `json:"listing_id"`
	ViewDate    time.Time `json:"view_date"`
	TotalViews  int       `json:"total_views"`
	UniqueViews int       `json:"unique_views"`
}

func (DailyViewRow) TableName() string { return "listing_daily_views" }

// RecordView records a view and increments the listing's view_count.
func (h *Handler) RecordView(c *gin.Context) {
	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid listing ID")
		return
	}

	// Check listing exists
	var exists int64
	h.db.Model(&Listing{}).Where("id = ?", listingID).Count(&exists)
	if exists == 0 {
		response.NotFound(c, "Listing")
		return
	}

	// Get viewer identity
	var viewerID *uuid.UUID
	if uid, ok := c.Get("user_id"); ok {
		if uidStr, ok := uid.(string); ok {
			parsed, _ := uuid.Parse(uidStr)
			viewerID = &parsed
		}
	}

	ipHash := hashIP(c.ClientIP())
	ua := c.GetHeader("User-Agent")
	if len(ua) > 500 {
		ua = ua[:500]
	}

	// Deduplicate: same viewer/IP within last 30 minutes doesn't count
	var recent int64
	dedup := h.db.Model(&ListingView{}).Where("listing_id = ? AND viewed_at > ?", listingID, time.Now().Add(-30*time.Minute))
	if viewerID != nil {
		dedup = dedup.Where("viewer_id = ?", *viewerID)
	} else {
		dedup = dedup.Where("ip_hash = ?", ipHash)
	}
	dedup.Count(&recent)

	if recent > 0 {
		response.OK(c, gin.H{"recorded": false, "reason": "duplicate"})
		return
	}

	view := ListingView{
		ListingID: listingID,
		ViewerID:  viewerID,
		IPHash:    ipHash,
		UserAgent: ua,
		ViewedAt:  time.Now(),
	}
	h.db.Create(&view)

	// Increment counter
	h.db.Model(&Listing{}).Where("id = ?", listingID).
		UpdateColumn("view_count", gorm.Expr("view_count + 1"))

	response.OK(c, gin.H{"recorded": true})
}

// GetViewAnalytics returns daily view stats for a specific listing (owner only).
func (h *Handler) GetViewAnalytics(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	listingID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	// Verify ownership
	var listing Listing
	if err := h.db.Where("id = ? AND user_id = ?", listingID, userID).First(&listing).Error; err != nil {
		response.NotFound(c, "Listing owned by you")
		return
	}

	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days < 1 || days > 365 {
		days = 30
	}

	since := time.Now().AddDate(0, 0, -days)

	// Query raw from listing_views (not materialized view, which needs REFRESH)
	type DayStat struct {
		ViewDate    string `json:"date"`
		TotalViews  int    `json:"total"`
		UniqueViews int    `json:"unique"`
	}
	var stats []DayStat
	h.db.Raw(`
		SELECT
			TO_CHAR(viewed_at, 'YYYY-MM-DD') AS view_date,
			COUNT(*) AS total_views,
			COUNT(DISTINCT COALESCE(viewer_id::text, ip_hash)) AS unique_views
		FROM listing_views
		WHERE listing_id = ? AND viewed_at >= ?
		GROUP BY TO_CHAR(viewed_at, 'YYYY-MM-DD')
		ORDER BY view_date
	`, listingID, since).Scan(&stats)

	// Summary
	var totalViews, uniqueViews int
	for _, s := range stats {
		totalViews += s.TotalViews
		uniqueViews += s.UniqueViews
	}

	response.OK(c, gin.H{
		"listing_id":   listingID,
		"days":         days,
		"total_views":  totalViews,
		"unique_views": uniqueViews,
		"daily":        stats,
	})
}

// GetMyViewsSummary returns daily view totals across all user's listings (for dashboard chart).
func (h *Handler) GetMyViewsSummary(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days < 1 || days > 365 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)

	type DayStat struct {
		ViewDate    string `json:"date"`
		TotalViews  int    `json:"total"`
		UniqueViews int    `json:"unique"`
	}
	var stats []DayStat
	h.db.Raw(`
		SELECT
			TO_CHAR(lv.viewed_at, 'YYYY-MM-DD') AS view_date,
			COUNT(*) AS total_views,
			COUNT(DISTINCT COALESCE(lv.viewer_id::text, lv.ip_hash)) AS unique_views
		FROM listing_views lv
		JOIN listings l ON l.id = lv.listing_id
		WHERE l.user_id = ? AND lv.viewed_at >= ?
		GROUP BY TO_CHAR(lv.viewed_at, 'YYYY-MM-DD')
		ORDER BY view_date
	`, userID, since).Scan(&stats)

	response.OK(c, stats)
}

// ExportViewsCSV exports view analytics as CSV for the authenticated user's listings.
func (h *Handler) ExportViewsCSV(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days < 1 || days > 365 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)

	type Row struct {
		ListingID   string `json:"listing_id"`
		Title       string `json:"title"`
		ViewDate    string `json:"date"`
		TotalViews  int    `json:"total"`
		UniqueViews int    `json:"unique"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
			l.id AS listing_id,
			l.title,
			TO_CHAR(lv.viewed_at, 'YYYY-MM-DD') AS view_date,
			COUNT(*) AS total_views,
			COUNT(DISTINCT COALESCE(lv.viewer_id::text, lv.ip_hash)) AS unique_views
		FROM listing_views lv
		JOIN listings l ON l.id = lv.listing_id
		WHERE l.user_id = ? AND lv.viewed_at >= ?
		GROUP BY l.id, l.title, TO_CHAR(lv.viewed_at, 'YYYY-MM-DD')
		ORDER BY l.title, view_date
	`, userID, since).Scan(&rows)

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition",
		fmt.Sprintf(`attachment; filename="listing-views-%s.csv"`, time.Now().Format("2006-01-02")))

	w := csv.NewWriter(c.Writer)
	defer w.Flush()

	w.Write([]string{"listing_id", "title", "date", "total_views", "unique_views"})
	for _, r := range rows {
		w.Write([]string{r.ListingID, r.Title, r.ViewDate,
			strconv.Itoa(r.TotalViews), strconv.Itoa(r.UniqueViews)})
	}
}

func hashIP(ip string) string {
	h := sha256.Sum256([]byte(ip + "geocore-salt"))
	return fmt.Sprintf("%x", h[:16])
}
