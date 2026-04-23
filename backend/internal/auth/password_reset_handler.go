package auth

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/security"
	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/pkg/email"
	"github.com/geocore-next/backend/pkg/response"
	pkgvalidator "github.com/geocore-next/backend/pkg/validator"
	"github.com/gin-gonic/gin"
)

// ════════════════════════════════════════════════════════════════════════════
// Request types
// ════════════════════════════════════════════════════════════════════════════

type ForgotPasswordReq struct {
	Email string `json:"email" binding:"required,email"`
}

type ResetPasswordReq struct {
	Token           string `json:"token"            binding:"required"`
	NewPassword     string `json:"new_password"     binding:"required"`
	ConfirmPassword string `json:"confirm_password" binding:"required"`
}

type ValidateResetTokenReq struct {
	Token string `json:"token" binding:"required"`
}

// ════════════════════════════════════════════════════════════════════════════
// Constants
// ════════════════════════════════════════════════════════════════════════════

const (
	pwResetRatePrefix = "pw-reset-rate:" // Redis key: rate-limit per email
	pwResetRateTTL    = time.Hour        // 3 requests per hour per email
	pwResetRateMax    = 3

	pwResetTokenBytes = 32        // 32 bytes → 64-char hex token
	pwResetTokenTTL   = time.Hour // token valid for 1 hour

	pwRevokePrefix = "revoke-before:"    // Redis key: revocation timestamp per user
	pwRevokeTTL    = 30 * 24 * time.Hour // keep record for JWT lifetime (30 days)
)

// ════════════════════════════════════════════════════════════════════════════
// ForgotPassword — POST /api/v1/auth/forgot-password
// ════════════════════════════════════════════════════════════════════════════

// ForgotPassword generates a secure reset token and emails it to the user.
//
// Security properties:
//   - Rate limited: max 3 requests per hour per email address (Redis INCR)
//   - Email enumeration safe: returns identical 200 response regardless of
//     whether the address is registered
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// Constant response to prevent email enumeration attacks
	const blindOK = "If that email address is registered, you will receive a password reset link shortly."

	ctx := c.Request.Context()

	// ── Rate limiting ──────────────────────────────────────────────────────────
	rateKey := pwResetRatePrefix + strings.ToLower(req.Email)
	count, err := h.rdb.Incr(ctx, rateKey).Result()
	if err != nil {
		response.InternalError(c, err)
		return
	}
	if count == 1 {
		h.rdb.Expire(ctx, rateKey, pwResetRateTTL) //nolint:errcheck
	}
	if count > pwResetRateMax {
		ttl, _ := h.rdb.TTL(ctx, rateKey).Result()
		waitSec := int(ttl.Seconds()) + 1
		response.RateLimited(c, fmt.Sprintf("Too many requests. Please wait %d seconds before trying again.", waitSec))
		return
	}

	// ── Look up user ───────────────────────────────────────────────────────────
	var user users.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		// Email not found — return blind response (enumeration-safe).
		// Keep the rate-limit counter to prevent abuse against unknown emails.
		response.OK(c, gin.H{"message": blindOK})
		return
	}

	// ── Generate cryptographically secure token ────────────────────────────────
	token, err := email.GenerateToken(pwResetTokenBytes)
	if err != nil {
		h.rdb.Del(ctx, rateKey) //nolint:errcheck
		response.InternalError(c, err)
		return
	}
	expiresAt := time.Now().Add(pwResetTokenTTL)

	if err := h.db.Model(&user).Updates(map[string]any{
		"password_reset_token":      token,
		"password_reset_expires_at": expiresAt,
	}).Error; err != nil {
		h.rdb.Del(ctx, rateKey) //nolint:errcheck
		response.InternalError(c, err)
		return
	}
	security.LogEvent(h.db, c, &user.ID, security.EventPasswordResetReq, map[string]any{
		"email": security.MaskEmail(user.Email),
	})

	// ── Send reset email — already async via SendAsync pipeline ────────────────
	if err := email.SendPasswordResetEmail(user.Email, user.Name, user.ID.String(), token); err != nil {
		slog.Error("failed to send password reset email",
			"user_id", user.ID.String(),
			"error", err.Error(),
		)
	}

	response.OK(c, gin.H{"message": blindOK})
}

// ════════════════════════════════════════════════════════════════════════════
// ValidateResetToken — POST /api/v1/auth/validate-reset-token
// ════════════════════════════════════════════════════════════════════════════

// ValidateResetToken checks whether a reset token is still valid.
// The frontend calls this before rendering the reset form to avoid showing
// a form that will fail on submission.
func (h *Handler) ValidateResetToken(c *gin.Context) {
	var req ValidateResetTokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, tokenErr := h.findUserByResetToken(req.Token)
	if tokenErr != nil {
		response.BadRequest(c, tokenErr.Error())
		return
	}

	response.OK(c, gin.H{
		"valid":      true,
		"email":      maskEmail(user.Email),
		"expires_at": user.PasswordResetExpiresAt,
	})
}

