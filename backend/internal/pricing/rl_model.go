package pricing

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ── RL State ────────────────────────────────────────────────────────────────────

// RLState represents the full state for a reinforcement learning decision.
// This is a sequential decision: price → user reaction → next decision → retention.
type RLState struct {
	// ── User Features ────────────────────────────────────────────────────────
	UserTrust        float64 `json:"user_trust"`         // 0-100
	CancelRate       float64 `json:"cancel_rate"`        // 0-1
	InsuranceBuyRate float64 `json:"insurance_buy_rate"` // 0-1
	RiskScore        float64 `json:"risk_score"`         // 0-1
	AccountAgeDays   float64 `json:"account_age_days"`

	// ── Order Features ──────────────────────────────────────────────────────
	OrderValueCents int64   `json:"order_value_cents"`
	DeliveryRisk    float64 `json:"delivery_risk"` // 0-1
	Category        string  `json:"category"`

	// ── Session History (the RL differentiator) ─────────────────────────────
	SessionStep    int       `json:"session_step"`    // which step in the session (0,1,2...)
	PreviousOffers []float64 `json:"previous_offers"` // prices shown before (cents)
	LastAccepted   bool      `json:"last_accepted"`   // did user accept last offer?
	RefusalCount   int       `json:"refusal_count"`   // how many times refused
	Churned        bool      `json:"churned"`         // did user leave after refusal?

	// ── Context ────────────────────────────────────────────────────────────
	UrgencyScore float64 `json:"urgency_score"` // 0-1
	TimeOfDay    int     `json:"time_of_day"`   // 0-23
	LiveDemand   float64 `json:"live_demand"`   // 0-1
}

// StateKey is a discretized string representation of state for Q-table lookup.
type StateKey string

// Discretize converts a continuous RLState into a discrete StateKey for Q-table.
func (s *RLState) Discretize() StateKey {
	trustBin := discretize(s.UserTrust, 0, 100, 5)                        // 5 bins: 0-20,20-40,...
	cancelBin := discretize(s.CancelRate, 0, 1, 4)                        // 4 bins: 0-.25,.25-.5,...
	riskBin := discretize(s.RiskScore, 0, 1, 4)                           // 4 bins
	valueBin := discretizeFloat(float64(s.OrderValueCents), 0, 100000, 5) // 5 bins
	refusalBin := s.RefusalCount
	if refusalBin > 3 {
		refusalBin = 3
	}
	stepBin := s.SessionStep
	if stepBin > 3 {
		stepBin = 3
	}

	return StateKey(fmt.Sprintf("t%d_c%d_r%d_v%d_f%d_s%d_a%v",
		trustBin, cancelBin, riskBin, valueBin, refusalBin, stepBin, s.LastAccepted))
}

func discretize(val, min, max float64, bins int) int {
	if val <= min {
		return 0
	}
	if val >= max {
		return bins - 1
	}
	step := (max - min) / float64(bins)
	return int((val - min) / step)
}

func discretizeFloat(val, min, max float64, bins int) int {
	return discretize(val, min, max, bins)
}

// ── RL Action ────────────────────────────────────────────────────────────────────

// RLAction represents a pricing action the RL agent can take.
type RLAction struct {
	ID           uuid.UUID `json:"id"`
	PricePercent float64   `json:"price_percent"` // 1, 1.5, 2, 2.5, 3, 3.5, 4
	UXVariant    string    `json:"ux_variant"`    // standard, discount_badge, social_proof, urgency
	Label        string    `json:"label"`         // human-readable
}

// ActionIndex maps an action to an integer index for Q-table.
type ActionIndex int

// All possible price actions.
var RLPriceActions = []float64{1.0, 1.5, 2.0, 2.5, 3.0, 3.5, 4.0}

// All possible UX variants.
var RLUXVariants = []string{"standard", "discount_badge", "social_proof", "urgency"}

// BuildAllActions generates all action combinations.
func BuildAllActions() []RLAction {
	var actions []RLAction
	for _, pct := range RLPriceActions {
		for _, ux := range RLUXVariants {
			label := actionLabel(pct, ux)
			actions = append(actions, RLAction{
				ID:           uuid.New(),
				PricePercent: pct,
				UXVariant:    ux,
				Label:        label,
			})
		}
	}
	return actions
}

func actionLabel(pct float64, ux string) string {
	switch ux {
	case "discount_badge":
		return fmt.Sprintf("%.1f%% + discount badge", pct)
	case "social_proof":
		return fmt.Sprintf("%.1f%% + social proof", pct)
	case "urgency":
		return fmt.Sprintf("%.1f%% + urgency nudge", pct)
	default:
		return fmt.Sprintf("%.1f%% standard", pct)
	}
}

