package users

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/geocore-next/backend/internal/security"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GDPRHandler handles GDPR-related requests (data export, account deletion).
type GDPRHandler struct {
	db *gorm.DB
}

// NewGDPRHandler creates a new GDPR handler.
func NewGDPRHandler(db *gorm.DB) *GDPRHandler {
	return &GDPRHandler{db: db}
}

// ExportData — GET /api/v1/user/data-export
// Returns all user data as a downloadable JSON file (Right to Data Portability).
func (h *GDPRHandler) ExportData(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	var user User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		response.NotFound(c, "user")
		return
	}

	// Collect user data (never include password hash or payment secrets)
	type ExportedListing struct {
		ID        uuid.UUID `json:"id"`
		Title     string    `json:"title"`
		Price     float64   `json:"price"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
	}

	type ExportedOrder struct {
		ID        uuid.UUID `json:"id"`
		Total     float64   `json:"total"`
		Status    string    `json:"status"`
		CreatedAt time.Time `json:"created_at"`
	}

	var listings []ExportedListing
	h.db.Table("listings").
		Select("id, title, price, status, created_at").
		Where("seller_id = ?", userID).
		Scan(&listings)

	var orders []ExportedOrder
	h.db.Table("orders").
		Select("id, total, status, created_at").
		Where("buyer_id = ?", userID).
		Scan(&orders)

	export := map[string]interface{}{
		"exported_at": time.Now().UTC().Format(time.RFC3339),
		"account": map[string]interface{}{
			"id":             user.ID,
			"name":           user.Name,
			"email":          user.Email,
			"phone":          user.Phone,
			"email_verified": user.EmailVerified,
			"created_at":     user.CreatedAt,
		},
		"listings": listings,
		"orders":   orders,
	}

	security.LogEvent(h.db, c, &userID, security.EventAccountCreated, map[string]any{
		"action": "data_export",
		"email":  security.MaskEmail(user.Email),
	})

	data, _ := json.MarshalIndent(export, "", "  ")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="mnbarh-data-%s.json"`, userID.String()[:8]))
	c.Data(200, "application/json", data)
}

// DeleteAccount — DELETE /api/v1/user/delete-account
// Anonymizes the user and soft-deletes their content (Right to be Forgotten).
// Transaction records are preserved for financial compliance (7 years).
func (h *GDPRHandler) DeleteAccount(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	var user User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		response.NotFound(c, "user")
		return
	}

	err := h.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// Anonymize user — keep the row for order/transaction FK integrity
		if err := tx.Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
			"email":      fmt.Sprintf("deleted-%s@mnbarh.deleted", userID.String()[:8]),
			"name":       "Deleted User",
			"phone":      "",
			"avatar_url": "",
			"deleted_at": now,
		}).Error; err != nil {
			return err
		}

		// Soft-delete all listings
		tx.Exec("UPDATE listings SET status = 'deleted', deleted_at = ? WHERE seller_id = ? AND deleted_at IS NULL", now, userID)

		return nil
	})

	if err != nil {
		response.InternalError(c, err)
		return
	}

	security.LogEvent(h.db, c, &userID, security.EventAccountDeleted, map[string]any{
		"email": security.MaskEmail(user.Email),
	})

	response.OK(c, gin.H{"message": "Account deleted. Transaction records retained for compliance."})
}
