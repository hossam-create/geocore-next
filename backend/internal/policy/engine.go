// Package policy provides an OPA-like policy engine for service authorization.
// Phase 5 — Zero Trust Security Model.
package policy

import (
	"context"
	"log/slog"
	"sync"
)

// Effect represents a policy decision effect.
type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

// Action represents what a service wants to do.
type Action struct {
	Service   string `json:"service"`
	Resource  string `json:"resource"`
	Operation string `json:"operation"` // GET, POST, PUBLISH, SUBSCRIBE
}

// Policy is a single allow/deny rule.
type Policy struct {
	ID        string `json:"id"`
	Service   string `json:"service"`   // "*" for all
	Resource  string `json:"resource"`  // "*" for all
	Operation string `json:"operation"` // "*" for all
	Effect    Effect `json:"effect"`
	Priority  int    `json:"priority"` // higher = evaluated first
}

// Engine evaluates policies to make authorization decisions.
type Engine struct {
	mu      sync.RWMutex
	policies []Policy
}

// NewEngine creates a policy engine with default deny-all.
func NewEngine() *Engine {
	return &Engine{
		policies: []Policy{
			{ID: "default-deny", Service: "*", Resource: "*", Operation: "*", Effect: EffectDeny, Priority: 0},
		},
	}
}

// AddPolicy inserts a new policy rule.
func (e *Engine) AddPolicy(p Policy) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.policies = append(e.policies, p)
	slog.Info("policy: added rule", "id", p.ID, "service", p.Service, "effect", string(p.Effect))
}

// Evaluate checks if an action is allowed by the policy set.
// Returns the highest-priority matching policy's effect.
func (e *Engine) Evaluate(ctx context.Context, a Action) Effect {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var bestMatch *Policy
	for i := range e.policies {
		p := &e.policies[i]
		if !matchRule(p.Service, a.Service) {
			continue
		}
		if !matchRule(p.Resource, a.Resource) {
			continue
		}
		if !matchRule(p.Operation, a.Operation) {
			continue
		}
		if bestMatch == nil || p.Priority > bestMatch.Priority {
			bestMatch = p
		}
	}

	if bestMatch == nil {
		return EffectDeny
	}

	slog.Debug("policy: evaluated",
		"service", a.Service,
		"resource", a.Resource,
		"operation", a.Operation,
		"effect", string(bestMatch.Effect),
		"rule", bestMatch.ID,
	)
	return bestMatch.Effect
}

func matchRule(rule, value string) bool {
	return rule == "*" || rule == value
}
