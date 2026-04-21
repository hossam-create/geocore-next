package policy

import (
	"log/slog"

	"github.com/geocore-next/backend/internal/cloudos/resources"
)

// Engine is the unified policy engine combining SLO, budget, and security gates.
type Engine struct {
	SLO      *SLOEngine
	Budget   *BudgetEngine
	Security *SecurityEngine
}

// NewEngine creates a policy engine with all safety gates.
func NewEngine() *Engine {
	return &Engine{
		SLO:      NewSLOEngine(),
		Budget:   NewBudgetEngine(),
		Security: NewSecurityEngine(),
	}
}

// Allowed checks a proposal through ALL safety gates.
// A proposal must pass EVERY gate to be approved.
func (e *Engine) Allowed(p resources.Proposal) bool {
	if !e.SLO.Check(p) {
		slog.Warn("cloudos: blocked by SLO engine", "resource", p.Resource)
		return false
	}
	if !e.Budget.Check(p) {
		slog.Warn("cloudos: blocked by budget engine", "resource", p.Resource)
		return false
	}
	if !e.Security.Check(p) {
		slog.Warn("cloudos: blocked by security engine", "resource", p.Resource)
		return false
	}
	return true
}
