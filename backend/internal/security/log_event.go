package security

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/geocore-next/backend/internal/config"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Extended security event types (Sprint 23).
const (
	EventSuspiciousActivity = "suspicious_activity"
	EventFraudFlag          = "fraud_flag"
	EventRapidActions       = "rapid_actions"
	EventMultipleIPs        = "multiple_ips"
	EventReplayAttack       = "replay_attack"
	EventUserFrozen         = "user_frozen"
	EventUserUnfrozen       = "user_unfrozen"
	EventAdminBlockIP       = "admin_block_ip"
	EventAdminUnblockIP     = "admin_unblock_ip"
)

// Severity levels.
const (
	SevSecLow      = "low"
	SevSecMedium   = "medium"
	SevSecHigh     = "high"
	SevSecCritical = "critical"
)

// EventPayload is forwarded to the external event-bus hook.
type EventPayload struct {
	UserID    string
	IP        string
	EventType string
	Severity  string
	Message   string
	Metadata  map[string]any
	Timestamp time.Time
}

// EmitHook is an optional callback wired from cmd/api/main.go that forwards
// events into the controltower.EventBus. Using a hook avoids a direct import
// cycle (controltower already imports security).
var EmitHook func(EventPayload)

// sharedRedis is set by InitSecurityOps so rules can use Redis counters.
var sharedRedis *redis.Client
var sharedRedisMu sync.RWMutex

// InitSecurityOps injects the Redis client used by rule evaluation.
func InitSecurityOps(rdb *redis.Client) {
	sharedRedisMu.Lock()
	sharedRedis = rdb
	sharedRedisMu.Unlock()
}

func getRedis() *redis.Client {
	sharedRedisMu.RLock()
	defer sharedRedisMu.RUnlock()
	return sharedRedis
}

// LogSecurityEvent is the unified entry point for Sprint 23.
// It persists the event, updates the user's risk profile, emits to the live bus,
// and — when ENABLE_AUTO_FREEZE is on — auto-freezes accounts above the threshold.
//
// Call sites:
//   - auth: failed login → LogSecurityEvent(db, uid, ip, EventLoginFailed, SevSecMedium, nil)
//   - exchange risk engine: fraud flag → EventFraudFlag, SevSecCritical
//   - middleware: rate limit → EventRateLimited
//   - livestream bid hub: rapid bids → EventRapidActions
func LogSecurityEvent(
	db *gorm.DB,
	userID *uuid.UUID,
	ip, ua, eventType, severity string,
	metadata map[string]any,
) {
	if !config.GetFlags().EnableSecurityMonitoring {
		return
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["severity"] = severity

	// 1. Persist to audit log.
	entry := SecurityAuditLog{
		UserID:    userID,
		EventType: eventType,
		IPAddress: ip,
		UserAgent: ua,
		Details:   metadata,
		RiskScore: severityScore(severity),
	}
	go db.Create(&entry) //nolint:errcheck

	// 2. Update per-user risk profile.
	if userID != nil && *userID != uuid.Nil {
		delta := scoringDelta(db, *userID, ip, eventType)
		if delta > 0 {
			AddRisk(db, *userID, delta, ip)
		}
		maybeFlagMultipleIPs(db, *userID, ip)
	}

	// 3. Broadcast to live dashboard.
	if EmitHook != nil {
		EmitHook(EventPayload{
			UserID:    uidStr(userID),
			IP:        ip,
			EventType: eventType,
			Severity:  severity,
			Message:   fmt.Sprintf("%s from %s", eventType, ip),
			Metadata:  metadata,
			Timestamp: time.Now().UTC(),
		})
	}
}

// scoringDelta decides how many risk points an event adds, applying burst
// detection for login failures and rapid-action detection for bid-like events.
func scoringDelta(db *gorm.DB, userID uuid.UUID, ip, eventType string) int {
	switch eventType {
	case EventLoginFailed:
		// Only add points once the burst threshold is crossed.
		if loginFailBurstHit(userID, ip) {
			return DeltaLoginFailBurst
		}
		return 0
	case EventFraudFlag:
		return DeltaFraudFlag
	case EventRapidActions:
		return DeltaRapidActions
	case EventSuspiciousActivity:
		return DeltaSuspiciousGeneric
	case EventRateLimited:
		return DeltaRateLimit
	case EventSessionRevoked:
		return DeltaSessionRevoke
	case EventPasswordResetReq:
		return DeltaPasswordReset
	case EventReplayAttack:
		return DeltaFraudFlag
	}
	return 0
}

// loginFailBurstHit returns true when this user+ip just crossed the 5-in-10-min
// failure threshold (one-shot per window so a single burst doesn't over-penalise).
func loginFailBurstHit(userID uuid.UUID, ip string) bool {
	rdb := getRedis()
	if rdb == nil {
		return false
	}
	ctx := context.Background()
	key := fmt.Sprintf("sec:loginfail:%s:%s", userID, ip)
	n, _ := rdb.Incr(ctx, key).Result()
	if n == 1 {
		rdb.Expire(ctx, key, LoginFailBurstWindow)
	}
	// Return true only when we've exactly crossed the threshold this call.
	return n == int64(LoginFailBurstTrigger)
}

// maybeFlagMultipleIPs bumps risk once when the distinct-IP count in 24h
// crosses the threshold.
func maybeFlagMultipleIPs(db *gorm.DB, userID uuid.UUID, ip string) {
	if ip == "" {
		return
	}
	rdb := getRedis()
	if rdb == nil {
		return
	}
	ctx := context.Background()
	setKey := fmt.Sprintf("sec:ips:%s", userID)
	added, _ := rdb.SAdd(ctx, setKey, ip).Result()
	if added == 1 {
		rdb.Expire(ctx, setKey, 24*time.Hour)
	}
	count, _ := rdb.SCard(ctx, setKey).Result()
	if count == int64(MultipleIPsThreshold) {
		AddRisk(db, userID, DeltaMultipleIPs, ip)
		// Update profile device count snapshot (approximation of unique devices).
		db.Model(&UserRiskProfile{}).
			Where("user_id = ?", userID).
			Update("device_count", count)
	}
}

func severityScore(severity string) int {
	switch severity {
	case SevSecCritical:
		return 90
	case SevSecHigh:
		return 70
	case SevSecMedium:
		return 40
	default:
		return 10
	}
}

func uidStr(u *uuid.UUID) string {
	if u == nil || *u == uuid.Nil {
		return ""
	}
	return u.String()
}

// ─── Spike Detection (Part 9) ────────────────────────────────────────────────

// SpikeDetector runs periodic aggregations to find abnormal patterns.
type SpikeDetector struct {
	db     *gorm.DB
	alerts *AlertService
	// "failed_logins" → last alert time, to avoid duplicate alerts.
	lastAlert map[string]time.Time
	mu        sync.Mutex
}

func NewSpikeDetector(db *gorm.DB, alerts *AlertService) *SpikeDetector {
	return &SpikeDetector{db: db, alerts: alerts, lastAlert: map[string]time.Time{}}
}

// Run loops every `interval` until ctx is done. Alerts on:
//   - failed-login spike (>50 in 5 min)
//   - fraud-flag spike (>10 in 15 min)
//   - auto-block spike (>20 in 15 min)
//   - liquidity drop (>50% vs 1h earlier)
func (sd *SpikeDetector) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if !config.GetFlags().EnableSecurityAlerts {
				continue
			}
			sd.checkLoginSpike()
			sd.checkFraudSpike()
			sd.checkBlockSpike()
			sd.checkLiquidityDrop()
		}
	}
}

