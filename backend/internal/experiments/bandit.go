package experiments

import (
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Bandit Optimization ──────────────────────────────────────────────────────────
//
// Thompson Sampling + UCB1 for multi-armed bandit optimization.
// Use cases: best notification timing, best message wording, best boost pricing, best insurance price.

// BanditArm represents an arm (variant) in a bandit experiment.
type BanditArm struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ExperimentID uuid.UUID `gorm:"type:uuid;not null;index" json:"experiment_id"`
	ArmName      string    `gorm:"size:50;not null" json:"arm_name"` // e.g., "9am", "2pm", "8pm"
	Alpha        float64   `gorm:"type:numeric(12,4);not null;default:1" json:"alpha"` // Beta distribution α
	Beta         float64   `gorm:"type:numeric(12,4);not null;default:1" json:"beta"`  // Beta distribution β
	TotalPulls   int       `gorm:"not null;default:0" json:"total_pulls"`
	TotalReward  float64   `gorm:"type:numeric(12,4);not null;default:0" json:"total_reward"`
	AvgReward    float64   `gorm:"type:numeric(8,4);not null;default:0" json:"avg_reward"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (BanditArm) TableName() string { return "experiment_bandit_arms" }

// BanditPull records a single pull (selection + reward).
type BanditPull struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ExperimentID uuid.UUID `gorm:"type:uuid;not null;index" json:"experiment_id"`
	ArmID        uuid.UUID `gorm:"type:uuid;not null;index" json:"arm_id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	Reward       float64   `gorm:"type:numeric(8,4);not null;default:0" json:"reward"` // 0 or 1 (binary)
	CreatedAt    time.Time `json:"created_at"`
}

func (BanditPull) TableName() string { return "experiment_bandit_pulls" }

// ── Thompson Sampling ────────────────────────────────────────────────────────────────

// ThompsonSample selects an arm using Thompson Sampling.
// Samples from Beta(α, β) for each arm, picks the one with highest sample.
func ThompsonSample(db *gorm.DB, experimentID uuid.UUID) *BanditArm {
	var arms []BanditArm
	db.Where("experiment_id = ?", experimentID).Find(&arms)

	if len(arms) == 0 {
		return nil
	}

	bestArm := &arms[0]
	bestSample := -1.0

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := range arms {
		// Sample from Beta(alpha, beta)
		sample := rngBeta(rng, arms[i].Alpha, arms[i].Beta)
		if sample > bestSample {
			bestSample = sample
			bestArm = &arms[i]
		}
	}

	return bestArm
}

// UpdateBanditArm updates arm statistics after observing a reward.
func UpdateBanditArm(db *gorm.DB, armID uuid.UUID, reward float64) {
	var arm BanditArm
	if err := db.Where("id = ?", armID).First(&arm).Error; err != nil {
		return
	}

	arm.TotalPulls++
	arm.TotalReward += reward
	arm.AvgReward = arm.TotalReward / float64(arm.TotalPulls)

	// Update Beta distribution parameters
	if reward > 0 {
		arm.Alpha += reward // success
	} else {
		arm.Beta += (1 - reward) // failure
	}

	db.Save(&arm)
}

// RecordBanditPull records a pull and updates the arm.
func RecordBanditPull(db *gorm.DB, experimentID, armID, userID uuid.UUID, reward float64) {
	db.Create(&BanditPull{
		ExperimentID: experimentID,
		ArmID:        armID,
		UserID:       userID,
		Reward:       reward,
	})
	UpdateBanditArm(db, armID, reward)
}

// ── UCB1 ────────────────────────────────────────────────────────────────────────────

