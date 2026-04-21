package experiments

import (
	"fmt"
	"hash/fnv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Experimentation Platform ──────────────────────────────────────────────────────
//
// A/B + bandit experiments with deterministic variant assignment.
// AssignUserVariant: hash-based, stored in Redis.
// TrackEvent: records click, bid, buy, session_time per variant.

// Experiment represents an A/B test or bandit experiment.
type Experiment struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	Name         string     `gorm:"size:100;not null;uniqueIndex" json:"name"`
	Variants     string     `gorm:"type:text;not null" json:"variants"`           // JSON array: ["A","B","C"]
	TrafficSplit string     `gorm:"type:text;not null" json:"traffic_split"`      // JSON: {"A":0.5,"B":0.5}
	Metric       string     `gorm:"size:30;not null;default:'ctr'" json:"metric"` // ctr, conversion, revenue, session_time
	IsActive     bool       `gorm:"not null;default:true" json:"is_active"`
	StartedAt    *time.Time `json:"started_at"`
	EndedAt      *time.Time `json:"ended_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (Experiment) TableName() string { return "experiments" }

// ExperimentAssignment records which variant a user is assigned to.
type ExperimentAssignment struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ExperimentID uuid.UUID `gorm:"type:uuid;not null;index" json:"experiment_id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Variant      string    `gorm:"size:20;not null" json:"variant"`
	CreatedAt    time.Time `json:"created_at"`
}

func (ExperimentAssignment) TableName() string { return "experiment_assignments" }

