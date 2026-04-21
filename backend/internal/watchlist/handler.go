package watchlist

import (
	"math"
	"strconv"

	"github.com/geocore-next/backend/internal/listings"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Handler struct {
	db *gorm.DB
}

func NewHandler(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

func (h *Handler) Add(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Unauthorized(c)
		return
	}

	listingID, err := uuid.Parse(c.Param("listing_id"))
	if err != nil {
		response.BadRequest(c, "Invalid listing_id")
		return
	}

	var exists int64
	if err := h.db.Model(&listings.Listing{}).
		Where("id = ?", listingID).
		Count(&exists).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	if exists == 0 {
		response.NotFound(c, "Listing")
		return
	}

	item := WatchlistItem{UserID: userID, ListingID: listingID}
	if err := h.db.Create(&item).Error; err != nil {
		// idempotent: duplicate key means already watched
		response.OK(c, gin.H{"watched": true})
		return
	}

	response.OK(c, gin.H{"watched": true})
}

func (h *Handler) Remove(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Unauthorized(c)
		return
	}

	listingID, err := uuid.Parse(c.Param("listing_id"))
	if err != nil {
		response.BadRequest(c, "Invalid listing_id")
		return
	}

	if err := h.db.Where("user_id = ? AND listing_id = ?", userID, listingID).
		Delete(&WatchlistItem{}).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, gin.H{"watched": false})
}

func (h *Handler) List(c *gin.Context) {
	userID, err := uuid.Parse(c.GetString("user_id"))
	if err != nil {
		response.Unauthorized(c)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 1
	}
	if perPage > 50 {
		perPage = 50
	}
	offset := (page - 1) * perPage

	q := h.db.Model(&listings.Listing{}).
		Joins("JOIN watchlist_items wi ON wi.listing_id = listings.id").
		Where("wi.user_id = ?", userID).
		Preload("Images").
		Preload("Category").
		Preload("Seller").
		Order("wi.created_at DESC")

	var total int64
	if err := q.Count(&total).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	var items []listings.Listing
	if err := q.Offset(offset).Limit(perPage).Find(&items).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	response.OKMeta(c, items, response.Meta{
		Total:   total,
		Page:    page,
		PerPage: perPage,
		Pages:   int64(math.Ceil(float64(total) / float64(perPage))),
	})
}
