package billing

import (
	"time"

	"gorm.io/gorm"
)

// EventType is a billable usage dimension.
type EventType string

const (
	EventRequests      EventType = "requests"
	EventKafkaEvents   EventType = "kafka_events"
	EventAIOpsIncident EventType = "aiops_incidents"
	EventChaosRun      EventType = "chaos_runs"
	EventStorageGBHour EventType = "storage_gb_hour"
)

// UsageEvent is one metered usage record persisted to the database.
type UsageEvent struct {
	ID        string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID  string    `gorm:"type:uuid;not null;index"                       json:"tenant_id"`
	EventType EventType `gorm:"type:varchar(50);not null"                      json:"event_type"`
	Quantity  int64     `gorm:"not null;default:1"                             json:"quantity"`
	Metadata  string    `gorm:"type:jsonb"                                     json:"-"`
	Ts        time.Time `gorm:"not null;default:now()"                         json:"ts"`
}

// UsageSummary aggregates usage quantities by event type for a given period.
type UsageSummary struct {
	TenantID string               `json:"tenant_id"`
	Start    time.Time            `json:"period_start"`
	End      time.Time            `json:"period_end"`
	Events   map[EventType]int64  `json:"events"`
}

// RecordEvent writes a single usage event directly to the DB (low-frequency path).
// For high-frequency recording use the Meter instead.
func RecordEvent(db *gorm.DB, tenantID string, et EventType, qty int64) error {
	return db.Create(&UsageEvent{
		TenantID:  tenantID,
		EventType: et,
		Quantity:  qty,
		Ts:        time.Now(),
	}).Error
}

// Summarize aggregates usage for a tenant between start and end.
func Summarize(db *gorm.DB, tenantID string, start, end time.Time) UsageSummary {
	type row struct {
		EventType EventType
		Total     int64
	}
	var rows []row
	db.Table("usage_events").
		Select("event_type, SUM(quantity) as total").
		Where("tenant_id = ? AND ts BETWEEN ? AND ?", tenantID, start, end).
		Group("event_type").
		Scan(&rows)

	events := make(map[EventType]int64, len(rows))
	for _, r := range rows {
		events[r.EventType] = r.Total
	}
	return UsageSummary{TenantID: tenantID, Start: start, End: end, Events: events}
}
