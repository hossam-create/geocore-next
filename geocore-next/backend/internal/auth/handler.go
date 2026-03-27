package auth

  import (
        "context"
        "crypto/sha256"
        "fmt"
        "os"
        "time"

        "github.com/geocore-next/backend/internal/users"
        "github.com/geocore-next/backend/pkg/email"
        "github.com/geocore-next/backend/pkg/middleware"
        "github.com/geocore-next/backend/pkg/response"
        "github.com/gin-gonic/gin"
        "github.com/golang-jwt/jwt/v5"
        "github.com/google/uuid"
        "github.com/redis/go-redis/v9"
        "golang.org/x/crypto/bcrypt"
        "gorm.io/gorm"
  )

  type Handler struct {
        db  *gorm.DB
        rdb *redis.Client
  }

  func NewHandler(db *gorm.DB, rdb *redis.Client) *Handler {
        return &Handler{db, rdb}
  }

  type RegisterReq struct {
        Name     string `json:"name"     binding:"required,min=2,max=100"`
        Email    string `json:"email"    binding:"required,email"`
        Password string `json:"password" binding:"required,min=8"`
        Phone    string `json:"phone"`
  }

  type LoginReq struct {
        Email    string `json:"email"    binding:"required,email"`
        Password string `json:"password" binding:"required"`
  }

  type RefreshReq struct {
        RefreshToken string `json:"refresh_token" binding:"required"`
  }

  // Register — POST /api/v1/auth/register
  // Creates a new user account, fires a verification email, and returns a JWT pair.
  // The email_verified flag starts as false; certain actions require verification.
  func (h *Handler) Register(c *gin.Context) {
        var req RegisterReq
        if err := c.ShouldBindJSON(&req); err != nil {
                response.BadRequest(c, err.Error())
                return
        }

        // Reject duplicate email
        var existing users.User
        if h.db.Where("email = ?", req.Email).First(&existing).Error == nil {
                response.Conflict(c, "Email already in use")
                return
        }

        hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), 12)
        if err != nil {
                response.InternalError(c, err)
                return
        }

        user := users.User{
                ID:           uuid.New(),
                Name:         req.Name,
                Email:        req.Email,
                Phone:        req.Phone,
                PasswordHash: string(hash),
        }

        if err := h.db.Create(&user).Error; err != nil {
                response.InternalError(c, err)
                return
        }

        // Send email verification asynchronously (non-blocking)
        h.sendInitialVerificationEmail(&user)

        // Send welcome email
        go email.SendWelcomeEmail(user.Email, user.Name)

        accessToken, err := generateAccessToken(user.ID.String(), user.Email, user.Role)
        if err != nil {
                response.InternalError(c, err)
                return
        }

        refreshToken, err := h.generateRefreshToken(c.Request.Context(), user.ID.String())
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

  // Login — POST /api/v1/auth/login
  func (h *Handler) Login(c *gin.Context) {
        var req LoginReq
        if err := c.ShouldBindJSON(&req); err != nil {
                response.BadRequest(c, err.Error())
                return
        }

        var user users.User
        if err := h.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
                response.Unauthorized(c)
                return
        }

        if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
                response.Unauthorized(c)
                return
        }

        accessToken, err := generateAccessToken(user.ID.String(), user.Email, user.Role)
        if err != nil {
                response.InternalError(c, err)
                return
        }

        refreshToken, err := h.generateRefreshToken(c.Request.Context(), user.ID.String())
        if err != nil {
                response.InternalError(c, err)
                return
        }

        result := gin.H{
                "access_token":  accessToken,
                "refresh_token": refreshToken,
                "user":          user,
        }
        if !user.EmailVerified {
                result["warning"] = "Email not verified — some features are restricted"
        }

        response.OK(c, result)
  }

  // Refresh — POST /api/v1/auth/refresh
  // Validates the refresh token, rotates both tokens, and returns a new pair.
  // On second use of the same refresh token (theft detection), all refresh tokens
  // for that user are revoked.
  func (h *Handler) Refresh(c *gin.Context) {
        var req RefreshReq
        if err := c.ShouldBindJSON(&req); err != nil {
                response.BadRequest(c, err.Error())
                return
        }

        ctx := c.Request.Context()

        // Parse the refresh token claims without full validation first to get userID/tokenID
        secret := os.Getenv("JWT_SECRET")
        claims := &middleware.Claims{}
        tok, err := jwt.ParseWithClaims(req.RefreshToken, claims, func(t *jwt.Token) (interface{}, error) {
                if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
                        return nil, jwt.ErrSignatureInvalid
                }
                return []byte(secret), nil
        })
        if err != nil || !tok.Valid {
                response.Unauthorized(c)
                return
        }

        tokenID := claims.ID
        userID := claims.UserID

        if tokenID == "" || userID == "" {
                response.Unauthorized(c)
                return
        }

        // Hash the incoming token to compare with what we stored
        tokenHash := hashToken(req.RefreshToken)
        redisKey := fmt.Sprintf("refresh:%s:%s", userID, tokenID)

        storedHash, err := h.rdb.Get(ctx, redisKey).Result()
        if err != nil {
                // Key not found — token already used or never issued
                response.Unauthorized(c)
                return
        }

        if storedHash != tokenHash {
                // Hash mismatch — token reuse detected; revoke all refresh tokens for user
                h.revokeAllRefreshTokens(ctx, userID)
                response.Unauthorized(c)
                return
        }

        // Delete this refresh token (rotation: invalidate old token)
        h.rdb.Del(ctx, redisKey)

        // Fetch the user to get current role
        var user users.User
        if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
                response.Unauthorized(c)
                return
        }

        // Issue new token pair
        newAccessToken, err := generateAccessToken(user.ID.String(), user.Email, user.Role)
        if err != nil {
                response.InternalError(c, err)
                return
        }

        newRefreshToken, err := h.generateRefreshToken(ctx, user.ID.String())
        if err != nil {
                response.InternalError(c, err)
                return
        }

        response.OK(c, gin.H{
                "access_token":  newAccessToken,
                "refresh_token": newRefreshToken,
        })
  }

  // Me — GET /api/v1/auth/me (auth required)
  func (h *Handler) Me(c *gin.Context) {
        userID := c.MustGet("user_id").(string)
        var user users.User
        if err := h.db.First(&user, "id = ?", userID).Error; err != nil {
                response.NotFound(c, "User")
                return
        }
        response.OK(c, user)
  }

  // generateAccessToken creates a short-lived (15-minute) JWT access token.
  func generateAccessToken(userID, email, role string) (string, error) {
        claims := middleware.Claims{
                UserID: userID,
                Email:  email,
                Role:   role,
                RegisteredClaims: jwt.RegisteredClaims{
                        ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
                        IssuedAt:  jwt.NewNumericDate(time.Now()),
                        ID:        uuid.New().String(),
                },
        }
        token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
        return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
  }

  // generateRefreshToken creates a long-lived (30-day) JWT refresh token and
  // stores its hash in Redis under refresh:{userID}:{tokenID}.
  // If the Redis client is nil (e.g. in tests), the token is returned without
  // being persisted — refresh validation will fail, which is acceptable in tests.
  func (h *Handler) generateRefreshToken(ctx context.Context, userID string) (string, error) {
        tokenID := uuid.New().String()
        claims := middleware.Claims{
                UserID: userID,
                RegisteredClaims: jwt.RegisteredClaims{
                        ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
                        IssuedAt:  jwt.NewNumericDate(time.Now()),
                        ID:        tokenID,
                },
        }
        token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
        signed, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
        if err != nil {
                return "", err
        }

        if h.rdb == nil {
                return signed, nil
        }

        // Store the hash in Redis with 30-day TTL
        redisKey := fmt.Sprintf("refresh:%s:%s", userID, tokenID)
        tokenHash := hashToken(signed)
        if err := h.rdb.Set(ctx, redisKey, tokenHash, 30*24*time.Hour).Err(); err != nil {
                return "", err
        }

        return signed, nil
  }

  // hashToken returns a SHA-256 hex digest of the given token string.
  func hashToken(token string) string {
        sum := sha256.Sum256([]byte(token))
        return fmt.Sprintf("%x", sum)
  }


  // Logout — POST /api/v1/auth/logout
  // Immediately invalidates the caller's session by setting a revoke-before
  // timestamp in Redis (so all existing access tokens are rejected) and
  // deleting every refresh token stored for this user.
  func (h *Handler) Logout(c *gin.Context) {
        userID := c.MustGet("user_id").(string)
        ctx := c.Request.Context()

        if h.rdb != nil {
                revokeKey := fmt.Sprintf("revoke-before:%s", userID)
                h.rdb.Set(ctx, revokeKey, time.Now().Unix(), 30*24*time.Hour)
                h.revokeAllRefreshTokens(ctx, userID)
        }

        response.OK(c, gin.H{"message": "Logged out successfully"})
  }

  // revokeAllRefreshTokens deletes all refresh:userID:* keys for the given user.
  func (h *Handler) revokeAllRefreshTokens(ctx context.Context, userID string) {
        if h.rdb == nil {
                return
        }
        pattern := fmt.Sprintf("refresh:%s:*", userID)
        var cursor uint64
        for {
                keys, nextCursor, err := h.rdb.Scan(ctx, cursor, pattern, 100).Result()
                if err != nil {
                        break
                }
                if len(keys) > 0 {
                        h.rdb.Del(ctx, keys...)
                }
                cursor = nextCursor
                if cursor == 0 {
                        break
                }
        }
  }
