package warroom

import "time"

// ActionType identifies a war room action.
type ActionType string

const (
	ActionEnableDegradedMode   ActionType = "enable_degraded_mode"
	ActionDisableDegradedMode  ActionType = "disable_degraded_mode"
	ActionScaleConsumers       ActionType = "scale_consumers"
	ActionEnableReadReplica    ActionType = "enable_read_replica"
	ActionDisableNonCritical   ActionType = "disable_noncritical_apis"
	ActionPauseFeatureRollouts ActionType = "pause_feature_rollouts"
	ActionTriggerRollback      ActionType = "trigger_rollback"
	ActionLockdown             ActionType = "full_lockdown"
	ActionRecover              ActionType = "recover_to_normal"
)

// ActionStatus tracks the lifecycle of a pending action.
type ActionStatus string

const (
	ActionPending  ActionStatus = "pending"
	ActionApproved ActionStatus = "approved"
	ActionRejected ActionStatus = "rejected"
	ActionExecuted ActionStatus = "executed"
)

// PendingAction is a recommended or queued action waiting for operator approval.
type PendingAction struct {
	ID          string            `json:"id"`
	Type        ActionType        `json:"type"`
	Label       string            `json:"label"`
	Description string            `json:"description"`
	RiskLevel   string            `json:"risk_level"` // low | medium | high | critical
	AutoApprove bool              `json:"auto_approve"`
	Status      ActionStatus      `json:"status"`
	Params      map[string]string `json:"params,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	ExecutedAt  *time.Time        `json:"executed_at,omitempty"`
}

// actionCatalog defines the metadata for each action type.
var actionCatalog = map[ActionType]struct {
	Label       string
	Description string
	Risk        string
	Auto        bool
}{
	ActionEnableDegradedMode: {
		Label:       "Enable Degraded Mode",
		Description: "Serve cached/simplified responses for non-critical endpoints. Reduces DB pressure.",
		Risk:        "low",
		Auto:        true,
	},
	ActionDisableDegradedMode: {
		Label:       "Disable Degraded Mode",
		Description: "Restore full API responses after incident resolution.",
		Risk:        "low",
		Auto:        true,
	},
	ActionScaleConsumers: {
		Label:       "Scale Kafka Consumers",
		Description: "Increase consumer replicas: kubectl scale deployment wallet-service --replicas=3",
		Risk:        "low",
		Auto:        false,
	},
	ActionEnableReadReplica: {
		Label:       "Enable DB Read Replica",
		Description: "Route GET requests to read replica to offload primary DB.",
		Risk:        "medium",
		Auto:        false,
	},
	ActionDisableNonCritical: {
		Label:       "Disable Non-Critical APIs",
		Description: "Pause /reports, /analytics, /recommendations endpoints to preserve capacity.",
		Risk:        "medium",
		Auto:        false,
	},
	ActionPauseFeatureRollouts: {
		Label:       "Pause Feature Flag Rollouts",
		Description: "Freeze all feature flag % rollouts to stable state during incident.",
		Risk:        "low",
		Auto:        true,
	},
	ActionTriggerRollback: {
		Label:       "Trigger Rollback",
		Description: "kubectl rollout undo deployment/api — requires GitOps approval.",
		Risk:        "critical",
		Auto:        false,
	},
	ActionLockdown: {
		Label:       "Full System Lockdown",
		Description: "Disable all write operations. Read-only mode. Requires explicit operator approval.",
		Risk:        "critical",
		Auto:        false,
	},
	ActionRecover: {
		Label:       "Exit to Normal",
		Description: "Transition system state back to NORMAL. Disable degraded mode.",
		Risk:        "low",
		Auto:        false,
	},
}

func newAction(t ActionType, params map[string]string) PendingAction {
	meta := actionCatalog[t]
	return PendingAction{
		ID:          newID(),
		Type:        t,
		Label:       meta.Label,
		Description: meta.Description,
		RiskLevel:   meta.Risk,
		AutoApprove: meta.Auto,
		Status:      ActionPending,
		Params:      params,
		CreatedAt:   time.Now(),
	}
}