// UCB1Select selects an arm using Upper Confidence Bound 1.
func UCB1Select(db *gorm.DB, experimentID uuid.UUID) *BanditArm {
	var arms []BanditArm
	db.Where("experiment_id = ?", experimentID).Find(&arms)

	if len(arms) == 0 {
		return nil
	}

	// Total pulls across all arms
	totalPulls := 0
	for _, a := range arms {
		totalPulls += a.TotalPulls
	}

	if totalPulls == 0 {
		// No pulls yet — pick random
		return &arms[0]
	}

	bestArm := &arms[0]
	bestUCB := -1.0

	for i := range arms {
		if arms[i].TotalPulls == 0 {
			return &arms[i] // must try unexplored arm first
		}

		// UCB1 = avg_reward + sqrt(2 * ln(total_pulls) / arm_pulls)
		ucb := arms[i].AvgReward + math.Sqrt(2*math.Log(float64(totalPulls))/float64(arms[i].TotalPulls))
		if ucb > bestUCB {
			bestUCB = ucb
			bestArm = &arms[i]
		}
	}

	return bestArm
}

// ── Bandit Creation ──────────────────────────────────────────────────────────────────

// CreateBanditArms creates arms for a bandit experiment.
func CreateBanditArms(db *gorm.DB, experimentID uuid.UUID, armNames []string) {
	for _, name := range armNames {
		db.Create(&BanditArm{
			ExperimentID: experimentID,
			ArmName:      name,
			Alpha:        1.0, // Beta(1,1) = uniform prior
			Beta:         1.0,
		})
	}
}

// ── Bandit Metrics ────────────────────────────────────────────────────────────────────

type BanditMetrics struct {
	ExperimentID uuid.UUID    `json:"experiment_id"`
	Arms         []BanditArm  `json:"arms"`
	BestArm      string       `json:"best_arm"`
	TotalPulls   int          `json:"total_pulls"`
	Regret       float64      `json:"regret"` // difference from optimal
}

func GetBanditMetrics(db *gorm.DB, experimentID uuid.UUID) *BanditMetrics {
	var arms []BanditArm
	db.Where("experiment_id = ?", experimentID).Order("avg_reward DESC").Find(&arms)

	totalPulls := 0
	bestArm := ""
	bestAvg := 0.0
	for _, a := range arms {
		totalPulls += a.TotalPulls
		if a.AvgReward > bestAvg {
			bestAvg = a.AvgReward
			bestArm = a.ArmName
		}
	}

	// Simple regret estimate: (optimal_reward - actual_avg_reward)
	actualAvg := 0.0
	if totalPulls > 0 {
		var totalReward float64
		for _, a := range arms {
			totalReward += a.TotalReward
		}
		actualAvg = totalReward / float64(totalPulls)
	}
	regret := bestAvg - actualAvg

	return &BanditMetrics{
		ExperimentID: experimentID,
		Arms:         arms,
		BestArm:      bestArm,
		TotalPulls:   totalPulls,
		Regret:       math.Max(0, regret),
	}
}

// ── Beta Distribution Sampling ────────────────────────────────────────────────────────

// rngBeta samples from Beta(α, β) using the Johnk algorithm.
func rngBeta(rng *rand.Rand, alpha, beta float64) float64 {
	if alpha < 1 || beta < 1 {
		// Fallback: simple uniform for small parameters
		return rng.Float64()
	}

	for {
		x := rngGamma(rng, alpha)
		y := rngGamma(rng, beta)
		if x+y > 0 {
			return x / (x + y)
		}
	}
}

// rngGamma samples from Gamma(shape, 1) using Marsaglia and Tsang's method.
func rngGamma(rng *rand.Rand, shape float64) float64 {
	if shape < 1 {
		return rngGamma(rng, shape+1) * math.Pow(rng.Float64(), 1/shape)
	}

	d := shape - 1.0/3.0
	c := 1.0 / math.Sqrt(9*d)

	for {
		x := rng.NormFloat64()
		v := 1 + c*x
		if v <= 0 {
			continue
		}
		v = v * v * v
		u := rng.Float64()
		if u < 1-0.0331*(x*x)*(x*x) {
			return d * v
		}
		if math.Log(u) < 0.5*x*x+d*(1-v+math.Log(v)) {
			return d * v
		}
	}
}
