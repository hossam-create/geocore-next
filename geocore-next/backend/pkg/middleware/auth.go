package middleware

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// RevocationRDB — set this in main.go so Auth() can check token revocation.
var RevocationRDB *redis.Client

// ValidateToken parses and validates a JWT token string.
// Returns the user ID on success, or an error.
// Used by WebSocket handlers that receive the token as a query parameter.
func ValidateToken(tokenStr string) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !tok.Valid {
		return "", fmt.Errorf("invalid or expired token")
	}

	if RevocationRDB != nil && claims.IssuedAt != nil {
		revokeKey := "revoke-before:" + claims.UserID
		val, redisErr := RevocationRDB.Get(context.Background(), revokeKey).Result()
		if redisErr == nil {
			revokedBefore, parseErr := strconv.ParseInt(val, 10, 64)
			if parseErr == nil && claims.IssuedAt.Unix() < revokedBefore {
				return "", fmt.Errorf("session expired — please sign in again")
			}
		}
	}
	return claims.UserID, nil
}

// parseFullClaims parses a JWT token and returns the full claims struct.
func parseFullClaims(tokenStr string) *Claims {
	claims := &Claims{}
	jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) { //nolint:errcheck
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	return claims
}

// Auth validates the JWT Bearer token in the Authorization header.
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization format"})
			return
		}

		userID, err := ValidateToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		claims := parseFullClaims(parts[1])

		c.Set("user_id", userID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)
		c.Next()
	}
}

// AdminWithDB fetches the user from DB to verify admin role.
// Must be used AFTER Auth() which sets "user_id" in context.
func AdminWithDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		var roleResult struct{ Role string }
		if err := db.Table("users").Select("role").Where("id = ?", userID).Scan(&roleResult).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server error"})
			return
		}
		if roleResult.Role != "admin" && roleResult.Role != "super_admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Set("user_role", roleResult.Role)
		c.Next()
	}
}

// AdminOnly checks user_role set in context (lightweight, DB-free).
// Use AdminWithDB for routes that need fresh role verification.
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, _ := c.Get("user_role")
		if role != "admin" && role != "super_admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}
