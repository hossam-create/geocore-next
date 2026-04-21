package pricing

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
)

// ── Lightweight Gradient Boosted Trees Model ─────────────────────────────────────
//
// The model is stored as JSON containing an ensemble of regression trees.
// Each tree outputs a log-odds contribution; the final prediction is:
//
//	probability = sigmoid(sum_of_leaves + bias)
//	optimal_price = base_price * (1 + price_multiplier)
//
// This is intentionally lightweight — no ONNX runtime, no CGo dependencies.
// Trees are evaluated in pure Go for zero-latency inference.

// TreeModel represents a single decision tree.
type TreeModel struct {
	// Internal nodes: if feature[idx] <= threshold → go left, else go right
	ChildrenLeft  []int     `json:"children_left"`
	ChildrenRight []int     `json:"children_right"`
	FeatureIndex  []int     `json:"feature_index"`
	Threshold     []float64 `json:"threshold"`
	// Leaf values
	Values []float64 `json:"values"`
}

// GBTModel is an ensemble of trees with metadata.
type GBTModel struct {
	Version       string       `json:"version"`
	NFeatures     int          `json:"n_features"`
	Bias          float64      `json:"bias"`            // initial log-odds
	LearningRate  float64      `json:"learning_rate"`   // shrinkage
	Trees         []TreeModel  `json:"trees"`
	PriceScale    float64      `json:"price_scale"`     // multiplier scale
	PriceBias     float64      `json:"price_bias"`      // base price offset
	TrainedAt     string       `json:"trained_at"`
	AUROC         float64      `json:"auroc"`
	FeatureNames  []string     `json:"feature_names"`
}

// ModelStore holds the loaded model in memory.
var (
	loadedModel *GBTModel
	modelMu     sync.RWMutex
)

// LoadModelFromJSON parses a JSON model definition.
func LoadModelFromJSON(jsonData []byte) error {
	var model GBTModel
	if err := json.Unmarshal(jsonData, &model); err != nil {
		return fmt.Errorf("model parse failed: %w", err)
	}
	if len(model.Trees) == 0 {
		return fmt.Errorf("model has no trees")
	}

	modelMu.Lock()
	loadedModel = &model
	modelMu.Unlock()

	return nil
}

// GetLoadedModel returns the currently loaded model (nil if none).
func GetLoadedModel() *GBTModel {
	modelMu.RLock()
	defer modelMu.RUnlock()
	return loadedModel
}

// Predict uses the loaded GBT model to predict buy probability and optimal price.
func Predict(features []float64) (buyProb float64, optimalPriceMultiplier float64, confidence float64, err error) {
	model := GetLoadedModel()
	if model == nil {
		return 0, 0, 0, fmt.Errorf("no model loaded")
	}

	if len(features) != model.NFeatures {
		return 0, 0, 0, fmt.Errorf("feature dimension mismatch: got %d, expected %d",
			len(features), model.NFeatures)
	}

	// Accumulate log-odds from each tree
	logOdds := model.Bias
	for _, tree := range model.Trees {
		leafValue := traverseTree(&tree, features)
		logOdds += model.LearningRate * leafValue
	}

	// Sigmoid to get probability
	buyProb = sigmoid(logOdds)

	// Price multiplier: higher buy probability → can charge more
	// optimal_price = base * (1 + price_multiplier)
	// price_multiplier is derived from a second set of predictions
	priceLogOdds := 0.0
	priceTrees := len(model.Trees) / 2 // second half for price prediction
	if priceTrees > 0 {
		priceBias := model.PriceBias
		priceLogOdds = priceBias
		for i := len(model.Trees) - priceTrees; i < len(model.Trees); i++ {
			leafValue := traverseTree(&model.Trees[i], features)
			priceLogOdds += model.LearningRate * leafValue
		}
		optimalPriceMultiplier = sigmoid(priceLogOdds) * model.PriceScale
	} else {
		// Fallback: derive from buy probability
		// Higher buy prob → user values protection → can charge slightly more
		optimalPriceMultiplier = buyProb * 0.02 // 0-2% additional
	}

	// Confidence: based on model AUROC and feature completeness
	confidence = model.AUROC
	if confidence < 0.5 {
		confidence = 0.5
	}

	return buyProb, optimalPriceMultiplier, confidence, nil
}

// traverseTree walks a single decision tree and returns the leaf value.
func traverseTree(tree *TreeModel, features []float64) float64 {
	node := 0 // root

	for {
		// Check if leaf node: children_left == -1 means leaf
		if tree.ChildrenLeft[node] == -1 {
			return tree.Values[node]
		}

		featureIdx := tree.FeatureIndex[node]
		if featureIdx >= len(features) {
			// Missing feature → go right (default path)
			node = tree.ChildrenRight[node]
			continue
		}

		if features[featureIdx] <= tree.Threshold[node] {
			node = tree.ChildrenLeft[node]
		} else {
			node = tree.ChildrenRight[node]
		}
	}
}

// sigmoid converts log-odds to probability.
func sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

// ── Default Seed Model ────────────────────────────────────────────────────────────
// A minimal bootstrap model with 3 trees for immediate use.
// This gets replaced by a trained model via admin API or config.

func DefaultSeedModel() *GBTModel {
	return &GBTModel{
		Version:      "seed-v1",
		NFeatures:    14,
		Bias:         -0.4, // base log-odds (slight bias toward not buying)
		LearningRate: 0.1,
		PriceScale:   0.03, // up to 3% additional
		PriceBias:    -0.2,
		AUROC:        0.65, // modest accuracy for seed model
		FeatureNames: []string{
			"order_value", "trust_score", "cancel_rate", "insurance_history",
			"abuse_count", "traveler_rating", "delay_prob", "route_risk",
			"time_of_day", "urgency", "buy_rate", "price_sensitivity",
			"account_age", "live_demand",
		},
		Trees: []TreeModel{
			// Tree 1: trust + cancel_rate → buy probability
			{
				ChildrenLeft:  []int{1, -1, -1},
				ChildrenRight: []int{2, -1, -1},
				FeatureIndex:  []int{1, 2, -1}, // trust_score, cancel_rate
				Threshold:     []float64{0.5, 0.3, 0},
				Values:        []float64{0, 0.8, -0.5}, // leaf values
			},
			// Tree 2: urgency + buy_rate
			{
				ChildrenLeft:  []int{1, -1, -1},
				ChildrenRight: []int{2, -1, -1},
				FeatureIndex:  []int{9, 10, -1}, // urgency, buy_rate
				Threshold:     []float64{0.5, 0.5, 0},
				Values:        []float64{0, 0.6, -0.3},
			},
			// Tree 3: abuse + price_sensitivity
			{
				ChildrenLeft:  []int{1, -1, -1},
				ChildrenRight: []int{2, -1, -1},
				FeatureIndex:  []int{4, 11, -1}, // abuse_count, price_sensitivity
				Threshold:     []float64{1.5, 0.6, 0},
				Values:        []float64{0, -0.4, 0.3},
			},
		},
	}
}

// InitSeedModel loads the default seed model into memory.
func InitSeedModel() {
	seed := DefaultSeedModel()
	modelMu.Lock()
	if loadedModel == nil {
		loadedModel = seed
	}
	modelMu.Unlock()
}
