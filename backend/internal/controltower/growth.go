package controltower

import (
	"time"

	"gorm.io/gorm"
)

// GrowthDashboard covers viral / growth-engine metrics.
type GrowthDashboard struct {
	WaitlistSize       int64            `json:"waitlist_size"`
	WaitlistGrowth24h  int64            `json:"waitlist_growth_24h"`
	InvitesSent        int64            `json:"invites_sent_total"`
	InvitesAccepted    int64            `json:"invites_accepted"`
	ReferralConversion float64          `json:"referral_conversion_pct"`
	QualifiedReferrals int64            `json:"qualified_referrals"`
	KFactor            float64          `json:"k_factor"` // avg invites sent × conversion rate
	TopInviters        []TopInviter     `json:"top_inviters"`
	DailyJoins         []DailyJoinPoint `json:"daily_joins_7d"`
	CapturedAt         time.Time        `json:"captured_at"`
}

type TopInviter struct {
	InviterID  string  `json:"inviter_id"`
	Sent       int64   `json:"invites_sent"`
	Qualified  int64   `json:"qualified"`
	Conversion float64 `json:"conversion_pct"`
}

type DailyJoinPoint struct {
	Date  string `json:"date"`
	Joins int64  `json:"joins"`
}

// GetGrowthDashboard aggregates waitlist + invite/referral stats.
func GetGrowthDashboard(db *gorm.DB) GrowthDashboard {
	g := GrowthDashboard{CapturedAt: time.Now().UTC()}

	// Waitlist size.
	db.Raw(`SELECT COUNT(*) FROM waitlist_users`).Scan(&g.WaitlistSize)
	db.Raw(`SELECT COUNT(*) FROM waitlist_users WHERE created_at > NOW() - INTERVAL '24 hours'`).Scan(&g.WaitlistGrowth24h)

	// Invites.
	db.Raw(`SELECT COUNT(*) FROM invite_usages`).Scan(&g.InvitesAccepted)
	db.Raw(`SELECT COUNT(*) FROM invites`).Scan(&g.InvitesSent)

	// Referral conversion.
	var qualified int64
	db.Raw(`SELECT COUNT(*) FROM invite_usages WHERE referral_status = 'qualified'`).Scan(&qualified)
	g.QualifiedReferrals = qualified
	if g.InvitesAccepted > 0 {
		g.ReferralConversion = float64(qualified) / float64(g.InvitesAccepted) * 100
	}

	// K-factor = (total invites sent / total inviters) × (accepted / sent).
	var totalInviters int64
	db.Raw(`SELECT COUNT(DISTINCT inviter_id) FROM invites`).Scan(&totalInviters)
	if totalInviters > 0 && g.InvitesSent > 0 {
		avgInvitesSent := float64(g.InvitesSent) / float64(totalInviters)
		conversionRate := float64(g.InvitesAccepted) / float64(g.InvitesSent)
		g.KFactor = avgInvitesSent * conversionRate
	}

	// Top inviters.
	db.Raw(`
		SELECT
			i.inviter_id::text                                            AS inviter_id,
			COUNT(u.id)                                                   AS sent,
			COUNT(u.id) FILTER (WHERE u.referral_status = 'qualified')   AS qualified,
			CASE WHEN COUNT(u.id) = 0 THEN 0
			     ELSE COUNT(u.id) FILTER (WHERE u.referral_status = 'qualified')::float
			          / COUNT(u.id) * 100
			END                                                           AS conversion_pct
		FROM invites i
		LEFT JOIN invite_usages u ON u.invite_id = i.id
		GROUP BY i.inviter_id
		ORDER BY qualified DESC
		LIMIT 20
	`).Scan(&g.TopInviters)

	// Daily joins (last 7 days).
	db.Raw(`
		SELECT
			TO_CHAR(DATE_TRUNC('day', created_at), 'YYYY-MM-DD') AS date,
			COUNT(*)                                              AS joins
		FROM waitlist_users
		WHERE created_at >= NOW() - INTERVAL '7 days'
		GROUP BY DATE_TRUNC('day', created_at)
		ORDER BY DATE_TRUNC('day', created_at)
	`).Scan(&g.DailyJoins)

	return g
}
