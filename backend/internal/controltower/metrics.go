package controltower

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// SystemMetrics is the live system health snapshot.
type SystemMetrics struct {
	ActiveUsers               int64     `json:"active_users"`
	ActiveSessions            int64     `json:"active_sessions"`
	BidsPerSecond             float64   `json:"bids_per_second"`
	ExchangeRequestsPerSecond float64   `json:"exchange_requests_per_second"`
	ErrorRate                 float64   `json:"error_rate"` // errors per 100 requests (last 5 min)
	BlockedUsers              int64     `json:"blocked_users"`
	PendingWithdrawals        int64     `json:"pending_withdrawals"`
	OpenExchangeRequests      int64     `json:"open_exchange_requests"`
	CapturedAt                time.Time `json:"captured_at"`
}

// GetSystemMetrics aggregates live system health from DB + Redis.
func GetSystemMetrics(db *gorm.DB, rdb *redis.Client) SystemMetrics {
	m := SystemMetrics{CapturedAt: time.Now().UTC()}

	// Active sessions (livestream, last 24h).
	db.Raw(`SELECT COUNT(*) FROM livestream_sessions WHERE status = 'live'`).Scan(&m.ActiveSessions)

	// Active users — users who placed a bid or exchange request in last 15 min.
	db.Raw(`SELECT COUNT(DISTINCT user_id) FROM (
		SELECT bidder_id AS user_id FROM live_auction_bids WHERE created_at > NOW() - INTERVAL '15 minutes'
		UNION
		SELECT requester_id AS user_id FROM exchange_requests WHERE created_at > NOW() - INTERVAL '15 minutes'
	) t`).Scan(&m.ActiveUsers)

	// Bids/sec — bids in last 60 sec / 60.
	var bidCount int64
	db.Raw(`SELECT COUNT(*) FROM live_auction_bids WHERE created_at > NOW() - INTERVAL '60 seconds'`).Scan(&bidCount)
	m.BidsPerSecond = float64(bidCount) / 60.0

	// Exchange req/sec.
	var exchCount int64
	db.Raw(`SELECT COUNT(*) FROM exchange_requests WHERE created_at > NOW() - INTERVAL '60 seconds'`).Scan(&exchCount)
	m.ExchangeRequestsPerSecond = float64(exchCount) / 60.0

	// Error rate: login_failed + rate_limited events in last 5 min per 100 total.
	var errEvents, totalEvents int64
	db.Raw(`SELECT COUNT(*) FROM security_audit_log WHERE event_type IN ('login_failed','rate_limited') AND created_at > NOW() - INTERVAL '5 minutes'`).Scan(&errEvents)
	db.Raw(`SELECT COUNT(*) FROM security_audit_log WHERE created_at > NOW() - INTERVAL '5 minutes'`).Scan(&totalEvents)
	if totalEvents > 0 {
		m.ErrorRate = float64(errEvents) / float64(totalEvents) * 100
	}

	// Blocked users in Redis (IDS).
	if rdb != nil {
		keys, _ := rdb.Keys(context.Background(), "ids:blocked:*").Result()
		m.BlockedUsers = int64(len(keys))
	}

	// Pending withdrawals.
	db.Raw(`SELECT COUNT(*) FROM withdraw_requests WHERE status = 'pending'`).Scan(&m.PendingWithdrawals)

	// Open exchange requests.
	db.Raw(`SELECT COUNT(*) FROM exchange_requests WHERE status = 'open'`).Scan(&m.OpenExchangeRequests)

	return m
}

// BlockedIPList returns the list of currently blocked IPs from Redis.
func BlockedIPList(rdb *redis.Client) []string {
	if rdb == nil {
		return nil
	}
	keys, _ := rdb.Keys(context.Background(), "ids:blocked:*").Result()
	ips := make([]string, 0, len(keys))
	for _, k := range keys {
		ips = append(ips, k[len("ids:blocked:"):])
	}
	return ips
}
