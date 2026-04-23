package auth

import (
	"time"

	"github.com/geocore-next/backend/internal/security"
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ════════════════════════════════════════════════════════════════════════════
// 2FA HTTP Handlers
// ════════════════════════════════════════════════════════════════════════════

// POST /api/v1/auth/2fa/enable — initiate 2FA setup
func (h *Handler) Enable2FA(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	userUUID, _ := uuid.Parse(userID)
	userEmail := c.GetString("user_email")

	var req Enable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	svc := NewTwoFAService(h.db)
	result, err := svc.Enable2FA(userUUID, userEmail, req.Password, "")
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	security.LogEvent(h.db, c, &userUUID, "2fa_setup_initiated", map[string]any{
		"user_id": userID,
	})

	response.OK(c, gin.H{
		"secret":       result.Secret,
		"qr_code_uri":  result.QRCodeURI,
		"backup_codes": result.BackupCodes,
		"message":      "Scan the QR code with your authenticator app, then verify with /auth/2fa/confirm",
	})
}

// POST /api/v1/auth/2fa/confirm — confirm 2FA setup with first TOTP code
func (h *Handler) ConfirmEnable2FA(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	userUUID, _ := uuid.Parse(userID)

	var req Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	svc := NewTwoFAService(h.db)
	if err := svc.ConfirmEnable2FA(userUUID, req.Code); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Revoke all existing sessions — user must re-login with 2FA
	if h.rdb != nil {
		RevokeAllSessions(c, h.rdb, userID)
	}

	security.LogEvent(h.db, c, &userUUID, "2fa_enabled", map[string]any{
		"user_id": userID,
	})

	response.OK(c, gin.H{
		"message": "2FA enabled successfully. Please log in again with your authenticator.",
	})
}

// POST /api/v1/auth/2fa/verify — verify TOTP/backup code after login
// This endpoint accepts a 2FA challenge token (from login response) + TOTP code.
func (h *Handler) Verify2FA(c *gin.Context) {
	var req struct {
		ChallengeToken string `json:"challenge_token" binding:"required"`
		Code           string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Validate the challenge token
	userID, err := Validate2FAChallengeToken(req.ChallengeToken)
	if err != nil {
		response.Unauthorized(c)
		return
	}

	userUUID, _ := uuid.Parse(userID)

	svc := NewTwoFAService(h.db)
	valid, usedBackup, err := svc.Verify2FACode(userUUID, req.Code)
	if err != nil || !valid {
		security.LogEvent(h.db, c, &userUUID, "2fa_failed", map[string]any{
			"user_id": userID,
		})
		response.BadRequest(c, "Invalid 2FA code")
		return
	}

	// 2FA passed — issue full access + refresh tokens
	var user struct {
		ID    uuid.UUID `gorm:"type:uuid"`
		Email string
	}
	if err := h.db.Table("users").Select("id, email").Where("id = ?", userID).First(&user).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	accessToken, err := generateAccessToken(userID, user.Email)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	refreshToken, err := generateRefreshToken(c.Request.Context(), h.rdb, userID, user.Email)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	security.LogEvent(h.db, c, &userUUID, "2fa_verified", map[string]any{
		"user_id":     userID,
		"used_backup": usedBackup,
	})

	response.OK(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"used_backup":   usedBackup,
	})
}

// POST /api/v1/auth/2fa/disable — disable 2FA
func (h *Handler) Disable2FA(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	userUUID, _ := uuid.Parse(userID)

	var req Disable2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	svc := NewTwoFAService(h.db)
	if err := svc.Disable2FA(userUUID, req.Password, req.Code); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	security.LogEvent(h.db, c, &userUUID, "2fa_disabled", map[string]any{
		"user_id": userID,
	})

	response.OK(c, gin.H{"message": "2FA disabled successfully."})
}

// GET /api/v1/auth/2fa/status — get 2FA status
func (h *Handler) Get2FAStatus(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	userUUID, _ := uuid.Parse(userID)

	svc := NewTwoFAService(h.db)
	enabled, verified, _ := svc.Get2FAStatus(userUUID)

	response.OK(c, gin.H{
		"enabled":  enabled,
		"verified": verified,
	})
}

// POST /api/v1/auth/2fa/backup-codes — regenerate backup codes
func (h *Handler) RegenerateBackupCodes(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	userUUID, _ := uuid.Parse(userID)

	var req Verify2FARequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	svc := NewTwoFAService(h.db)
	codes, err := svc.RegenerateBackupCodes(userUUID, req.Code)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	security.LogEvent(h.db, c, &userUUID, "2fa_backup_regenerated", map[string]any{
		"user_id": userID,
	})

	response.OK(c, gin.H{
		"backup_codes": codes,
		"message":      "Store these codes securely. Previous codes are no longer valid.",
	})
}

// RevokeAllSessions invalidates all active sessions for a user.
func RevokeAllSessions(c *gin.Context, rdb *redis.Client, userID string) {
	if middleware.RevocationRDB != nil {
		now := time.Now().Unix()
		middleware.RevocationRDB.Set(c.Request.Context(), "revoke-before:"+userID, now, 0)
	}
}