// ActionIndexMap maps (price_percent, ux_variant) → index.
var actionIndexMap map[string]ActionIndex
var indexToAction map[ActionIndex]RLAction

func init() {
	actions := BuildAllActions()
	actionIndexMap = make(map[string]ActionIndex, len(actions))
	indexToAction = make(map[ActionIndex]RLAction, len(actions))
	for i, a := range actions {
		key := fmt.Sprintf("%.1f_%s", a.PricePercent, a.UXVariant)
		actionIndexMap[key] = ActionIndex(i)
		indexToAction[ActionIndex(i)] = a
	}
}

// GetActionIndex returns the Q-table index for an action.
func GetActionIndex(a RLAction) ActionIndex {
	key := fmt.Sprintf("%.1f_%s", a.PricePercent, a.UXVariant)
	if idx, ok := actionIndexMap[key]; ok {
		return idx
	}
	return 0
}

// GetActionFromIndex returns the action for a Q-table index.
func GetActionFromIndex(idx ActionIndex) RLAction {
	if a, ok := indexToAction[idx]; ok {
		return a
	}
	return RLAction{PricePercent: 2.0, UXVariant: "standard", Label: "2.0% standard"}
}

// TotalActions returns the total number of possible actions.
func TotalActions() int {
	return len(RLPriceActions) * len(RLUXVariants) // 7 × 4 = 28
}

// ── RL Transition ────────────────────────────────────────────────────────────────

// RLTransition records a full (s, a, r, s') transition for offline training.
type RLTransition struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	OrderID   uuid.UUID `gorm:"type:uuid;not null" json:"order_id"`
	SessionID string    `gorm:"size:50;not null;index" json:"session_id"`

	// State
	StateKey  string `gorm:"size:100;not null;index" json:"state_key"`
	StateJSON string `gorm:"type:text" json:"state_json"`

	// Action
	ActionIndex  int     `gorm:"not null" json:"action_index"`
	PricePercent float64 `gorm:"type:numeric(5,2);not null" json:"price_percent"`
	UXVariant    string  `gorm:"size:30;not null" json:"ux_variant"`
	PriceCents   int64   `gorm:"not null" json:"price_cents"`

	// Reward components
	RewardRevenue   float64 `gorm:"type:numeric(12,2);not null;default:0" json:"reward_revenue"`
	RewardClaimCost float64 `gorm:"type:numeric(12,2);not null;default:0" json:"reward_claim_cost"`
	RewardChurn     float64 `gorm:"type:numeric(12,2);not null;default:0" json:"reward_churn"`
	RewardTotal     float64 `gorm:"type:numeric(12,2);not null;default:0" json:"reward_total"`

	// Outcome
	DidBuy   bool `gorm:"not null;default:false" json:"did_buy"`
	DidClaim bool `gorm:"not null;default:false" json:"did_claim"`
	DidChurn bool `gorm:"not null;default:false" json:"did_churn"`

	// Next state
	NextStateKey  string `gorm:"size:100;not null;default:''" json:"next_state_key"`
	NextStateJSON string `gorm:"type:text" json:"next_state_json"`

	// Training
	IsTerminal bool   `gorm:"not null;default:false" json:"is_terminal"`
	EpisodeID  string `gorm:"size:50;index" json:"episode_id"`

	CreatedAt time.Time `json:"created_at"`
}

func (RLTransition) TableName() string { return "rl_transitions" }

// ── RL Config ────────────────────────────────────────────────────────────────────

// RolloutPhase controls how much traffic the RL engine handles.
type RolloutPhase string

const (
	RolloutShadow   RolloutPhase = "shadow"    // decide but don't execute (log only)
	RolloutCanary5  RolloutPhase = "canary_5"  // 5% traffic
	RolloutCanary25 RolloutPhase = "canary_25" // 25% traffic
	RolloutCanary50 RolloutPhase = "canary_50" // 50% traffic
	RolloutFull     RolloutPhase = "full"      // 100% traffic
)

