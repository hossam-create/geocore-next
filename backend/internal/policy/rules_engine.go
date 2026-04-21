package policy

import (
	"log/slog"
)

// DataRule is a single data-level policy rule (e.g. "amount > 10000 → deny").
// Complements the service-level Engine in engine.go.
type DataRule struct {
	Name     string                       `json:"name"`
	Field    string                       `json:"field"`
	Operator string                       `json:"operator"` // gt, lt, eq, gte, lte, contains
	Value    any                          `json:"value"`
	OnDeny   string                       `json:"on_deny"` // block, challenge, log
	Reason   string                       `json:"reason"`
	Custom   func(input map[string]any) bool `json:"-"` // custom evaluator
}

// DataDecision is the result of evaluating data-level rules.
type DataDecision struct {
	Allow  bool   `json:"allow"`
	Reason string `json:"reason,omitempty"`
	Rule   string `json:"rule,omitempty"`
}

// RulesEngine evaluates data-level policy rules against input data (OPA-like).
type RulesEngine struct {
	rules []DataRule
}

// NewRulesEngine creates a rules engine with the given data rules.
func NewRulesEngine(rules []DataRule) *RulesEngine {
	return &RulesEngine{rules: rules}
}

// Evaluate checks all data rules against the input.
// Returns the first denial, or allow if all rules pass.
func (re *RulesEngine) Evaluate(input map[string]any) DataDecision {
	for _, rule := range re.rules {
		if !re.evaluateRule(rule, input) {
			slog.Warn("policy: data rule denied",
				"rule", rule.Name,
				"field", rule.Field,
				"reason", rule.Reason,
			)
			return DataDecision{
				Allow:  false,
				Reason: rule.Reason,
				Rule:   rule.Name,
			}
		}
	}
	return DataDecision{Allow: true}
}

func (re *RulesEngine) evaluateRule(rule DataRule, input map[string]any) bool {
	if rule.Custom != nil {
		return rule.Custom(input)
	}

	val, ok := input[rule.Field]
	if !ok {
		return true // field not present = rule doesn't apply
	}

	switch rule.Operator {
	case "gt":
		return compareNumbers(val, rule.Value) > 0
	case "lt":
		return compareNumbers(val, rule.Value) < 0
	case "gte":
		return compareNumbers(val, rule.Value) >= 0
	case "lte":
		return compareNumbers(val, rule.Value) <= 0
	case "eq":
		return val == rule.Value
	case "contains":
		s, ok := val.(string)
		if !ok {
			return false
		}
		target, ok := rule.Value.(string)
		if !ok {
			return false
		}
		return s == target
	default:
		return true
	}
}

// AddRule appends a data rule to the engine.
func (re *RulesEngine) AddRule(rule DataRule) {
	re.rules = append(re.rules, rule)
}

// DataRules returns all registered data rules.
func (re *RulesEngine) DataRules() []DataRule {
	return re.rules
}

func compareNumbers(a, b any) int {
	af := toFloat64(a)
	bf := toFloat64(b)
	switch {
	case af < bf:
		return -1
	case af > bf:
		return 1
	default:
		return 0
	}
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	default:
		return 0
	}
}

// PredefinedDataRules returns common Geocore data-level policy rules.
func PredefinedDataRules() []DataRule {
	return []DataRule{
		{
			Name: "max_transaction_amount",
			Field: "amount",
			Operator: "gt",
			Value: float64(10000),
			OnDeny: "block",
			Reason: "transaction exceeds maximum allowed amount",
		},
		{
			Name: "max_wallet_transfer",
			Field: "transfer_amount",
			Operator: "gt",
			Value: float64(5000),
			OnDeny: "challenge",
			Reason: "large wallet transfer requires additional verification",
		},
		{
			Name: "block_sanctioned_country",
			Field: "country",
			Operator: "eq",
			Value: "XX",
			OnDeny: "block",
			Reason: "country is on sanctions list",
		},
	}
}
