package fraud

import (
        "context"
        "fmt"
        "log/slog"
        "time"

        "github.com/google/uuid"
        "github.com/redis/go-redis/v9"
        "gorm.io/gorm"
)

// RiskLevel categorizes the overall fraud score.
type RiskLevel string

const (
        RiskLow    RiskLevel = "low"    // score < 0.40
        RiskMedium RiskLevel = "medium" // score 0.40 – 0.69
        RiskHigh   RiskLevel = "high"   // score >= 0.70

        highRiskThreshold = 0.70
)

// Signal is a single observable fraud indicator.
type Signal struct {
        Name        string  `json:"name"`
        Value       float64 `json:"value"`
        Description string  `json:"description"`
}

// Score is the result returned by Evaluate.
type Score struct {
        UserID    uuid.UUID `json:"user_id"`
        RiskScore float64   `json:"risk_score"` // 0.0 – 1.0
        Level     RiskLevel `json:"level"`
        Signals   []Signal  `json:"signals"`
}

// Detector evaluates fraud risk for a given user using live DB and Redis signals.
type Detector struct {
        db  *gorm.DB
        rdb *redis.Client
}

// New creates a Detector. rdb may be nil — Redis-based signals are skipped gracefully.
func New(db *gorm.DB, rdb *redis.Client) *Detector {
        return &Detector{db: db, rdb: rdb}
}

// Evaluate computes a composite fraud risk score for the given user by querying
// live DB and Redis signals. It logs a warning for high-risk results.
//
// Signal weights:
//
//      listing_velocity (0–10+ listings/24 h) — weight 0.40
//      bid_velocity     (0–20+ bids/1 h)      — weight 0.35
//      new_account      (account < 10 min old) — weight 0.25
func (d *Detector) Evaluate(ctx context.Context, userID uuid.UUID) Score {
        signals := make([]Signal, 0, 3)
        totalWeight := 0.0

        // ── Signal 1: Listing velocity — live DB count ──────────────────────────────
        // More than 10 listings in 24 h is suspicious (score saturates at 10).
        listingVelocityScore, listingCount := d.listingVelocity(ctx, userID)
        signals = append(signals, Signal{
                Name:        "listing_velocity",
                Value:       float64(listingCount),
                Description: fmt.Sprintf("%d listing(s) created in the last 24 hours", listingCount),
        })
        totalWeight += listingVelocityScore * 0.40

        // ── Signal 2: Bid velocity — live DB count ──────────────────────────────────
        // More than 20 bids in 1 h is suspicious (score saturates at 20).
        bidVelocityScore, bidCount := d.bidVelocity(ctx, userID)
        signals = append(signals, Signal{
                Name:        "bid_velocity",
                Value:       float64(bidCount),
                Description: fmt.Sprintf("%d bid(s) placed in the last hour", bidCount),
        })
        totalWeight += bidVelocityScore * 0.35

        // ── Signal 3: New account flag — live DB check ──────────────────────────────
        // Accounts created within the last 10 minutes are treated as untrusted.
        newAccountScore, accountAgeMin := d.newAccountFlag(ctx, userID)
        signals = append(signals, Signal{
                Name:        "new_account",
                Value:       float64(accountAgeMin),
                Description: fmt.Sprintf("account created %.0f minute(s) ago", accountAgeMin),
        })
        totalWeight += newAccountScore * 0.25

        level := classifyRisk(totalWeight)

        result := Score{
                UserID:    userID,
                RiskScore: clamp(totalWeight),
                Level:     level,
                Signals:   signals,
        }

        if totalWeight >= highRiskThreshold {
                slog.Warn("fraud: high-risk user detected",
                        "user_id",         userID.String(),
                        "risk_score",      result.RiskScore,
                        "listing_count",   listingCount,
                        "bid_count",       bidCount,
                        "account_age_min", accountAgeMin,
                )
        }

        return result
}

// ── Signal implementations ────────────────────────────────────────────────────

// listingVelocity counts listings the user created in the past 24 hours.
// Returns a normalized 0–1 score (saturates at 10 listings) and the raw count.
// On DB error, logs and returns a conservative score of 1.0 (fail-strict).
func (d *Detector) listingVelocity(ctx context.Context, userID uuid.UUID) (score float64, count int) {
        since := time.Now().Add(-24 * time.Hour)
        var total int64
        // listings.user_id holds the creator's ID (not seller_id).
        err := d.db.WithContext(ctx).
                Table("listings").
                Where("user_id = ? AND created_at >= ?", userID, since).
                Count(&total).Error
        if err != nil {
                slog.Error("fraud: listingVelocity DB query failed",
                        "user_id", userID.String(), "error", err.Error())
                return 1.0, 0 // conservative: treat unknown velocity as high risk
        }
        count = int(total)

        const cap = 10
        if count >= cap {
                return 1.0, count
        }
        return float64(count) / float64(cap), count
}

// bidVelocity counts bids the user placed in the past hour.
// Returns a normalized 0–1 score (saturates at 20 bids) and the raw count.
// On DB error, logs and returns a conservative score of 1.0 (fail-strict).
func (d *Detector) bidVelocity(ctx context.Context, userID uuid.UUID) (score float64, count int) {
        since := time.Now().Add(-time.Hour)
        var total int64
        // bids.user_id is the bidder; bids.placed_at is the timestamp (not created_at).
        err := d.db.WithContext(ctx).
                Table("bids").
                Where("user_id = ? AND placed_at >= ?", userID, since).
                Count(&total).Error
        if err != nil {
                slog.Error("fraud: bidVelocity DB query failed",
                        "user_id", userID.String(), "error", err.Error())
                return 1.0, 0 // conservative: treat unknown velocity as high risk
        }
        count = int(total)

        const cap = 20
        if count >= cap {
                return 1.0, count
        }
        return float64(count) / float64(cap), count
}

// newAccountFlag checks whether the account was created within the last 10 minutes.
// Returns 1.0 if the account is brand-new, 0.0 otherwise, plus the age in minutes.
// On DB error, logs and returns 0.0 (fail-open: don't penalize if age unknown).
func (d *Detector) newAccountFlag(ctx context.Context, userID uuid.UUID) (score float64, ageMin float64) {
        var createdAt time.Time
        err := d.db.WithContext(ctx).
                Table("users").
                Where("id = ?", userID).
                Pluck("created_at", &createdAt).Error
        if err != nil {
                slog.Error("fraud: newAccountFlag DB query failed",
                        "user_id", userID.String(), "error", err.Error())
                return 0.0, 0 // fail-open: unknown age should not penalize the user
        }
        if createdAt.IsZero() {
                return 0, 0
        }
        ageMin = time.Since(createdAt).Minutes()
        if ageMin < 10 {
                return 1.0, ageMin
        }
        return 0.0, ageMin
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func classifyRisk(score float64) RiskLevel {
        switch {
        case score >= highRiskThreshold:
                return RiskHigh
        case score >= 0.40:
                return RiskMedium
        default:
                return RiskLow
        }
}

func clamp(v float64) float64 {
        if v > 1.0 {
                return 1.0
        }
        if v < 0.0 {
                return 0.0
        }
        return v
}
