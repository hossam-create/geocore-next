package auth

import (
	"context"
	"strings"
	"time"

	"github.com/geocore-next/backend/internal/security"
	"github.com/geocore-next/backend/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Refresh — POST /api/v1/auth/refresh
//
// Security properties:
//   - Refresh token rotation: old token is deleted, new pair issued on every call
//   - Reuse detection: if a refresh token that was already consumed is submitted,
//     the entire user session is revoked immediately (all devices logged out)
//   - Rate limited at the route level (3 per minute per IP)
func (h *Handler) Refresh(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	ctx := context.Background()
	redisKey := refreshTokenPrefix + req.RefreshToken
	usedKey := refreshTokenUsedPrefix + req.RefreshToken

	// ── 1. Look up the refresh token in Redis ─────────────────────────────
	val, err := h.rdb.Get(ctx, redisKey).Result()
	if err != nil {
		// If this token was already consumed before, this is a reuse attempt.
		if err == redis.Nil {
			if reusedUserID, usedErr := h.rdb.Get(ctx, usedKey).Result(); usedErr == nil && reusedUserID != "" {
				h.revokeAllSessions(ctx, reusedUserID)
				if parsedID, parseErr := uuid.Parse(reusedUserID); parseErr == nil {
					security.LogEvent(h.db, c, &parsedID, security.EventSessionRevoked, map[string]any{
						"reason": "refresh_token_reuse_detected",
					})
				}
			}
		}
		response.Unauthorized(c)
		return
	}

	// ── 2. Parse stored value: "{userID}:{email}" ─────────────────────────
	parts := strings.SplitN(val, ":", 2)
	if len(parts) != 2 {
		h.rdb.Del(ctx, redisKey) //nolint:errcheck
		response.Unauthorized(c)
		return
	}
	userID, email := parts[0], parts[1]

	// ── 3. Atomically delete the consumed token (rotation) ────────────────
	deleted, err := h.rdb.Del(ctx, redisKey).Result()
	if err != nil || deleted == 0 {
		// Race condition: another request consumed it first → treat as reuse
		h.revokeAllSessions(ctx, userID)
		if parsedID, parseErr := uuid.Parse(userID); parseErr == nil {
			security.LogEvent(h.db, c, &parsedID, security.EventSessionRevoked, map[string]any{
				"reason": "refresh_token_race_or_reuse",
			})
		}
		response.Unauthorized(c)
		return
	}
	// Mark consumed refresh token so future reuse attempts can be detected.
	h.rdb.Set(ctx, usedKey, userID, refreshTokenExpiry) //nolint:errcheck

	// ── 4. Issue fresh token pair ─────────────────────────────────────────
	accessToken, err := generateAccessToken(userID, email)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	newRefreshToken, err := generateRefreshToken(ctx, h.rdb, userID, email)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.OK(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
	})
}

// Logout — POST /api/v1/auth/logout
// Revokes the provided refresh token and sets a revoke-before timestamp so
// any outstanding access tokens for this user become invalid.
func (h *Handler) Logout(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	// Ignore bind error — body is optional
	_ = c.ShouldBindJSON(&req)

	ctx := context.Background()

	if req.RefreshToken != "" {
		h.rdb.Del(ctx, refreshTokenPrefix+req.RefreshToken) //nolint:errcheck
	}

	// Revoke all access tokens for this user
	userID := c.GetString("user_id")
	if userID != "" {
		h.revokeAllSessions(ctx, userID)
		if parsedID, parseErr := uuid.Parse(userID); parseErr == nil {
			security.LogEvent(h.db, c, &parsedID, security.EventSessionRevoked, map[string]any{
				"reason": "logout",
			})
		}
	}

	response.OK(c, gin.H{"message": "Logged out successfully"})
}

// revokeAllSessions sets a revoke-before timestamp so all access tokens
// issued before now become invalid for this user.
func (h *Handler) revokeAllSessions(ctx context.Context, userID string) {
	revokeKey := "revoke-before:" + userID
	h.rdb.Set(ctx, revokeKey, time.Now().Unix(), 30*24*time.Hour) //nolint:errcheck
}
