package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/geocore-next/backend/internal/config"
	"github.com/geocore-next/backend/internal/invite"
	"github.com/geocore-next/backend/internal/notifications"
	"github.com/geocore-next/backend/internal/referral"
	"github.com/geocore-next/backend/internal/security"
	"github.com/geocore-next/backend/internal/users"
	"github.com/geocore-next/backend/pkg/email"
	"github.com/geocore-next/backend/pkg/events"
	"github.com/geocore-next/backend/pkg/jwtkeys"
	"github.com/geocore-next/backend/pkg/kafka"
	"github.com/geocore-next/backend/pkg/middleware"
	"github.com/geocore-next/backend/pkg/response"
	pkgvalidator "github.com/geocore-next/backend/pkg/validator"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Handler handles authentication-related HTTP requests.
// It provides endpoints for user registration, login, and user info retrieval.
type Handler struct {
	db               *gorm.DB
	rdb              *redis.Client
	notificationsSvc *notifications.Service
}

// NewHandler creates a new authentication handler with the given database and Redis client.
func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
	return &Handler{
		db:               db,
		rdb:              rdb,
		notificationsSvc: notificationService,
	}
}

// RegisterReq defines the request payload for user registration.
type RegisterReq struct {
	Name       string `json:"name"        binding:"required,min=2,max=100"`
	Email      string `json:"email"       binding:"required,email"`
	Password   string `json:"password"    binding:"required,min=10,max=72"`
	Phone      string `json:"phone"`
	InviteCode string `json:"invite_code"` // required when ENABLE_INVITE_ONLY=true
}

// LoginReq defines the request payload for user login.
type LoginReq struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Register — POST /api/v1/auth/register
// Creates a new user account, fires a verification email, and returns a JWT.
// The email_verified flag starts as false; certain actions require verification.
func (h *Handler) Register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// ── Invite-only gate (Part 2) ────────────────────────────────────────────
	if config.GetFlags().EnableInviteOnly {
		code := req.InviteCode
		if code == "" {
			code = c.Query("invite")
		}
		if code == "" {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Invite required",
				"message": "This platform is currently in private access mode.",
			})
			return
		}
		if _, ivErr := invite.ValidateInviteCode(h.db, code); ivErr != nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   ivErr.Error(),
				"message": "Your invite code is invalid, expired, or exhausted.",
			})
			return
		}
		// Anti-abuse: rapid signup detection
		if invite.CheckRapidSignup(c.Request.Context(), h.rdb, c.ClientIP()) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too many signups from this location. Please try again later.",
				"message": "Suspicious activity detected.",
			})
			return
		}
	}

	// Validate password strength
	if !pkgvalidator.PasswordStrength(req.Password) {
		response.BadRequest(c, "Password must be at least 10 characters with uppercase, lowercase, digit, and special character")
		return
	}

	// Reject duplicate email
	var existing users.User
	if h.db.Where("email = ?", req.Email).First(&existing).Error == nil {
		response.Conflict(c, "Email already in use")
		return
	}

	hash, err := security.HashPassword(req.Password)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	newID := uuid.New()
	user := users.User{
		ID:           newID,
		Name:         req.Name,
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: hash,
		ReferralCode: referral.GenerateCode(newID),
	}

	if err := h.db.Create(&user).Error; err != nil {
		response.InternalError(c, err)
		return
	}
	security.LogEvent(h.db, c, &user.ID, security.EventAccountCreated, map[string]any{
		"email": security.MaskEmail(user.Email),
	})

	// Publish domain event for in-process consumers
	events.Publish(events.Event{
		Type: events.EventUserRegistered,
		Payload: map[string]interface{}{
			"user_id": user.ID.String(),
			"email":   user.Email,
			"name":    user.Name,
		},
	})

	// Transactional outbox for Kafka delivery
	_ = kafka.WriteOutbox(h.db, kafka.TopicUsers, kafka.New(
		"user.created",
		user.ID.String(),
		"user",
		kafka.Actor{Type: "user", ID: user.ID.String()},
		map[string]interface{}{
			"user_id": user.ID.String(),
			"email":   user.Email,
			"name":    user.Name,
		},
		kafka.EventMeta{Source: "api-service"},
	))

	// Link referral if a code was provided in the query string
	if refCode := c.Query("ref"); refCode != "" {
		go referral.LinkReferral(h.db, user.ID, refCode)
	}

	// Consume invite code and wire pending reward (Part 2/4)
	if config.GetFlags().EnableInviteOnly {
		code := req.InviteCode
		if code == "" {
			code = c.Query("invite")
		}
		if code != "" {
			go func() {
				_ = invite.UseInvite(h.db, code, user.ID)
				if inviterID, ok := invite.GetInviterForUser(h.db, user.ID); ok {
					_ = invite.CreatePendingReward(h.db, inviterID, user.ID)
				}
			}()
		}
	}

	// Send email verification asynchronously (non-blocking)
	h.sendInitialVerificationEmail(&user)

	// Send welcome email — already async via SendAsync pipeline
	_ = email.SendWelcomeEmail(user.Email, user.Name)

	accessToken, err := generateAccessToken(user.ID.String(), user.Email)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	refreshToken, err := generateRefreshToken(c.Request.Context(), h.rdb, user.ID.String(), user.Email)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.Created(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          user,
		"message":       "Registration successful! Please check your email to verify your account.",
	})
}

