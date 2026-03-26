package auth

  import (
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

  // Register — POST /api/v1/auth/register
  // Creates a new user account, fires a verification email, and returns a JWT.
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

        token, err := generateToken(user.ID.String(), user.Email)
        if err != nil {
                response.InternalError(c, err)
                return
        }

        response.Created(c, gin.H{
                "token":   token,
                "user":    user,
                "message": "Registration successful! Please check your email to verify your account.",
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

        token, err := generateToken(user.ID.String(), user.Email)
        if err != nil {
                response.InternalError(c, err)
                return
        }

        // Optionally warn the client if email is not yet verified
        result := gin.H{"token": token, "user": user}
        if !user.EmailVerified {
                result["warning"] = "Email not verified — some features are restricted"
        }

        response.OK(c, result)
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

  func generateToken(userID, email string) (string, error) {
        claims := middleware.Claims{
                UserID: userID,
                Email:  email,
                RegisteredClaims: jwt.RegisteredClaims{
                        ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
                        IssuedAt:  jwt.NewNumericDate(time.Now()),
                },
        }
        token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
        return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
  }
  