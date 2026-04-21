package runtime

import (
	"context"
	"log/slog"
	"time"

	"github.com/geocore-next/backend/internal/cloudos/resources"
)

// RollbackEvent records a rollback action.
type RollbackEvent struct {
	Resource    string    `json:"resource"`
	Action      string    `json:"action"`
	Reason      string    `json:"reason"`
	TriggeredAt time.Time `json:"triggered_at"`
}

// RollbackEngine handles automatic rollback of failed changes.
type RollbackEngine struct {
	history []RollbackEvent
}

// NewRollbackEngine creates a rollback engine.
func NewRollbackEngine() *RollbackEngine {
	return &RollbackEngine{}
}

// Rollback reverts a proposal that failed post-deploy verification.
func (r *RollbackEngine) Rollback(ctx context.Context, p resources.Proposal) error {
	slog.Error("cloudos: ROLLING BACK",
		"resource", p.Resource, "action", p.Action, "target", p.Target)

	r.history = append(r.history, RollbackEvent{
		Resource:    p.Resource,
		Action:      p.Action,
		Reason:      "post-deploy verification failed",
		TriggeredAt: time.Now().UTC(),
	})

	// Production: kubectl rollout undo, restore previous config
	return nil
}

// History returns recent rollback events.
func (r *RollbackEngine) History(n int) []RollbackEvent {
	if n > len(r.history) {
		n = len(r.history)
	}
	result := make([]RollbackEvent, n)
	copy(result, r.history[len(r.history)-n:])
	return result
}
