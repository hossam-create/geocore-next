package constants

import "time"

// Server timeouts
const (
	ServerReadTimeout  = 15 * time.Second
	ServerWriteTimeout = 30 * time.Second
	ServerIdleTimeout  = 60 * time.Second
	ShutdownTimeout    = 30 * time.Second
)

// Database connection pool
const (
	DBMaxIdleConns = 10
	DBMaxOpenConns = 100
)

// Redis timeouts
const (
	RedisConnectTimeout = 5 * time.Second
	RedisPingTimeout    = 5 * time.Second
)

// JWT configuration
const (
	JWTAccessTokenExpiry  = 15 * time.Minute
	JWTRefreshTokenExpiry = 7 * 24 * time.Hour
	JWTSecretMinLength    = 32
)

// Rate limiting
const (
	RateLimitDefault      = 100
	RateLimitAuth         = 10
	RateLimitAuthWindow   = 15 * time.Minute
	RateLimitAPIWindow    = time.Minute
	RateLimitUploadWindow = time.Hour
)

// File upload limits
const (
	MaxUploadSize     = 50 << 20 // 50 MB total
	MaxImageSize      = 5 << 20  // 5 MB per image
	MaxImagesPerUpload = 10
)

// Image processing
const (
	ImageJPEGQuality   = 85
	ImageMaxOriginalDim = 4096
	ImageMaxLargeDim    = 1200
	ImageMaxMediumDim   = 600
	ImageMaxThumbnailDim = 200
)

// Pagination
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Auction settings
const (
	AuctionMinDuration     = 1 * time.Hour
	AuctionMaxDuration     = 30 * 24 * time.Hour
	AuctionAntiSnipeWindow = 5 * time.Minute
	AuctionAntiSnipeExtend = 2 * time.Minute
)

// Escrow settings
const (
	EscrowHoldDuration    = 14 * 24 * time.Hour
	EscrowDisputeWindow   = 7 * 24 * time.Hour
)

// Wallet settings
const (
	WalletMinDeposit    = 1.0
	WalletMaxDeposit    = 100000.0
	WalletMinWithdrawal = 10.0
)

// Loyalty settings
const (
	LoyaltyDailyBonusPoints = 10
	LoyaltyReferralPoints   = 100
	LoyaltyStreakMultiplier = 1.5
)

// Background jobs
const (
	JobMaxRetries       = 3
	JobRetryBaseDelay   = time.Second
	JobDefaultPriority  = 5
	JobWorkerCount      = 4
	JobFailedQueueLimit = 1000
)

// WebSocket settings
const (
	WSReadBufferSize  = 1024
	WSWriteBufferSize = 1024
	WSPingInterval    = 30 * time.Second
	WSPongTimeout     = 60 * time.Second
)

// Cache TTLs
const (
	CacheShortTTL  = 5 * time.Minute
	CacheMediumTTL = 30 * time.Minute
	CacheLongTTL   = 24 * time.Hour
)

// Password requirements
const (
	PasswordMinLength = 10
	PasswordMaxLength = 72
)

// Bcrypt cost
const (
	BcryptCost = 12
)

// CORS settings
const (
	CORSMaxAge = 12 * time.Hour
)

// Dispute settings
const (
	DisputeResponseDeadline = 72 * time.Hour
	DisputeMaxMessages      = 100
	DisputeMaxEvidence      = 20
)
