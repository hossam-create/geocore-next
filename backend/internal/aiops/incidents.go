package aiops

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type Severity string

const (
	SeverityP0 Severity = "P0" // page immediately
	SeverityP1 Severity = "P1" // slack warning
	SeverityP2 Severity = "P2" // email digest
)

type IncidentStatus string

const (
	StatusOpen     IncidentStatus = "open"
	StatusResolved IncidentStatus = "resolved"
	StatusIgnored  IncidentStatus = "ignored"
)

type Incident struct {
	ID          string         `json:"id"`
	Severity    Severity       `json:"severity"`
	Service     string         `json:"service"`
	Metric      string         `json:"metric"`
	Value       float64        `json:"value"`
	Baseline    float64        `json:"baseline"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	RCA         string         `json:"rca"`
	Runbook     string         `json:"runbook"`
	Status      IncidentStatus `json:"status"`
	DetectedAt  time.Time      `json:"detected_at"`
	ResolvedAt  *time.Time     `json:"resolved_at,omitempty"`
}

type incidentRegistry struct {
	mu    sync.RWMutex
	items []*Incident
}

var registry = &incidentRegistry{}

func (r *incidentRegistry) Add(inc *Incident) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = append([]*Incident{inc}, r.items...)
	if len(r.items) > 500 {
		r.items = r.items[:500]
	}
}

func (r *incidentRegistry) List() []*Incident {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Incident, len(r.items))
	copy(out, r.items)
	return out
}

func (r *incidentRegistry) Get(id string) *Incident {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, inc := range r.items {
		if inc.ID == id {
			return inc
		}
	}
	return nil
}

func (r *incidentRegistry) UpdateStatus(id string, status IncidentStatus) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, inc := range r.items {
		if inc.ID == id {
			inc.Status = status
			if status == StatusResolved {
				t := time.Now()
				inc.ResolvedAt = &t
			}
			return true
		}
	}
	return false
}

func (r *incidentRegistry) OpenCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := 0
	for _, inc := range r.items {
		if inc.Status == StatusOpen {
			n++
		}
	}
	return n
}

func newIncidentID() string {
	return uuid.New().String()
}

// GetOpenCount returns the current number of open incidents.
// Exported for in-process use by the stress testing validator.
func GetOpenCount() int {
	return registry.OpenCount()
}
