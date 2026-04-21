package pricing

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Shared State Encoder ──────────────────────────────────────────────────────────
//
// Converts CrossState → embedding → multi-head outputs.
// In production, this would be a neural network (ONNX/JSON).
// For now, we use a feature-based approach with Q-tables per head.

// StateEmbedding is a fixed-size feature vector derived from CrossState.
type StateEmbedding [16]float64

// EncodeState converts CrossState into a fixed-size embedding.
// This is the "shared encoder" — all heads use the same embedding.
func EncodeState(s *CrossState) StateEmbedding {
	var emb StateEmbedding

	// Normalize all features to [0, 1]
	emb[0] = s.UserTrust / 100.0
	emb[1] = s.CancelRate
	emb[2] = s.BuyRate
	emb[3] = s.AccountAgeDays / 365.0
	if emb[3] > 1 {
		emb[3] = 1
	}
	emb[4] = s.RiskScore
	emb[5] = s.DemandScore
	emb[6] = s.SupplyScore
	emb[7] = float64(s.SessionStep) / 3.0
	if emb[7] > 1 {
		emb[7] = 1
	}
	emb[8] = float64(s.RefusalCount) / 3.0
	if emb[8] > 1 {
		emb[8] = 1
	}
	emb[9] = s.DeliveryRisk
	emb[10] = s.UrgencyScore
	emb[11] = float64(s.ItemPriceCents) / 500000.0
	if emb[11] > 1 {
		emb[11] = 1
	}
	// Binary features
	if s.IsLiveHot {
		emb[12] = 1
	}
	if s.Device == "mobile" {
		emb[13] = 1
	}
	// Segment encoding
	segMap := map[string]float64{"vip": 1.0, "regular": 0.5, "new": 0.2, "at_risk": 0.0}
	emb[14] = segMap[s.UserSegment]
	// Previous price average
	if len(s.PreviousPrices) > 0 {
		sum := 0.0
		for _, p := range s.PreviousPrices {
			sum += p
		}
		emb[15] = sum / float64(len(s.PreviousPrices)) / 100000.0
		if emb[15] > 1 {
			emb[15] = 1
		}
	}

	return emb
}

// ── Multi-Head Q-Tables ────────────────────────────────────────────────────────────

// Each head has its own Q-table mapping (state_key × action_index) → Q-value.

// Pricing actions: 7 price levels
var crossPricingActions = []float64{1.0, 1.5, 2.0, 2.5, 3.0, 3.5, 4.0}

// Ranking actions: 5 boost levels
var crossRankingActions = []int{0, 25, 50, 75, 100}

// Recs actions: 4 strategies
var crossRecsActions = []string{"rl", "cf", "popular", "similar"}

var (
	crossPricingQ QTable
	crossRankingQ QTable
	crossRecsQ    QTable
	crossQMu      sync.RWMutex
)

func init() {
	crossPricingQ = make(QTable)
	crossRankingQ = make(QTable)
	crossRecsQ = make(QTable)
}

// ── Pricing Head ──────────────────────────────────────────────────────────────────

// PricingHeadPredict selects a price action using the pricing Q-table.
func PricingHeadPredict(stateKey CrossStateKey, epsilon float64) (int, float64, bool) {
	sk := StateKey(string(stateKey) + "_p")
	isExploration := rand.Float64() < epsilon

	if isExploration {
		idx := rand.Intn(len(crossPricingActions))
		return idx, crossPricingActions[idx], true
	}

	// Exploit: best Q-value
	crossQMu.RLock()
	defer crossQMu.RUnlock()

	if actions, ok := crossPricingQ[sk]; ok {
		bestIdx := 0
		bestVal := math.Inf(-1)
		for a, v := range actions {
			if v > bestVal {
				bestVal = v
				bestIdx = int(a)
			}
		}
		if bestVal != math.Inf(-1) {
			return bestIdx, crossPricingActions[bestIdx], false
		}
	}

	// Default: 2% (middle)
	return 3, 2.0, false
}

// ── Ranking Head ──────────────────────────────────────────────────────────────────

