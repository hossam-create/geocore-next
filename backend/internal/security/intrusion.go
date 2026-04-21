package security

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Thresholds.
const (
	ReqSpikeLimit      = 120 // requests per minute before auto-block
	ReqSpikeWindow     = time.Minute
	ReqBlockDuration   = 15 * time.Minute
	LoginFailLimit     = 6 // failures before temp block
	LoginFailWindow    = 10 * time.Minute
	LoginBlockDuration = 30 * time.Minute
)

// IDS is the in-process intrusion detection service backed by Redis.
type IDS struct {
	rdb     *redis.Client
	db      *gorm.DB
	OnBlock func(ip, reason string) // optional callback — wired externally to event bus
}

func NewIDS(rdb *redis.Client, db *gorm.DB) *IDS {
	return &IDS{rdb: rdb, db: db}
}

// IsBlocked returns true when the IP is on the auto-block list.
func (ids *IDS) IsBlocked(ctx context.Context, ip string) bool {
	if ids.rdb == nil {
		return false
	}
	exists, _ := ids.rdb.Exists(ctx, blockKey(ip)).Result()
	return exists > 0
}

// Block manually blocks an IP for the given duration and logs the event.
func (ids *IDS) Block(ctx context.Context, ip, reason string, duration time.Duration) {
	if ids.rdb == nil {
		return
	}
	ids.rdb.Set(ctx, blockKey(ip), reason, duration)
	LogEventDirect(ids.db, nil, "auto_block", ip, "", map[string]any{
		"reason":   reason,
		"duration": duration.String(),
	})
	slog.Warn("ids: IP blocked", "ip", ip, "reason", reason, "duration", duration)
	if ids.OnBlock != nil {
		ids.OnBlock(ip, reason)
	}
}

// TrackRequest increments the per-minute request counter for an IP.
// Auto-blocks when the spike threshold is exceeded.
func (ids *IDS) TrackRequest(ctx context.Context, ip string) {
	if ids.rdb == nil {
		return
	}
	key := fmt.Sprintf("ids:req:%s:%d", ip, time.Now().Unix()/60)
	n, _ := ids.rdb.Incr(ctx, key).Result()
	if n == 1 {
		ids.rdb.Expire(ctx, key, 2*time.Minute)
	}
	if n > ReqSpikeLimit {
		ids.Block(ctx, ip, "request_spike", ReqBlockDuration)
	}
}

// TrackLoginFailure increments login failure counter and blocks on threshold.
func (ids *IDS) TrackLoginFailure(ctx context.Context, ip string) {
	if ids.rdb == nil {
		return
	}
	key := fmt.Sprintf("ids:loginfail:%s", ip)
	n, _ := ids.rdb.Incr(ctx, key).Result()
	if n == 1 {
		ids.rdb.Expire(ctx, key, LoginFailWindow)
	}
	if int(n) >= LoginFailLimit {
		ids.Block(ctx, ip, "repeated_login_failures", LoginBlockDuration)
	}
}

// ResetLoginFailures clears the failure counter after a successful login.
func (ids *IDS) ResetLoginFailures(ctx context.Context, ip string) {
	if ids.rdb != nil {
		ids.rdb.Del(ctx, fmt.Sprintf("ids:loginfail:%s", ip))
	}
}

// Middleware returns a Gin middleware that blocks flagged IPs before routing.
func (ids *IDS) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		ctx := c.Request.Context()
		if ids.IsBlocked(ctx, ip) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":   "access_blocked",
				"message": "Your IP has been temporarily blocked due to suspicious activity.",
			})
			return
		}
		ids.TrackRequest(ctx, ip)
		c.Next()
	}
}

// SuspiciousPatterns scans recent audit logs and auto-blocks repeat offenders.
// Designed to be called from a periodic job.
func (ids *IDS) SuspiciousPatterns(ctx context.Context, window time.Duration) {
	if ids.db == nil {
		return
	}
	since := time.Now().Add(-window)
	type row struct {
		IPAddress string
		Cnt       int64
	}
	var rows []row
	ids.db.Raw(`
		SELECT ip_address, COUNT(*) AS cnt
		FROM security_audit_log
		WHERE event_type = ? AND created_at >= ?
		GROUP BY ip_address
		HAVING COUNT(*) >= ?`,
		EventLoginFailed, since, LoginFailLimit,
	).Scan(&rows)

	for _, r := range rows {
		if !ids.IsBlocked(ctx, r.IPAddress) {
			ids.Block(ctx, r.IPAddress,
				fmt.Sprintf("scan: %d login failures in %s", r.Cnt, window),
				LoginBlockDuration)
		}
	}
}

func blockKey(ip string) string {
	return fmt.Sprintf("ids:blocked:%s", ip)
}