// ════════════════════════════════════════════════════════════════════════════
// ResetPassword — POST /api/v1/auth/reset-password
// ════════════════════════════════════════════════════════════════════════════

// ResetPassword validates the token, sets the new password, and revokes all
// JWTs issued before the reset.
//
// Security properties:
//   - Token consumed on use (cleared from DB)
//   - All previous JWTs revoked via Redis revoke-before key
//   - Security confirmation email sent to alert the account owner
//   - Password change logged for audit purposes
func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// ── Password match check ───────────────────────────────────────────────────
	if req.NewPassword != req.ConfirmPassword {
		response.BadRequest(c, "Passwords do not match")
		return
	}

	// ── Password strength validation ───────────────────────────────────────────
	if err := validatePasswordStrength(req.NewPassword); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// ── Validate reset token ───────────────────────────────────────────────────
	user, tokenErr := h.findUserByResetToken(req.Token)
	if tokenErr != nil {
		response.BadRequest(c, tokenErr.Error())
		return
	}

	// ── Hash new password (Argon2id) ───────────────────────────────────────────
	hash, err := security.HashPassword(req.NewPassword)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	// ── Persist all changes atomically ────────────────────────────────────────
	now := time.Now()
	if err := h.db.Model(user).Updates(map[string]any{
		"password_hash":             hash,
		"password_reset_token":      "", // consume and invalidate the token
		"password_reset_expires_at": nil,
		"password_changed_at":       now,
	}).Error; err != nil {
		response.InternalError(c, err)
		return
	}

	// ── Revoke all JWTs issued before this moment ──────────────────────────────
	// The Auth() middleware reads this key and rejects stale tokens.
	ctx := c.Request.Context()
	revokeKey := pwRevokePrefix + user.ID.String()
	if err := h.rdb.Set(ctx, revokeKey, now.Unix(), pwRevokeTTL).Err(); err != nil {
		// Non-fatal: log and continue
		slog.Warn("could not store JWT revocation record",
			"user_id", user.ID.String(),
			"error", err.Error(),
		)
	}

	// ── Audit log ──────────────────────────────────────────────────────────────
	security.LogEvent(h.db, c, &user.ID, security.EventPasswordResetDone, map[string]any{
		"email": security.MaskEmail(user.Email),
	})
	slog.Info("password reset completed",
		"user_id", user.ID.String(),
		"email", user.Email,
		"ip", c.ClientIP(),
		"user_agent", c.Request.UserAgent(),
	)

	// ── Security confirmation email (non-blocking via SendAsync) ────────────────
	if err := email.SendPasswordChangedEmail(user.Email, user.Name, user.ID.String()); err != nil {
		slog.Error("failed to send password changed confirmation",
			"user_id", user.ID.String(),
			"error", err.Error(),
		)
	}

	response.OK(c, gin.H{
		"message": "Password reset successful. You can now sign in with your new password.",
	})
}

// ════════════════════════════════════════════════════════════════════════════
// Internal helpers
// ════════════════════════════════════════════════════════════════════════════

// findUserByResetToken retrieves and validates a password-reset token.
// Returns the user if the token is valid and not expired, or an error.
func (h *Handler) findUserByResetToken(token string) (*users.User, error) {
	if token == "" {
		return nil, errors.New("reset token is required")
	}
	var user users.User
	if err := h.db.Where("password_reset_token = ?", token).First(&user).Error; err != nil {
		return nil, errors.New("invalid or expired reset token")
	}
	if user.PasswordResetExpiresAt == nil || time.Now().After(*user.PasswordResetExpiresAt) {
		// Proactively clear the expired token to keep the DB tidy
		h.db.Model(&user).Updates(map[string]any{ //nolint:errcheck
			"password_reset_token":      "",
			"password_reset_expires_at": nil,
		})
		return nil, errors.New("reset token has expired — please request a new one")
	}
	return &user, nil
}

// validatePasswordStrength enforces password complexity rules:
//   - Minimum 8 characters
//   - At least one uppercase letter (A-Z)
//   - At least one lowercase letter (a-z)
//   - At least one digit (0-9)
func validatePasswordStrength(p string) error {
	if !pkgvalidator.PasswordStrength(p) {
		return errors.New("password must be at least 10 characters with uppercase, lowercase, digit, and special character")
	}
	return nil
}

// maskEmail partially obscures an email for security-safe display in API responses.
// Example: "ahmed.ali@example.com" → "ah***@example.com"
func maskEmail(e string) string {
	for i, r := range e {
		if r == '@' {
			if i <= 2 {
				return "***@" + e[i+1:]
			}
			return e[:2] + "***@" + e[i+1:]
		}
	}
	return "***"
}
