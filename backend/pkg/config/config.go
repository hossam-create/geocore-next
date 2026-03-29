package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration loaded from environment variables.
// This is the single source of truth for all configuration values.
type Config struct {
	// Server
	AppEnv      string
	Port        string
	FrontendURL string
	AppBaseURL  string

	// Database
	DatabaseURL string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string

	// Redis
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisURL      string

	// JWT
	JWTSecret        string
	JWTRefreshSecret string
	JWTExpiry        time.Duration
	JWTRefreshExpiry time.Duration

	// SMTP
	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	// Stripe
	StripeSecretKey    string
	StripeWebhookSecret string

	// Twilio
	TwilioAccountSID string
	TwilioAuthToken  string
	TwilioFromNumber string

	// Cloudflare R2 / S3
	R2Endpoint    string
	R2Bucket      string
	R2AccessKeyID string
	R2SecretKey   string
	R2PublicURL   string

	// Meilisearch
	MeiliHost      string
	MeiliMasterKey string

	// PostHog
	PostHogAPIKey string
	PostHogHost   string

	// WebSocket
	AllowedOrigin string

	// Rate Limiting
	RateLimitWhitelist string
}

// Load reads all environment variables and returns a Config struct.
// It should be called once at application startup.
func Load() (*Config, error) {
	cfg := &Config{
		// Server
		AppEnv:      getenv("APP_ENV", "development"),
		Port:        getenv("PORT", "8080"),
		FrontendURL: getenv("FRONTEND_URL", "http://localhost:3000"),
		AppBaseURL:  getenv("APP_BASE_URL", "http://localhost:3000"),

		// Database
		DatabaseURL: os.Getenv("DATABASE_URL"),
		DBHost:      getenv2("DB_HOST", "PGHOST", "localhost"),
		DBPort:      getenv2("DB_PORT", "PGPORT", "5432"),
		DBUser:      getenv2("DB_USER", "PGUSER", "geocore"),
		DBPassword:  getenv2("DB_PASSWORD", "PGPASSWORD", "geocore_secret"),
		DBName:      getenv2("DB_NAME", "PGDATABASE", "geocore_dev"),
		DBSSLMode:   getenv("DB_SSLMODE", "disable"),

		// Redis
		RedisHost:     getenv("REDIS_HOST", "localhost"),
		RedisPort:     getenv("REDIS_PORT", "6379"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisURL:      os.Getenv("REDIS_URL"),

		// JWT
		JWTSecret:        os.Getenv("JWT_SECRET"),
		JWTRefreshSecret: os.Getenv("JWT_REFRESH_SECRET"),
		JWTExpiry:        getDuration("JWT_EXPIRY", 15*time.Minute),
		JWTRefreshExpiry: getDuration("JWT_REFRESH_EXPIRY", 7*24*time.Hour),

		// SMTP
		SMTPHost: os.Getenv("SMTP_HOST"),
		SMTPPort: getenv("SMTP_PORT", "587"),
		SMTPUser: os.Getenv("SMTP_USER"),
		SMTPPass: os.Getenv("SMTP_PASS"),
		SMTPFrom: getenv("SMTP_FROM", "noreply@geocore.app"),

		// Stripe
		StripeSecretKey:    os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),

		// Twilio
		TwilioAccountSID: os.Getenv("TWILIO_ACCOUNT_SID"),
		TwilioAuthToken:  os.Getenv("TWILIO_AUTH_TOKEN"),
		TwilioFromNumber: os.Getenv("TWILIO_FROM_NUMBER"),

		// Cloudflare R2 / S3
		R2Endpoint:    os.Getenv("R2_ENDPOINT"),
		R2Bucket:      os.Getenv("R2_BUCKET"),
		R2AccessKeyID: os.Getenv("R2_ACCESS_KEY_ID"),
		R2SecretKey:   os.Getenv("R2_SECRET_ACCESS_KEY"),
		R2PublicURL:   os.Getenv("R2_PUBLIC_URL"),

		// Meilisearch
		MeiliHost:      getenv("MEILI_HOST", "http://localhost:7700"),
		MeiliMasterKey: os.Getenv("MEILI_MASTER_KEY"),

		// PostHog
		PostHogAPIKey: os.Getenv("POSTHOG_API_KEY"),
		PostHogHost:   getenv("POSTHOG_HOST", "https://app.posthog.com"),

		// WebSocket
		AllowedOrigin: os.Getenv("ALLOWED_ORIGIN"),

		// Rate Limiting
		RateLimitWhitelist: os.Getenv("RATE_LIMIT_WHITELIST"),
	}

	// Validate required fields
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// validate checks that required configuration values are set.
func (c *Config) validate() error {
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET environment variable is required")
	}
	if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters for security")
	}
	if c.AppEnv == "production" {
		if c.JWTSecret == "change_this_to_a_secure_random_string_min_32_chars" {
			return fmt.Errorf("JWT_SECRET must not use placeholder value in production")
		}
	}
	return nil
}

// IsProduction returns true if running in production mode.
func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "development"
}

// DatabaseDSN returns the PostgreSQL connection string.
func (c *Config) DatabaseDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		c.DBHost, c.DBUser, c.DBPassword, c.DBName, c.DBPort, c.DBSSLMode,
	)
}

// RedisAddr returns the Redis address in host:port format.
func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.RedisHost, c.RedisPort)
}

// getenv returns the value of an environment variable or a default value.
func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getenv2 tries two environment variable names before falling back to default.
func getenv2(key1, key2, fallback string) string {
	if v := os.Getenv(key1); v != "" {
		return v
	}
	if v := os.Getenv(key2); v != "" {
		return v
	}
	return fallback
}

// getDuration parses a duration from an environment variable.
func getDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

// getInt parses an integer from an environment variable.
func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return i
}