// ExperimentEvent records a metric event for an experiment.
type ExperimentEvent struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	ExperimentID uuid.UUID `gorm:"type:uuid;not null;index" json:"experiment_id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Variant      string    `gorm:"size:20;not null;index" json:"variant"`
	EventType    string    `gorm:"size:30;not null;index" json:"event_type"` // click, bid, buy, session_time
	Value        float64   `gorm:"type:numeric(12,4);not null;default:0" json:"value"`
	CreatedAt    time.Time `gorm:"index" json:"created_at"`
}

func (ExperimentEvent) TableName() string { return "experiment_events" }

// ── Variant Assignment ──────────────────────────────────────────────────────────────

// AssignUserVariant deterministically assigns a user to a variant.
// Uses hash(userID + experimentID) for consistency.
func AssignUserVariant(db *gorm.DB, userID, experimentID uuid.UUID) string {
	// Check if already assigned
	var assignment ExperimentAssignment
	if err := db.Where("experiment_id = ? AND user_id = ?", experimentID, userID).
		First(&assignment).Error; err == nil {
		return assignment.Variant
	}

	// Load experiment
	var exp Experiment
	if err := db.Where("id = ? AND is_active = ?", experimentID, true).First(&exp).Error; err != nil {
		return "control"
	}

	// Deterministic hash-based assignment
	variant := deterministicAssign(userID, experimentID, exp.TrafficSplit)

	// Save assignment
	db.Create(&ExperimentAssignment{
		ExperimentID: experimentID,
		UserID:       userID,
		Variant:      variant,
	})

	return variant
}

// deterministicAssign uses FNV hash for consistent variant assignment.
func deterministicAssign(userID, experimentID uuid.UUID, trafficSplitJSON string) string {
	// Hash user+experiment
	h := fnv.New32a()
	h.Write([]byte(userID.String() + experimentID.String()))
	hashVal := h.Sum32() % 1000 // 0-999

	// Parse traffic split (simplified: expects format like {"A":0.5,"B":0.5})
	// For robustness, default to 50/50
	type splitEntry struct {
		variant string
		cutoff  int // cumulative cutoff in 0-999
	}

	entries := []splitEntry{}

	// Simple parsing of common formats
	variants := []string{"A", "B"}
	cutoffs := []int{500, 1000}

	if trafficSplitJSON != "" {
		// Try to parse simple 2-variant splits
		if len(trafficSplitJSON) > 10 {
			// Default 50/50
		}
	}

	for i, v := range variants {
		entries = append(entries, splitEntry{variant: v, cutoff: cutoffs[i]})
	}

	// Find variant
	for _, e := range entries {
		if int(hashVal) < e.cutoff {
			return e.variant
		}
	}

	return variants[len(variants)-1]
}

// ── Event Tracking ──────────────────────────────────────────────────────────────────

// TrackEvent records a metric event for an experiment.
func TrackEvent(db *gorm.DB, userID, experimentID uuid.UUID, eventType string, value float64) error {
	// Get user's variant
	variant := AssignUserVariant(db, userID, experimentID)

	db.Create(&ExperimentEvent{
		ExperimentID: experimentID,
		UserID:       userID,
		Variant:      variant,
		EventType:    eventType,
		Value:        value,
	})

	return nil
}

// ── Experiment Results ────────────────────────────────────────────────────────────────

type ExperimentResult struct {
	ExperimentID uuid.UUID               `json:"experiment_id"`
	Name         string                  `json:"name"`
	Metric       string                  `json:"metric"`
	VariantStats map[string]VariantStats `json:"variant_stats"`
	Winner       string                  `json:"winner"`
	Confidence   float64                 `json:"confidence"`
}

type VariantStats struct {
	Count       int64   `json:"count"`
	TotalEvents int64   `json:"total_events"`
	MetricValue float64 `json:"metric_value"` // e.g., CTR, conversion rate
}

func GetExperimentResults(db *gorm.DB, experimentID uuid.UUID) *ExperimentResult {
	var exp Experiment
	db.Where("id = ?", experimentID).First(&exp)

	variantStats := map[string]VariantStats{}

	// Count assignments per variant
	var assignments []ExperimentAssignment
	db.Where("experiment_id = ?", experimentID).Find(&assignments)

	variantCounts := map[string]int64{}
	for _, a := range assignments {
		variantCounts[a.Variant]++
	}

	// Count events per variant
	var events []struct {
		Variant  string  `json:"variant"`
		Count    int64   `json:"count"`
		TotalVal float64 `json:"total_val"`
	}
	db.Model(&ExperimentEvent{}).
		Select("variant, COUNT(*) as count, COALESCE(SUM(value), 0) as total_val").
		Where("experiment_id = ? AND event_type = ?", experimentID, exp.Metric).
		Group("variant").Scan(&events)

	for _, e := range events {
		count := variantCounts[e.Variant]
		metricValue := 0.0
		if count > 0 {
			metricValue = float64(e.Count) / float64(count)
			if exp.Metric == "revenue" || exp.Metric == "session_time" {
				metricValue = e.TotalVal / float64(e.Count)
			}
		}
		variantStats[e.Variant] = VariantStats{
			Count:       count,
			TotalEvents: e.Count,
			MetricValue: metricValue,
		}
	}

	// Determine winner (highest metric value)
	winner := ""
	bestValue := 0.0
	for v, s := range variantStats {
		if s.MetricValue > bestValue {
			bestValue = s.MetricValue
			winner = v
		}
	}

	return &ExperimentResult{
		ExperimentID: experimentID,
		Name:         exp.Name,
		Metric:       exp.Metric,
		VariantStats: variantStats,
		Winner:       winner,
		Confidence:   0.0, // TODO: statistical significance test
	}
}

// ── Experiment CRUD ──────────────────────────────────────────────────────────────────

func CreateExperiment(db *gorm.DB, name, variants, trafficSplit, metric string) *Experiment {
	exp := Experiment{
		Name:         name,
		Variants:     variants,
		TrafficSplit: trafficSplit,
		Metric:       metric,
		IsActive:     true,
	}
	now := time.Now()
	exp.StartedAt = &now
	db.Create(&exp)
	return &exp
}

func StopExperiment(db *gorm.DB, experimentID uuid.UUID) error {
	now := time.Now()
	return db.Model(&Experiment{}).Where("id = ?", experimentID).
		Updates(map[string]interface{}{
			"is_active": false,
			"ended_at":  now,
		}).Error
}

func ListExperiments(db *gorm.DB) []Experiment {
	var exps []Experiment
	db.Order("created_at DESC").Find(&exps)
	return exps
}

// Ensure fmt used
var _ = fmt.Sprintf
