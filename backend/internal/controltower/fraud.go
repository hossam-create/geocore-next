package controltower

import (
	"time"

	"gorm.io/gorm"
)

// FraudRadar is the aggregated fraud intelligence snapshot.
type FraudRadar struct {
	SuspiciousUsers  []SuspiciousUser  `json:"suspicious_users"`
	HighRiskMatches  []RiskyMatch      `json:"high_risk_matches"`
	RecentBlocks     []RecentBlock     `json:"recent_blocks"`
	FraudTrends      []FraudTrendPoint `json:"fraud_trends"`
	TotalFlagged     int64             `json:"total_flagged_24h"`
	CircularTrades   int64             `json:"circular_trades_detected"`
	FakeProofAttempts int64            `json:"fake_proof_attempts"`
	CapturedAt       time.Time         `json:"captured_at"`
}

type SuspiciousUser struct {
	UserID    string    `json:"user_id"`
	RiskScore int       `json:"risk_score"`
	Reason    string    `json:"reason"`
	FlaggedAt time.Time `json:"flagged_at"`
}

type RiskyMatch struct {
	MatchID     string    `json:"match_id"`
	RiskScore   int       `json:"risk_score"`
	RiskReasons []string  `json:"risk_reasons"`
	CreatedAt   time.Time `json:"created_at"`
}

type RecentBlock struct {
	IPAddress string    `json:"ip_address"`
	Reason    string    `json:"reason"`
	BlockedAt time.Time `json:"blocked_at"`
}

type FraudTrendPoint struct {
	Hour  string `json:"hour"`
	Count int64  `json:"count"`
}

// GetFraudRadar aggregates fraud signals from exchange risk, audit log, and IDS.
func GetFraudRadar(db *gorm.DB) FraudRadar {
	r := FraudRadar{CapturedAt: time.Now().UTC()}

	// Suspicious users from exchange risk flags (last 24h).
	db.Raw(`
		SELECT
			requester_id::text AS user_id,
			risk_score,
			risk_reasons AS reason,
			created_at AS flagged_at
		FROM exchange_risk_flags
		WHERE created_at > NOW() - INTERVAL '24 hours'
		  AND risk_score >= 60
		ORDER BY risk_score DESC
		LIMIT 50
	`).Scan(&r.SuspiciousUsers)

	// High-risk matches.
	db.Raw(`
		SELECT
			m.id::text AS match_id,
			f.risk_score,
			f.risk_reasons,
			m.created_at
		FROM exchange_matches m
		JOIN exchange_risk_flags f ON f.match_id = m.id
		WHERE m.created_at > NOW() - INTERVAL '24 hours'
		  AND f.risk_score >= 70
		ORDER BY f.risk_score DESC
		LIMIT 20
	`).Scan(&r.HighRiskMatches)

	// Recent IDS blocks from audit log.
	db.Raw(`
		SELECT
			ip_address,
			details->>'reason' AS reason,
			created_at AS blocked_at
		FROM security_audit_log
		WHERE event_type = 'auto_block'
		  AND created_at > NOW() - INTERVAL '24 hours'
		ORDER BY created_at DESC
		LIMIT 30
	`).Scan(&r.RecentBlocks)

	// Hourly fraud trend (last 24h).
	db.Raw(`
		SELECT
			TO_CHAR(DATE_TRUNC('hour', created_at), 'HH24:00') AS hour,
			COUNT(*)                                             AS count
		FROM security_audit_log
		WHERE event_type IN ('auto_block','rate_limited','login_failed')
		  AND created_at > NOW() - INTERVAL '24 hours'
		GROUP BY DATE_TRUNC('hour', created_at)
		ORDER BY DATE_TRUNC('hour', created_at)
	`).Scan(&r.FraudTrends)

	// Total flagged.
	db.Raw(`SELECT COUNT(*) FROM exchange_risk_flags WHERE created_at > NOW() - INTERVAL '24 hours'`).Scan(&r.TotalFlagged)

	// Circular trades (settlements with circular_trade flag).
	db.Raw(`
		SELECT COUNT(*) FROM exchange_risk_flags
		WHERE risk_reasons::text ILIKE '%circular%'
		  AND created_at > NOW() - INTERVAL '24 hours'
	`).Scan(&r.CircularTrades)

	// Fake proof attempts (dispute with fake_proof reason).
	db.Raw(`
		SELECT COUNT(*) FROM exchange_settlements
		WHERE proof_status = 'rejected'
		  AND updated_at > NOW() - INTERVAL '24 hours'
	`).Scan(&r.FakeProofAttempts)

	return r
}