func (sd *SpikeDetector) checkLoginSpike() {
	var n int64
	sd.db.Raw(`SELECT COUNT(*) FROM security_audit_log
	           WHERE event_type = ? AND created_at > NOW() - INTERVAL '5 minutes'`,
		EventLoginFailed).Scan(&n)
	if n > 50 {
		sd.fire("failed_logins", fmt.Sprintf("Failed-login spike: %d in 5 min", n))
	}
}

func (sd *SpikeDetector) checkFraudSpike() {
	var n int64
	sd.db.Raw(`SELECT COUNT(*) FROM exchange_risk_flags
	           WHERE created_at > NOW() - INTERVAL '15 minutes'`).Scan(&n)
	if n > 10 {
		sd.fire("fraud_flags", fmt.Sprintf("Fraud-flag spike: %d in 15 min", n))
	}
}

func (sd *SpikeDetector) checkBlockSpike() {
	var n int64
	sd.db.Raw(`SELECT COUNT(*) FROM security_audit_log
	           WHERE event_type = 'auto_block' AND created_at > NOW() - INTERVAL '15 minutes'`).Scan(&n)
	if n > 20 {
		sd.fire("auto_blocks", fmt.Sprintf("IDS auto-block spike: %d in 15 min", n))
	}
}

func (sd *SpikeDetector) checkLiquidityDrop() {
	var now, hourAgo float64
	sd.db.Raw(`SELECT COALESCE(SUM(buy_volume + sell_volume), 0) FROM exchange_liquidity_profiles`).Scan(&now)
	// Snapshot stored in Redis — approximation. If unavailable, skip silently.
	rdb := getRedis()
	if rdb == nil {
		return
	}
	ctx := context.Background()
	prev, _ := rdb.Get(ctx, "sec:liq:snapshot").Float64()
	rdb.Set(ctx, "sec:liq:snapshot", now, time.Hour)
	_ = hourAgo
	if prev > 0 && now < prev*0.5 {
		sd.fire("liquidity_drop",
			fmt.Sprintf("Liquidity dropped >50%%: prev=%.0f now=%.0f", prev, now))
	}
}

func (sd *SpikeDetector) fire(key, msg string) {
	sd.mu.Lock()
	last := sd.lastAlert[key]
	// 10-min cooldown per alert key.
	if time.Since(last) < 10*time.Minute {
		sd.mu.Unlock()
		return
	}
	sd.lastAlert[key] = time.Now()
	sd.mu.Unlock()

	if sd.alerts != nil {
		sd.alerts.Send("security_spike", SeverityWarning, msg,
			map[string]string{"rule": key})
	}
	if EmitHook != nil {
		EmitHook(EventPayload{
			EventType: "security_spike",
			Severity:  SevSecHigh,
			Message:   msg,
			Metadata:  map[string]any{"rule": key},
			Timestamp: time.Now().UTC(),
		})
	}
}
