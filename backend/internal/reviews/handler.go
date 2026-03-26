package reviews

import (
        "net/http"

        "github.com/geocore-next/backend/internal/users"
        "github.com/geocore-next/backend/pkg/response"
        "github.com/gin-gonic/gin"
        "github.com/google/uuid"
        "gorm.io/gorm"
)

// eligibleToReview checks whether the reviewer has completed a successful
// purchase of a listing owned by sellerID. This prevents arbitrary review spam.
func (h *Handler) eligibleToReview(reviewerID, sellerID uuid.UUID) bool {
        var count int64
        h.db.Table("payments").
                Joins("JOIN listings ON listings.id = payments.listing_id").
                Where("payments.user_id = ? AND listings.user_id = ? AND payments.status = 'succeeded'",
                        reviewerID, sellerID).
                Count(&count)
        return count > 0
}

type Handler struct {
        db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
        return &Handler{db: db}
}

// GET /users/:id/reviews — list reviews for a seller
func (h *Handler) List(c *gin.Context) {
        sellerID, err := uuid.Parse(c.Param("id"))
        if err != nil {
                response.BadRequest(c, "invalid seller ID")
                return
        }

        var reviews []Review
        if err := h.db.Where("seller_id = ? AND deleted_at IS NULL", sellerID).
                Order("created_at DESC").
                Limit(50).
                Find(&reviews).Error; err != nil {
                response.InternalError(c, err)
                return
        }

        response.OK(c, reviews)
}

// POST /users/:id/reviews — submit a review for a seller
func (h *Handler) Create(c *gin.Context) {
        reviewerID, err := uuid.Parse(c.GetString("user_id"))
        if err != nil {
                response.Unauthorized(c)
                return
        }

        sellerID, err := uuid.Parse(c.Param("id"))
        if err != nil {
                response.BadRequest(c, "invalid seller ID")
                return
        }

        if reviewerID == sellerID {
                c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "you cannot review yourself"})
                return
        }

        var req struct {
                Rating  int    `json:"rating" binding:"required,min=1,max=5"`
                Comment string `json:"comment"`
        }
        if err := c.ShouldBindJSON(&req); err != nil {
                response.BadRequest(c, err.Error())
                return
        }

        // Eligibility gate: reviewer must have a completed purchase from this seller
        if !h.eligibleToReview(reviewerID, sellerID) {
                c.JSON(http.StatusForbidden, gin.H{"error": "you can only review sellers you have completed a purchase from"})
                return
        }

        // Check if reviewer already reviewed this seller
        var existing Review
        if err := h.db.Where("seller_id = ? AND reviewer_id = ? AND deleted_at IS NULL", sellerID, reviewerID).
                First(&existing).Error; err == nil {
                c.JSON(http.StatusConflict, gin.H{"error": "you have already reviewed this seller"})
                return
        }

        // Get reviewer name
        var reviewer users.User
        if err := h.db.Select("name").Where("id = ?", reviewerID).First(&reviewer).Error; err != nil {
                response.InternalError(c, err)
                return
        }

        review := Review{
                SellerID:     sellerID,
                ReviewerID:   reviewerID,
                ReviewerName: reviewer.Name,
                Rating:       req.Rating,
                Comment:      req.Comment,
        }
        if err := h.db.Create(&review).Error; err != nil {
                response.InternalError(c, err)
                return
        }

        // Update seller's aggregate rating
        var avgResult struct {
                AvgRating float64
                Count     int64
        }
        h.db.Model(&Review{}).
                Select("AVG(rating) as avg_rating, COUNT(*) as count").
                Where("seller_id = ? AND deleted_at IS NULL", sellerID).
                Scan(&avgResult)

        h.db.Model(&users.User{}).Where("id = ?", sellerID).Updates(map[string]interface{}{
                "rating":       avgResult.AvgRating,
                "review_count": avgResult.Count,
        })

        c.JSON(http.StatusCreated, gin.H{"data": review, "message": "Review submitted successfully"})
}
