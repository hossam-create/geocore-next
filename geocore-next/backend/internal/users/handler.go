package users

import (
        "github.com/geocore-next/backend/pkg/response"
        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "github.com/redis/go-redis/v9"
        "gorm.io/gorm"
)

type Handler struct {
        db  *gorm.DB
        rdb *redis.Client
}

func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
        return &Handler{db, rdb}
}

func (h *Handler) GetProfile(c *gin.Context) {
        id, err := uuid.Parse(c.Param("id"))
        if err != nil {
                response.BadRequest(c, "Invalid user ID")
                return
        }
        var user User
        if err := h.db.First(&user, "id = ?", id).Error; err != nil {
                response.NotFound(c, "User")
                return
        }
        response.OK(c, user.ToPublic())
}

func (h *Handler) UpdateMe(c *gin.Context) {
        userID := c.MustGet("user_id").(string)
        var req struct {
                Name     string `json:"name"`
                Bio      string `json:"bio"`
                Location string `json:"location"`
                Language string `json:"language"`
                Currency string `json:"currency"`
        }
        if err := c.ShouldBindJSON(&req); err != nil {
                response.BadRequest(c, err.Error())
                return
        }
        var user User
        if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
                response.NotFound(c, "User")
                return
        }
        if req.Name != "" {
                user.Name = req.Name
        }
        user.Bio = req.Bio
        user.Location = req.Location
        if req.Language != "" {
                user.Language = req.Language
        }
        if req.Currency != "" {
                user.Currency = req.Currency
        }
        h.db.Save(&user)
        response.OK(c, user)
}

func (h *Handler) GetMe(c *gin.Context) {
        userID := c.MustGet("user_id").(string)
        var user User
        if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
                response.NotFound(c, "User")
                return
        }
        response.OK(c, user)
}

// GetStats — GET /api/v1/users/me/stats
// Returns aggregated seller statistics: listing counts, revenue, wallet balance,
// store visits, and rating. All counts are derived live from the database.
func (h *Handler) GetStats(c *gin.Context) {
        userIDStr := c.MustGet("user_id").(string)
        uid, err := uuid.Parse(userIDStr)
        if err != nil {
                response.BadRequest(c, "Invalid user ID")
                return
        }

        var user User
        if err := h.db.Select("id, balance, sold_count, rating, review_count").
                First(&user, "id = ?", uid).Error; err != nil {
                response.NotFound(c, "User")
                return
        }

        var totalListings, activeListings int64
        h.db.Table("listings").
                Where("user_id = ? AND deleted_at IS NULL", uid).
                Count(&totalListings)
        h.db.Table("listings").
                Where("user_id = ? AND status = ? AND deleted_at IS NULL", uid, "active").
                Count(&activeListings)

        var totalRevenue float64
        h.db.Table("escrow_accounts").
                Where("seller_id = ? AND status = ?", uid, "released").
                Select("COALESCE(SUM(amount), 0)").
                Scan(&totalRevenue)

        var storeViews int64
        h.db.Table("storefronts").
                Where("user_id = ? AND deleted_at IS NULL", uid).
                Select("COALESCE(SUM(views), 0)").
                Scan(&storeViews)

        response.OK(c, gin.H{
                "total_listings":  totalListings,
                "active_listings": activeListings,
                "total_revenue":   totalRevenue,
                "wallet_balance":  user.Balance,
                "sold_count":      user.SoldCount,
                "store_visits":    storeViews,
                "rating":          user.Rating,
                "review_count":    user.ReviewCount,
        })
}
