package redteam

import (
	"time"

	"github.com/google/uuid"
)

// RedTeamRun is the persistent audit trail for every simulation run.
// Stored so admins can verify that defenses keep holding over time.
type RedTeamRun struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Scenario     string         `gorm:"size:32;not null;index"                          json:"scenario"`
	StartedAt    time.Time      `gorm:"index"                                           json:"started_at"`
	DurationMs   int64          `                                                       json:"duration_ms"`
	Attempts     int            `                                                       json:"attempts"`
	Blocked      int            `                                                       json:"blocked"`
	FirstBlockAt int            `                                                       json:"first_block_at"`
	Passed       bool           `gorm:"index"                                           json:"passed"`
	InitiatedBy  *uuid.UUID     `gorm:"type:uuid"                                       json:"initiated_by,omitempty"`
	Result       map[string]any `gorm:"type:jsonb;serializer:json;default:'{}'"          json:"result"`
	CreatedAt    time.Time      `                                                       json:"created_at"`
}

func (RedTeamRun) TableName() string { return "redteam_runs" }

// ScenarioResult is what each simulator returns AND what the API serialises.
// Single canonical shape so the dashboard UI does not need per-scenario knowledge.
type ScenarioResult struct {
	Scenario         string    `json:"scenario"`
	StartedAt        time.Time `json:"started_at"`
	DurationMs       int64     `json:"duration_ms"`
	Attempts         int       `json:"attempts"`           // total attack attempts issued
	Blocked          int       `json:"blocked"`            // attempts stopped by defenses
	FirstBlockAt     int       `json:"first_block_at"`     // attempt# where defense first fired (-1 if never)
	SystemResponded  bool      `json:"system_responded"`   // did ANY defensive path return a decision?
	DefenseTriggered bool      `json:"defense_triggered"`  // did defense eventually stop the attack?
	TriggeredAlerts  []string  `json:"triggered_alerts"`   // names of defensive mechanisms that fired
	Passed           bool      `json:"passed"`             // overall: did the system survive the attack?
	Notes            []string  `json:"notes"`              // human-readable log lines
	Metrics          map[string]any `json:"metrics,omitempty"`
}
