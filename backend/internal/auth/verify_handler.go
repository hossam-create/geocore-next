package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/pkg/email"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	verifyTokenTTL = 24 * time.Hour
	resendCooldown = 5 * time.Minute
	tokenByteLen   = 32
)

// VerifyEmail — POST /api/v1/auth/verify-email
// Public endpoint. Body: {"token": "<hex-token>"}
func (h *Handler) VerifyEmail(c *gin.Context) {
	var req struct {
		Token string `json:"token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var user users.User
	if err := h.db.Where("verification_token = ?", req.Token).First(&user).Error; err != nil {
		response.BadRequest(c, "Invalid or expired verification token")
		return
	}

	if user.EmailVerified {
		response.BadRequest(c, "Email is already verified")
		return
	}

	if user.VerificationTokenExpiresAt == nil || time.Now().After(*user.VerificationTokenExpiresAt) {
		response.BadRequest(c, "Verification token has expired — please request a new one")
		return
	}

	// Mark verified and clear the one-time token
	if err := h.db.Model(&user).Updates(map[string]interface{}{
		"email_verified":                true,
		"verification_token":            "",
		"verification_token_expires_at": nil,
	}).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Issue a fresh JWT so the client gets an updated session token
	token, _ := generateAccessToken(user.ID.String(), user.Email)
	response.OK(c, gin.H{
		"message": "Email verified successfully! You can now use all features.",
		"token":   token,
	})
}

// ResendVerification — POST /api/v1/auth/resend-verification
// Auth required. Rate limited to once every 5 minutes via Redis.
func (h *Handler) ResendVerification(c *gin.Context) {
	userID := c.MustGet("user_id").(string)

	var user users.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		response.NotFound(c, "User")
		return
	}

	if user.EmailVerified {
		response.BadRequest(c, "Your email is already verified")
		return
	}

	// Rate limit: one email per resendCooldown window
	cooldownKey := fmt.Sprintf("resend_verify:%s", userID)
	ctx := context.Background()

	exists, err := h.rdb.Exists(ctx, cooldownKey).Result()
	if err == nil && exists > 0 {
		ttl, _ := h.rdb.TTL(ctx, cooldownKey).Result()
		response.BadRequest(c, fmt.Sprintf(
			"Please wait %d seconds before requesting another verification email",
			int(ttl.Seconds()),
		))
		return
	}

	// Generate a fresh token
	token, err := email.GenerateToken(tokenByteLen)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	expiresAt := time.Now().Add(verifyTokenTTL)
	if err := h.db.Model(&user).Updates(map[string]interface{}{
		"verification_token":            token,
		"verification_token_expires_at": expiresAt,
	}).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// Send email asynchronously — token is persisted regardless
	go func() {
		if sendErr := email.SendVerificationEmail(user.Email, token); sendErr != nil {
			fmt.Printf("[email] failed to send to %s: %v\n", user.Email, sendErr)
		}
	}()

	// Set Redis cooldown key so user cannot spam resend
	h.rdb.Set(ctx, cooldownKey, 1, resendCooldown)

	response.OK(c, gin.H{"message": "Verification email sent — please check your inbox"})
}

// sendInitialVerificationEmail generates a token, saves it, and dispatches
// the verification email in a goroutine. Called from Register after user creation.
func (h *Handler) sendInitialVerificationEmail(user *users.User) {
	token, err := email.GenerateToken(tokenByteLen)
	if err != nil {
		fmt.Printf("[email] GenerateToken failed for %s: %v\n", user.Email, err)
		return
	}

	expiresAt := time.Now().Add(verifyTokenTTL)
	h.db.Model(user).Updates(map[string]interface{}{
		"verification_token":            token,
		"verification_token_expires_at": expiresAt,
	})

	go func() {
		if sendErr := email.SendVerificationEmail(user.Email, token); sendErr != nil {
			fmt.Printf("[email] failed to send initial email to %s: %v\n", user.Email, sendErr)
		}
	}()
}