// RankingHeadPredict selects a boost score using the ranking Q-table.
func RankingHeadPredict(stateKey CrossStateKey, epsilon float64) (int, int, bool) {
	sk := StateKey(string(stateKey) + "_r")
	isExploration := rand.Float64() < epsilon

	if isExploration {
		idx := rand.Intn(len(crossRankingActions))
		return idx, crossRankingActions[idx], true
	}

	crossQMu.RLock()
	defer crossQMu.RUnlock()

	if actions, ok := crossRankingQ[sk]; ok {
		bestIdx := 0
		bestVal := math.Inf(-1)
		for a, v := range actions {
			if v > bestVal {
				bestVal = v
				bestIdx = int(a)
			}
		}
		if bestVal != math.Inf(-1) {
			return bestIdx, crossRankingActions[bestIdx], false
		}
	}

	// Default: boost 50
	return 2, 50, false
}

// ── Recs Head ──────────────────────────────────────────────────────────────────────

// RecsHeadPredict selects a recommendation strategy using the recs Q-table.
func RecsHeadPredict(stateKey CrossStateKey, epsilon float64) (int, string, bool) {
	sk := StateKey(string(stateKey) + "_c")
	isExploration := rand.Float64() < epsilon

	if isExploration {
		idx := rand.Intn(len(crossRecsActions))
		return idx, crossRecsActions[idx], true
	}

	crossQMu.RLock()
	defer crossQMu.RUnlock()

	if actions, ok := crossRecsQ[sk]; ok {
		bestIdx := 0
		bestVal := math.Inf(-1)
		for a, v := range actions {
			if v > bestVal {
				bestVal = v
				bestIdx = int(a)
			}
		}
		if bestVal != math.Inf(-1) {
			return bestIdx, crossRecsActions[bestIdx], false
		}
	}

	// Default: popular
	return 2, "popular", false
}

// ── Q-Table Updates ────────────────────────────────────────────────────────────────

func CrossPricingQUpdate(stateKey CrossStateKey, actionIdx int, reward, alpha, gamma float64, nextKey CrossStateKey) {
	sk := StateKey(string(stateKey) + "_p")
	nsk := StateKey(string(nextKey) + "_p")

	crossQMu.Lock()
	defer crossQMu.Unlock()

	if crossPricingQ[sk] == nil {
		crossPricingQ[sk] = make(map[ActionIndex]float64)
	}
	currentQ := crossPricingQ[sk][ActionIndex(actionIdx)]
	maxNext := 0.0
	if actions, ok := crossPricingQ[nsk]; ok {
		for _, v := range actions {
			if v > maxNext {
				maxNext = v
			}
		}
	}
	crossPricingQ[sk][ActionIndex(actionIdx)] = currentQ + alpha*(reward+gamma*maxNext-currentQ)
}

func CrossRankingQUpdate(stateKey CrossStateKey, actionIdx int, reward, alpha, gamma float64, nextKey CrossStateKey) {
	sk := StateKey(string(stateKey) + "_r")
	nsk := StateKey(string(nextKey) + "_r")

	crossQMu.Lock()
	defer crossQMu.Unlock()

	if crossRankingQ[sk] == nil {
		crossRankingQ[sk] = make(map[ActionIndex]float64)
	}
	currentQ := crossRankingQ[sk][ActionIndex(actionIdx)]
	maxNext := 0.0
	if actions, ok := crossRankingQ[nsk]; ok {
		for _, v := range actions {
			if v > maxNext {
				maxNext = v
			}
		}
	}
	crossRankingQ[sk][ActionIndex(actionIdx)] = currentQ + alpha*(reward+gamma*maxNext-currentQ)
}

func CrossRecsQUpdate(stateKey CrossStateKey, actionIdx int, reward, alpha, gamma float64, nextKey CrossStateKey) {
	sk := StateKey(string(stateKey) + "_c")
	nsk := StateKey(string(nextKey) + "_c")

	crossQMu.Lock()
	defer crossQMu.Unlock()

	if crossRecsQ[sk] == nil {
		crossRecsQ[sk] = make(map[ActionIndex]float64)
	}
	currentQ := crossRecsQ[sk][ActionIndex(actionIdx)]
	maxNext := 0.0
	if actions, ok := crossRecsQ[nsk]; ok {
		for _, v := range actions {
			if v > maxNext {
				maxNext = v
			}
		}
	}
	crossRecsQ[sk][ActionIndex(actionIdx)] = currentQ + alpha*(reward+gamma*maxNext-currentQ)
}

// ── State Builder ──────────────────────────────────────────────────────────────────

