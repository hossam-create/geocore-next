package warroom

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/geocore-next/backend/internal/aiops"
	"github.com/google/uuid"
)

// newID generates a UUID for action and transition IDs.
func newID() string { return uuid.New().String() }

// Controller is the War Room brain.
// It evaluates system signals every 30s, transitions the state machine,
// and queues recommended actions for operator review.
type Controller struct {
	mu        sync.RWMutex
	state     SystemState
	history   []StateTransition    // last 50 transitions
	pending   []PendingAction      // all pending/approved/rejected actions
	dash      *dashboardBuilder
}

func newController() *Controller {
	return &Controller{
		state: StateNormal,
		dash:  newDashboardBuilder(),
	}
}

// Start begins the background evaluation loop (30s interval).
func (c *Controller) Start(ctx context.Context) {
	slog.Info("warroom: controller started")
	go c.loop(ctx)
}

func (c *Controller) loop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.Evaluate()
		case <-ctx.Done():
			return
		}
	}
}

// Evaluate reads live signals and transitions system state accordingly.
func (c *Controller) Evaluate() {
	openIncidents := aiops.GetOpenCount()

	var newState SystemState
	var reason string

	switch {
	case openIncidents == 0:
		newState = StateNormal
		reason = "no open incidents, system healthy"
	case openIncidents <= 2:
		newState = StateDegraded
		reason = "1–2 open incidents detected"
	default:
		newState = StateIncidentActive
		reason = "3+ open incidents — multiple services affected"
	}

	c.mu.Lock()
	prevState := c.state

	if newState.severity() > prevState.severity() || newState == StateNormal && prevState != StateNormal {
		c.transition(prevState, newState, reason)
		c.queueActionsForState(newState)
	}
	c.mu.Unlock()

	if prevState != newState {
		slog.Warn("warroom: state transition",
			"from", prevState, "to", newState, "reason", reason,
			"open_incidents", openIncidents,
		)
	}
}

// transition records a state change (caller must hold c.mu).
func (c *Controller) transition(from, to SystemState, reason string) {
	c.state = to
	t := StateTransition{From: from, To: to, Reason: reason, At: time.Now()}
	c.history = append([]StateTransition{t}, c.history...)
	if len(c.history) > 50 {
		c.history = c.history[:50]
	}
}

// queueActionsForState creates recommended actions for the new state (caller must hold c.mu).
func (c *Controller) queueActionsForState(state SystemState) {
	switch state {
	case StateDegraded:
		c.addAction(newAction(ActionEnableDegradedMode, nil))
		c.addAction(newAction(ActionPauseFeatureRollouts, nil))
	case StateIncidentActive:
		c.addAction(newAction(ActionEnableDegradedMode, nil))
		c.addAction(newAction(ActionScaleConsumers, map[string]string{"replicas": "3"}))
		c.addAction(newAction(ActionEnableReadReplica, nil))
		c.addAction(newAction(ActionDisableNonCritical, nil))
	case StateLockdown:
		c.addAction(newAction(ActionLockdown, nil))
	case StateNormal:
		c.addAction(newAction(ActionDisableDegradedMode, nil))
	}
}

// addAction appends an action; auto-approve low-risk ones immediately (caller must hold c.mu).
func (c *Controller) addAction(a PendingAction) {
	if a.AutoApprove {
		a.Status = ActionApproved
		now := time.Now()
		a.ExecutedAt = &now
		slog.Info("warroom: auto-approved action", "type", a.Type, "label", a.Label)
	}
	c.pending = append([]PendingAction{a}, c.pending...)
	if len(c.pending) > 200 {
		c.pending = c.pending[:200]
	}
}

// ManualTransition allows operators to force a state change.
func (c *Controller) ManualTransition(to SystemState, reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	from := c.state
	c.transition(from, to, "manual override: "+reason)
	c.queueActionsForState(to)
	slog.Warn("warroom: manual state transition", "from", from, "to", to, "reason", reason)
}

// ApproveAction marks a pending action as approved.
func (c *Controller) ApproveAction(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.pending {
		if c.pending[i].ID == id && c.pending[i].Status == ActionPending {
			c.pending[i].Status = ActionApproved
			now := time.Now()
			c.pending[i].ExecutedAt = &now
			slog.Info("warroom: action approved", "id", id, "type", c.pending[i].Type)
			return true
		}
	}
	return false
}

// RejectAction marks a pending action as rejected.
func (c *Controller) RejectAction(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range c.pending {
		if c.pending[i].ID == id && c.pending[i].Status == ActionPending {
			c.pending[i].Status = ActionRejected
			slog.Info("warroom: action rejected", "id", id)
			return true
		}
	}
	return false
}

func (c *Controller) State() SystemState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

func (c *Controller) Dashboard() DashboardView {
	c.mu.RLock()
	state := c.state
	history := make([]StateTransition, len(c.history))
	copy(history, c.history)
	pending := make([]PendingAction, len(c.pending))
	copy(pending, c.pending)
	c.mu.RUnlock()

	return c.dash.Build(state, history, pending)
}
