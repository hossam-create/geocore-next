package tenant

import "github.com/geocore-next/backend/internal/billing"

// RecordRequest records a billable API request for a tenant.
// No-op when the GlobalMeter is uninitialised or tenantID is empty.
func RecordRequest(tenantID string, count int64) {
	if billing.GlobalMeter == nil || tenantID == "" {
		return
	}
	billing.GlobalMeter.Record(tenantID, billing.EventRequests, count)
}

// RecordKafkaEvent records billable Kafka event processing.
func RecordKafkaEvent(tenantID string, count int64) {
	if billing.GlobalMeter == nil || tenantID == "" {
		return
	}
	billing.GlobalMeter.Record(tenantID, billing.EventKafkaEvents, count)
}

// RecordAIOpsIncident records a billed AIOps incident resolution.
func RecordAIOpsIncident(tenantID string) {
	if billing.GlobalMeter == nil || tenantID == "" {
		return
	}
	billing.GlobalMeter.Record(tenantID, billing.EventAIOpsIncident, 1)
}

// RecordChaosRun records a billed chaos/stress test execution.
func RecordChaosRun(tenantID string) {
	if billing.GlobalMeter == nil || tenantID == "" {
		return
	}
	billing.GlobalMeter.Record(tenantID, billing.EventChaosRun, 1)
}