// BuildCrossState constructs CrossState from PricingContext + additional data.
func BuildCrossState(db *gorm.DB, ctx *PricingContext) *CrossState {
	state := &CrossState{
		UserTrust:      ctx.TrustScore,
		UserSegment:    classifyUserSegment(ctx.TrustScore, ctx.InsuranceBuyRate),
		CancelRate:     ctx.CancellationRate,
		BuyRate:        ctx.InsuranceBuyRate,
		AccountAgeDays: ctx.AccountAgeDays,
		RiskScore:      1.0 - ctx.TrustScore/100.0,
		DeliveryRisk:   ctx.DeliveryRiskScore,
		ItemPriceCents: ctx.OrderPriceCents,
		UrgencyScore:   ctx.UrgencyScore,
		DemandScore:    ctx.LiveDemand,
	}

	// Load session
	session := loadOrCreateSession(db, ctx.UserID, ctx.OrderID)
	if session != nil {
		state.SessionStep = session.CurrentStep
		state.RefusalCount = session.RefusalCount
		if session.PreviousOffers != "" && session.PreviousOffers != "[]" {
			var offers []float64
			json.Unmarshal([]byte(session.PreviousOffers), &offers)
			state.PreviousPrices = offers
		}
	}

	return state
}

func classifyUserSegment(trust, buyRate float64) string {
	if trust > 70 && buyRate > 0.5 {
		return "vip"
	} else if trust > 40 {
		return "regular"
	} else if trust < 20 || buyRate < 0.1 {
		return "at_risk"
	}
	return "new"
}

// ── Fallback Functions (per-head) ──────────────────────────────────────────────────

// FallbackPricing returns a safe price using bandit or rules.
func FallbackPricing(db *gorm.DB, ctx *PricingContext) (int64, float64, string) {
	// Try bandit first
	result, err := SelectPrice(db, ctx)
	if err == nil && result != nil {
		return result.PriceCents, result.PricePercent, "bandit"
	}
	// Then rules
	ruleResult := CalculateRuleBasedPrice(ctx, nil)
	return ruleResult.PriceCents, ruleResult.PricePercent, "rules"
}

// FallbackRanking returns a heuristic boost score.
func FallbackRanking(state *CrossState) (int, string) {
	// Heuristic: trust > 60 → boost 70, trust < 30 → boost 30, else 50
	if state.UserTrust > 60 {
		return 70, "heuristic"
	} else if state.UserTrust < 30 {
		return 30, "heuristic"
	}
	return 50, "heuristic"
}

// FallbackRecs returns popular items as fallback.
func FallbackRecs(db *gorm.DB, category string, limit int) ([]string, string) {
	// In production: query popular items from DB
	// For now: return empty with strategy marker
	return []string{}, "popular"
}

// ── Cross Q-Table Serialization ────────────────────────────────────────────────────

type crossQTablesJSON struct {
	Pricing map[string]map[string]float64 `json:"pricing"`
	Ranking map[string]map[string]float64 `json:"ranking"`
	Recs    map[string]map[string]float64 `json:"recs"`
}

func SerializeCrossQTables() (string, error) {
	crossQMu.RLock()
	defer crossQMu.RUnlock()

	data := crossQTablesJSON{
		Pricing: qTableToJSON(crossPricingQ),
		Ranking: qTableToJSON(crossRankingQ),
		Recs:    qTableToJSON(crossRecsQ),
	}
	bytes, err := json.Marshal(data)
	return string(bytes), err
}

func DeserializeCrossQTables(jsonStr string) error {
	var data crossQTablesJSON
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return err
	}

	crossQMu.Lock()
	defer crossQMu.Unlock()

	crossPricingQ = jsonToQTable(data.Pricing)
	crossRankingQ = jsonToQTable(data.Ranking)
	crossRecsQ = jsonToQTable(data.Recs)
	return nil
}

func qTableToJSON(table QTable) map[string]map[string]float64 {
	result := make(map[string]map[string]float64, len(table))
	for state, actions := range table {
		values := make(map[string]float64, len(actions))
		for a, v := range actions {
			values[fmt.Sprintf("%d", a)] = v
		}
		result[string(state)] = values
	}
	return result
}

func jsonToQTable(data map[string]map[string]float64) QTable {
	table := make(QTable, len(data))
	for state, values := range data {
		actions := make(map[ActionIndex]float64, len(values))
		for k, v := range values {
			var idx int
			fmt.Sscanf(k, "%d", &idx)
			actions[ActionIndex(idx)] = v
		}
		table[StateKey(state)] = actions
	}
	return table
}

// Ensure imports used
var _ = uuid.New
var _ = math.Abs