// Login authenticates a user with email and password.
// POST /api/v1/auth/login
// Returns a JWT token on successful authentication.
func (h *Handler) Login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	var user users.User
	if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		security.LogEvent(h.db, c, nil, security.EventLoginFailed, map[string]any{
			"email": security.MaskEmail(req.Email),
		})
		response.Unauthorized(c)
		return
	}

	if !security.VerifyPassword(user.PasswordHash, req.Password) {
		security.LogEvent(h.db, c, &user.ID, security.EventLoginFailed, map[string]any{
			"email": security.MaskEmail(user.Email),
		})
		response.Unauthorized(c)
		return
	}
	security.LogEvent(h.db, c, &user.ID, security.EventLoginSuccess, map[string]any{
		"email": security.MaskEmail(user.Email),
	})

	// ── 2FA check ────────────────────────────────────────────────────────────
	svc := NewTwoFAService(h.db)
	if svc.Is2FAEnabled(user.ID) {
		challengeToken, err := Generate2FAChallengeToken(user.ID.String())
		if err != nil {
			response.InternalError(c, err)
			return
		}
		response.OK(c, gin.H{
			"requires_2fa":    true,
			"challenge_token": challengeToken,
			"expires_in":      int(twoFATokenExpiry.Seconds()),
			"message":         "2FA verification required. POST /auth/2fa/verify with your code.",
		})
		return
	}

	accessToken, err := generateAccessToken(user.ID.String(), user.Email)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	refreshToken, err := generateRefreshToken(c.Request.Context(), h.rdb, user.ID.String(), user.Email)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	result := gin.H{"access_token": accessToken, "refresh_token": refreshToken, "user": user}
	if !user.EmailVerified {
		result["warning"] = "Email not verified — some features are restricted"
	}

	response.OK(c, result)
}

// Me retrieves the current authenticated user's profile.
// GET /api/v1/auth/me (auth required)
func (h *Handler) Me(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	var user users.User
	if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
		response.NotFound(c, "User")
		return
	}
	response.OK(c, user)
}

const (
	accessTokenExpiry      = 15 * time.Minute
	refreshTokenExpiry     = 7 * 24 * time.Hour
	refreshTokenPrefix     = "refresh:"
	refreshTokenUsedPrefix = "refresh:used:"
)

// generateAccessToken issues a short-lived RS256 JWT (15 minutes).
func generateAccessToken(userID, email string) (string, error) {
	claims := middleware.Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(accessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.NewString(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(jwtkeys.Private())
}

// generateRefreshToken creates a cryptographically secure random token,
// stores it in Redis (7-day TTL), and returns the raw token string.
// Key: refresh:{token}  Value: {userID}:{email}
func generateRefreshToken(ctx context.Context, rdb *redis.Client, userID, email string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	val := userID + ":" + email
	if err := rdb.Set(ctx, refreshTokenPrefix+token, val, refreshTokenExpiry).Err(); err != nil {
		return "", err
	}
	return token, nil
}