type RLConfig struct {
	ID                      uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Algorithm               string    `gorm:"size:20;not null;default:'q_learning'" json:"algorithm"`         // q_learning | policy_gradient
	LearningRate            float64   `gorm:"type:numeric(5,4);not null;default:0.1" json:"learning_rate"`    // α
	DiscountFactor          float64   `gorm:"type:numeric(5,4);not null;default:0.95" json:"discount_factor"` // γ
	Epsilon                 float64   `gorm:"type:numeric(5,4);not null;default:0.1" json:"epsilon"`          // exploration rate
	EpsilonDecay            float64   `gorm:"type:numeric(5,4);not null;default:0.995" json:"epsilon_decay"`  // ε decay per episode
	MinEpsilon              float64   `gorm:"type:numeric(5,4);not null;default:0.05" json:"min_epsilon"`
	ChurnPenalty            float64   `gorm:"type:numeric(8,2);not null;default:5.0" json:"churn_penalty"` // penalty for user leaving
	MinPricePercent         float64   `gorm:"type:numeric(5,2);not null;default:1" json:"min_price_percent"`
	MaxPricePercent         float64   `gorm:"type:numeric(5,2);not null;default:4" json:"max_price_percent"`
	ConversionDropThreshold float64   `gorm:"type:numeric(5,2);not null;default:0.08" json:"conversion_drop_threshold"` // 8% → kill switch
	SessionCooldownMinutes  int       `gorm:"not null;default:5" json:"session_cooldown_minutes"`
	MaxSessionSteps         int       `gorm:"not null;default:3" json:"max_session_steps"`            // max re-offers per session
	RolloutPhase            string    `gorm:"size:20;not null;default:'shadow'" json:"rollout_phase"` // shadow → canary → full
	KillSwitchActive        bool      `gorm:"not null;default:false" json:"kill_switch_active"`
	FallbackPricePercent    float64   `gorm:"type:numeric(5,2);not null;default:2" json:"fallback_price_percent"`
	IsActive                bool      `gorm:"not null;default:true" json:"is_active"`
	QTableJSON              string    `gorm:"type:text" json:"q_table_json,omitempty"` // serialized Q-table
	PolicyJSON              string    `gorm:"type:text" json:"policy_json,omitempty"`  // serialized policy (for policy gradient)
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

func (RLConfig) TableName() string { return "rl_configs" }

// ── RL Session ────────────────────────────────────────────────────────────────────

// RLSession tracks a user's sequential interaction with the pricing system.
type RLSession struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_rl_session_user_order,1" json:"user_id"`
	OrderID        uuid.UUID `gorm:"type:uuid;not null;uniqueIndex:idx_rl_session_user_order,2" json:"order_id"`
	EpisodeID      string    `gorm:"size:50;not null;index" json:"episode_id"`
	CurrentStep    int       `gorm:"not null;default:0" json:"current_step"`
	PreviousOffers string    `gorm:"type:text" json:"previous_offers"` // JSON array of offered prices
	RefusalCount   int       `gorm:"not null;default:0" json:"refusal_count"`
	LastActionIdx  int       `gorm:"not null;default:-1" json:"last_action_idx"`
	TotalReward    float64   `gorm:"type:numeric(12,2);not null;default:0" json:"total_reward"`
	IsComplete     bool      `gorm:"not null;default:false" json:"is_complete"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (RLSession) TableName() string { return "rl_sessions" }

// ── RL Selection Result ──────────────────────────────────────────────────────────

type RLSelectionResult struct {
	Action           RLAction `json:"action"`
	PriceCents       int64    `json:"price_cents"`
	PricePercent     float64  `json:"price_percent"`
	UXVariant        string   `json:"ux_variant"`
	StateKey         StateKey `json:"state_key"`
	Confidence       float64  `json:"confidence"`
	IsExploration    bool     `json:"is_exploration"`
	IsShadow         bool     `json:"is_shadow"` // true = decision logged but not executed
	SessionStep      int      `json:"session_step"`
	AnchorPrice      int64    `json:"anchor_price"`
	KillSwitchOn     bool     `json:"kill_switch_on"`
	RecommendedLabel string   `json:"recommended_label"`
}

// ── RL Dashboard ──────────────────────────────────────────────────────────────────

type RLDashboard struct {
	Config           RLConfig      `json:"config"`
	TotalTransitions int64         `json:"total_transitions"`
	TotalEpisodes    int64         `json:"total_episodes"`
	AvgRewardPerStep float64       `json:"avg_reward_per_step"`
	AvgEpisodeReward float64       `json:"avg_episode_reward"`
	AttachRate       float64       `json:"attach_rate"`
	ChurnRate        float64       `json:"churn_rate"`
	ClaimRate        float64       `json:"claim_rate"`
	ConversionTrend  string        `json:"conversion_trend"`
	KillSwitchActive bool          `json:"kill_switch_active"`
	RolloutPhase     RolloutPhase  `json:"rollout_phase"`
	TopActions       []ActionStats `json:"top_actions"`
}

type ActionStats struct {
	PricePercent   float64 `json:"price_percent"`
	UXVariant      string  `json:"ux_variant"`
	Count          int64   `json:"count"`
	AvgReward      float64 `json:"avg_reward"`
	ConversionRate float64 `json:"conversion_rate"`
}
