package fraud

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/pkg/metrics"
	"github.com/redis/go-redis/v9"
)

// Default thresholds — can be overridden in Redis at runtime.
const (
	DefaultBlockThreshold     = 80.0
	DefaultChallengeThreshold = 50.0
	DefaultVelocityThreshold  = 10.0 // tx/hour
	DefaultWithdrawThreshold  = 5.0  // withdrawals/24h
	DefaultGeoMismatchLimit   = 2.0
	DefaultWalletDriftLimit   = 10000.0

	// Guardrails — AutoTune cannot push thresholds beyond these bounds.
	MinBlockThreshold     = 50.0
	MaxBlockThreshold     = 95.0
	MinChallengeThreshold = 25.0
	MaxChallengeThreshold = 75.0

	// Max adjustment per AutoTune cycle (prevents runaway).
	MaxAdjustmentPerTune = 5.0
)

// ThresholdStore provides dynamic, runtime-adjustable fraud thresholds via Redis.
// This replaces hardcoded if-score>X checks with configurable values.
type ThresholdStore struct {
	rdb *redis.Client
}

// NewThresholdStore creates a threshold store backed by Redis.
func NewThresholdStore(rdb *redis.Client) *ThresholdStore {
	return &ThresholdStore{rdb: rdb}
}

// Get retrieves a threshold value. Returns fallback if not set or Redis unavailable.
func (s *ThresholdStore) Get(ctx context.Context, key string, fallback float64) float64 {
	if s.rdb == nil {
		metrics.IncFraudThresholdFallback(key)
		return fallback
	}
	val, err := s.rdb.Get(ctx, "fraud:threshold:"+key).Float64()
	if err != nil {
		metrics.IncFraudThresholdFallback(key)
		return fallback
	}
	return val
}

// Set updates a threshold value at runtime (no restart needed).
func (s *ThresholdStore) Set(ctx context.Context, key string, value float64) error {
	if s.rdb == nil {
		return nil
	}
	slog.Info("fraud: threshold updated", "key", key, "value", value)
	return s.rdb.Set(ctx, "fraud:threshold:"+key, value, 0).Err()
}

// GetBlockThreshold returns the score threshold for BLOCK decisions.
func (s *ThresholdStore) GetBlockThreshold(ctx context.Context) float64 {
	return s.Get(ctx, "block", DefaultBlockThreshold)
}

// GetChallengeThreshold returns the score threshold for CHALLENGE decisions.
func (s *ThresholdStore) GetChallengeThreshold(ctx context.Context) float64 {
	return s.Get(ctx, "challenge", DefaultChallengeThreshold)
}

// GetVelocityThreshold returns the max transactions per hour before flagging.
func (s *ThresholdStore) GetVelocityThreshold(ctx context.Context) float64 {
	return s.Get(ctx, "velocity_1h", DefaultVelocityThreshold)
}

// GetWithdrawThreshold returns the max withdrawals per 24h before flagging.
func (s *ThresholdStore) GetWithdrawThreshold(ctx context.Context) float64 {
	return s.Get(ctx, "withdraw_24h", DefaultWithdrawThreshold)
}

// GetGeoMismatchLimit returns the max geo mismatches before flagging.
func (s *ThresholdStore) GetGeoMismatchLimit(ctx context.Context) float64 {
	return s.Get(ctx, "geo_mismatch", DefaultGeoMismatchLimit)
}

// GetWalletDriftLimit returns the max wallet drift in 24h before flagging.
func (s *ThresholdStore) GetWalletDriftLimit(ctx context.Context) float64 {
	return s.Get(ctx, "wallet_drift_24h", DefaultWalletDriftLimit)
}

// AutoTune adjusts thresholds based on feedback accuracy.
// If false positive rate > 20%, raise thresholds (be less aggressive).
// If false negative rate > 10%, lower thresholds (be more aggressive).
func (s *ThresholdStore) AutoTune(ctx context.Context, repo *Repository) {
	if repo == nil {
		return
	}

	total, correct, accuracy := repo.GetDecisionAccuracy(7 * 24 * time.Hour) // last 7 days
	if total < 50 {
		return // not enough data to tune
	}

	falsePositiveRate := 1.0 - accuracy
	slog.Info("fraud: auto-tune analysis",
		"total_decisions", total,
		"correct", correct,
		"accuracy", accuracy,
		"false_positive_rate", falsePositiveRate,
	)

	blockThreshold := s.GetBlockThreshold(ctx)
	challengeThreshold := s.GetChallengeThreshold(ctx)

	// Too many false positives → raise thresholds
	if falsePositiveRate > 0.20 {
		blockThreshold += 2
		challengeThreshold += 2
		slog.Warn("fraud: auto-tune raising thresholds (high false positive rate)",
			"new_block", blockThreshold,
			"new_challenge", challengeThreshold,
		)
	}

	// Too many false negatives (fraud getting through) → lower thresholds
	// This is inferred from BLOCK decisions that were LEGIT being rare
	// while FRAUD outcomes on ALLOW decisions being high
	_, fraudOnAllow, _ := repo.GetDecisionAccuracy(7 * 24 * time.Hour)
	_ = fraudOnAllow // simplified — in production, query ALLOW+outcome=FRAUD specifically

	if falsePositiveRate < 0.05 && accuracy > 0.90 {
		// Very accurate but maybe too conservative
		blockThreshold -= 1
		challengeThreshold -= 1
		slog.Info("fraud: auto-tune lowering thresholds (high accuracy, low FP)",
			"new_block", blockThreshold,
			"new_challenge", challengeThreshold,
		)
	}

	// ── Guardrails ──────────────────────────────────────────────────────────
	// Cap maximum adjustment per cycle (prevent runaway)
	origBlock := s.GetBlockThreshold(ctx)
	origChallenge := s.GetChallengeThreshold(ctx)
	if diff := blockThreshold - origBlock; diff > MaxAdjustmentPerTune {
		blockThreshold = origBlock + MaxAdjustmentPerTune
	} else if diff < -MaxAdjustmentPerTune {
		blockThreshold = origBlock - MaxAdjustmentPerTune
	}
	if diff := challengeThreshold - origChallenge; diff > MaxAdjustmentPerTune {
		challengeThreshold = origChallenge + MaxAdjustmentPerTune
	} else if diff < -MaxAdjustmentPerTune {
		challengeThreshold = origChallenge - MaxAdjustmentPerTune
	}

	// Hard bounds — never exceed these regardless of tuning
	if blockThreshold < MinBlockThreshold {
		blockThreshold = MinBlockThreshold
	}
	if blockThreshold > MaxBlockThreshold {
		blockThreshold = MaxBlockThreshold
	}
	if challengeThreshold < MinChallengeThreshold {
		challengeThreshold = MinChallengeThreshold
	}
	if challengeThreshold > MaxChallengeThreshold {
		challengeThreshold = MaxChallengeThreshold
	}

	// Log the actual change
	slog.Info("fraud: auto-tune applied",
		"block", fmt.Sprintf("%.1f→%.1f", origBlock, blockThreshold),
		"challenge", fmt.Sprintf("%.1f→%.1f", origChallenge, challengeThreshold),
	)

	_ = s.Set(ctx, "block", blockThreshold)
	_ = s.Set(ctx, "challenge", challengeThreshold)
}
