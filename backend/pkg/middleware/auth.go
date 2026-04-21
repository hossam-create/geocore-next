package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/geocore-next/backend/pkg/jwtkeys"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

const (
	PermAdminDashboardRead = "admin.dashboard.read"

	PermUsersRead   = "users.read"
	PermUsersWrite  = "users.write"
	PermUsersDelete = "users.delete"
	PermUsersBan    = "users.ban"

	PermListingsModerate = "listings.moderate"
	PermListingsDelete   = "listings.delete"

	PermFinanceRead = "finance.read"

	PermAuditLogsRead = "audit.logs.read"

	PermCatalogManage = "catalog.manage"
	PermPlansManage   = "plans.manage"

	PermReportsReview = "reports.review"

	PermOpsRead   = "ops.read"
	PermOpsManage = "ops.manage"

	PermSettingsRead  = "settings.read"
	PermSettingsWrite = "settings.write"

	PermSupportTicketsRead  = "support.tickets.read"
	PermSupportTicketsReply = "support.tickets.reply"
	PermSupportTicketsWrite = "support.tickets.write"
)

func buildPermSet(perms ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(perms))
	for _, p := range perms {
		set[p] = struct{}{}
	}
	return set
}

var rolePermissions = map[string]map[string]struct{}{
	"admin": buildPermSet(
		PermAdminDashboardRead,
		PermUsersRead, PermUsersWrite, PermUsersDelete, PermUsersBan,
		PermListingsModerate, PermListingsDelete,
		PermFinanceRead,
		PermAuditLogsRead,
		PermCatalogManage,
		PermPlansManage,
		PermReportsReview,
		PermOpsRead, PermOpsManage,
		PermSettingsRead, PermSettingsWrite,
		PermSupportTicketsRead, PermSupportTicketsReply, PermSupportTicketsWrite,
	),
	"ops_admin": buildPermSet(
		PermAdminDashboardRead,
		PermListingsModerate,
		PermReportsReview,
		PermOpsRead, PermOpsManage,
	),
	"finance_admin": buildPermSet(
		PermAdminDashboardRead,
		PermFinanceRead,
		PermAuditLogsRead,
	),
	"support_admin": buildPermSet(
		PermAdminDashboardRead,
		PermSupportTicketsRead, PermSupportTicketsReply, PermSupportTicketsWrite,
		PermReportsReview,
	),
}

func isInternalRole(role string) bool {
	if role == "super_admin" {
		return true
	}
	_, ok := rolePermissions[role]
	return ok
}

func hasPermission(role, perm string) bool {
	if role == "super_admin" {
		return true
	}
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	_, ok = perms[perm]
	return ok
}

var RevocationRDB *redis.Client

func ValidateToken(tokenStr string) (string, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return jwtkeys.Public(), nil
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

		claims := &Claims{}
		jwt.ParseWithClaims(parts[1], claims, func(t *jwt.Token) (interface{}, error) {
			return jwtkeys.Public(), nil
		})

		c.Set("user_id", userID)
		c.Set("user_email", claims.Email)
		c.Next()
	}
}

func AdminWithDB(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString("user_id")
		var roleResult struct{ Role string }
		if err := db.Table("users").Select("role").Where("id = ?", userID).Scan(&roleResult).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "server error"})
			return
		}
		if !isInternalRole(roleResult.Role) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Set("user_role", roleResult.Role)
		c.Next()
	}
}

func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("user_role")
		if !isInternalRole(role) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin access required"})
			return
		}
		c.Next()
	}
}

func RequireAnyPermission(perms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(perms) == 0 {
			c.Next()
			return
		}

		role := c.GetString("user_role")
		for _, p := range perms {
			if hasPermission(role, p) {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}

func RequireAllPermissions(perms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("user_role")
		for _, p := range perms {
			if !hasPermission(role, p) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
				return
			}
		}

		c.Next()
	}
}
