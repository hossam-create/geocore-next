package pricing

import (
	"encoding/json"
	"math"
	"math/rand"
	"sync"
)

// ── Q-Learning Engine ────────────────────────────────────────────────────────────
//
// Q(s,a) ← Q(s,a) + α[r + γ·max_a' Q(s',a') - Q(s,a)]
//
// State space: discretized RLState → StateKey
// Action space: 28 actions (7 prices × 4 UX variants)
// Learning: online Q-update after each transition
// Export: serialize Q-table as JSON for persistence

// QTable maps StateKey → ActionIndex → Q-value.
type QTable map[StateKey]map[ActionIndex]float64

var (
	qTable     QTable
	qTableMu   sync.RWMutex
)

func init() {
	qTable = make(QTable)
}

// GetQValue returns the Q-value for a state-action pair.
func GetQValue(state StateKey, action ActionIndex) float64 {
	qTableMu.RLock()
	defer qTableMu.RUnlock()

	if actions, ok := qTable[state]; ok {
		if val, ok := actions[action]; ok {
			return val
		}
	}
	return 0.0 // default Q-value for unseen state-action pairs
}

// SetQValue sets the Q-value for a state-action pair.
func SetQValue(state StateKey, action ActionIndex, value float64) {
	qTableMu.Lock()
	defer qTableMu.Unlock()

	if qTable[state] == nil {
		qTable[state] = make(map[ActionIndex]float64)
	}
	qTable[state][action] = value
}

// QUpdate performs a single Q-learning update.
// Q(s,a) ← Q(s,a) + α[r + γ·max_a' Q(s',a') - Q(s,a)]
func QUpdate(state StateKey, action ActionIndex, reward float64, nextState StateKey, alpha, gamma float64) {
	currentQ := GetQValue(state, action)
	maxNextQ := MaxQ(nextState)

	newQ := currentQ + alpha*(reward+gamma*maxNextQ-currentQ)
	SetQValue(state, action, newQ)
}

// MaxQ returns the maximum Q-value for a state.
func MaxQ(state StateKey) float64 {
	qTableMu.RLock()
	defer qTableMu.RUnlock()

	if actions, ok := qTable[state]; ok {
		maxVal := math.Inf(-1)
		for _, v := range actions {
			if v > maxVal {
				maxVal = v
			}
		}
		if maxVal != math.Inf(-1) {
			return maxVal
		}
	}
	return 0.0
}

// ArgMaxQ returns the action with the highest Q-value for a state.
func ArgMaxQ(state StateKey) ActionIndex {
	qTableMu.RLock()
	defer qTableMu.RUnlock()

	if actions, ok := qTable[state]; ok {
		bestAction := ActionIndex(0)
		bestVal := math.Inf(-1)
		for a, v := range actions {
			if v > bestVal {
				bestVal = v
				bestAction = a
			}
		}
		if bestVal != math.Inf(-1) {
			return bestAction
		}
	}

	// Default: pick a reasonable action (2% standard)
	return GetActionIndex(RLAction{PricePercent: 2.0, UXVariant: "standard"})
}

// SelectActionQ selects an action using ε-greedy policy.
func SelectActionQ(state StateKey, epsilon float64) (ActionIndex, bool) {
	isExploration := rand.Float64() < epsilon

	if isExploration {
		// Explore: random action
		totalActions := TotalActions()
		return ActionIndex(rand.Intn(totalActions)), true
	}

	// Exploit: best known action
	return ArgMaxQ(state), false
}

// ── Q-Table Serialization ────────────────────────────────────────────────────────

// SerializeQTable converts the Q-table to JSON for persistence.
func SerializeQTable() (string, error) {
	qTableMu.RLock()
	defer qTableMu.RUnlock()

	// Convert to serializable format
	type QEntry struct {
		State  string             `json:"state"`
		Values map[string]float64 `json:"values"`
	}

	entries := make([]QEntry, 0, len(qTable))
	for state, actions := range qTable {
		values := make(map[string]float64, len(actions))
		for action, value := range actions {
			values[string(rune(action+'0'))] = value
		}
		entries = append(entries, QEntry{
			State:  string(state),
			Values: values,
		})
	}

	data, err := json.Marshal(entries)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// DeserializeQTable loads a Q-table from JSON.
func DeserializeQTable(jsonStr string) error {
	type QEntry struct {
		State  string             `json:"state"`
		Values map[string]float64 `json:"values"`
	}

	var entries []QEntry
	if err := json.Unmarshal([]byte(jsonStr), &entries); err != nil {
		return err
	}

	qTableMu.Lock()
	defer qTableMu.Unlock()

	newTable := make(QTable, len(entries))
	for _, entry := range entries {
		actions := make(map[ActionIndex]float64, len(entry.Values))
		for k, v := range entry.Values {
			idx := ActionIndex([]rune(k)[0] - '0')
			actions[idx] = v
		}
		newTable[StateKey(entry.State)] = actions
	}
	qTable = newTable

	return nil
}

// ── Policy Gradient (placeholder for future) ─────────────────────────────────────

// PolicyNetwork represents a policy gradient model.
// Currently a placeholder — will be replaced by trained model.
type PolicyNetwork struct {
	WeightsJSON string `json:"weights_json"`
	Version     string `json:"version"`
}

// PredictPolicy is a placeholder for policy gradient inference.
// Falls back to Q-learning if no policy is loaded.
func PredictPolicy(state *RLState) (ActionIndex, float64) {
	// For now, delegate to Q-learning
	stateKey := state.Discretize()
	action := ArgMaxQ(stateKey)
	return action, GetQValue(stateKey, action)
}

// ── Q-Table Stats ──────────────────────────────────────────────────────────────

// QTableStats returns statistics about the Q-table.
func QTableStats() map[string]interface{} {
	qTableMu.RLock()
	defer qTableMu.RUnlock()

	totalStates := len(qTable)
	totalEntries := 0
	nonZeroEntries := 0

	for _, actions := range qTable {
		totalEntries += len(actions)
		for _, v := range actions {
			if v != 0 {
				nonZeroEntries++
			}
		}
	}

	return map[string]interface{}{
		"total_states":     totalStates,
		"total_entries":   totalEntries,
		"non_zero_entries": nonZeroEntries,
	}
}

// Ensure math import used
var _ = math.Abs
