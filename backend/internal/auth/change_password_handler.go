package auth

import (
	"time"

	"github.com/geocore-next/backend/internal/security"
	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/pkg/response"
	pkgvalidator "github.com/geocore-next/backend/pkg/validator"
	"github.com/gin-gonic/gin"
)

type ChangePasswordReq struct {
	OldPassword     string `json:"old_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

// ChangePassword — POST /api/v1/auth/change-password
// Auth required.
//
// Security properties:
//   - Verifies the old password
//   - Enforces password complexity
//   - Updates password_hash + password_changed_at
//   - Revokes all existing access tokens via revoke-before timestamp
//   - Writes an audit event
func (h *Handler) ChangePassword(c *gin.Context) {
	userID := c.GetString("user_id")
	if userID == "" {
		response.Unauthorized(c)
		return
	}

	var req ChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if req.NewPassword != req.ConfirmPassword {
		response.BadRequest(c, "Passwords do not match")
		return
	}

	if !pkgvalidator.PasswordStrength(req.NewPassword) {
		response.BadRequest(c, "Password must be at least 10 characters with uppercase, lowercase, digit, and special character")
		return
	}

	var user users.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		response.NotFound(c, "User")
		return
	}

	if !security.VerifyPassword(user.PasswordHash, req.OldPassword) {
		security.LogEvent(h.db, c, &user.ID, security.EventPasswordChange, map[string]any{
			"result": "old_password_invalid",
		})
		response.Unauthorized(c)
		return
	}

	hash, err := security.HashPassword(req.NewPassword)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	now := time.Now()
	if err := h.db.Model(&user).Updates(map[string]any{
		"password_hash":       hash,
		"password_changed_at": now,
	}).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Revoke all access tokens for this user
	h.revokeAllSessions(c.Request.Context(), user.ID.String())

	security.LogEvent(h.db, c, &user.ID, security.EventPasswordChange, map[string]any{
		"result": "success",
	})

	response.OK(c, gin.H{"message": "Password changed successfully"})
}
