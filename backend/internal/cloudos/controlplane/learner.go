package controlplane

import (
	"sync"
	"time"

	"github.com/geocore-next/backend/internal/cloudos/resources"
)

// Outcome records the result of an applied proposal for learning.
type Outcome struct {
	Proposal   resources.Proposal
	Healthy    bool
	LearnedAt  time.Time
}

// Learner records and learns from proposal outcomes.
type Learner struct {
	mu       sync.RWMutex
	outcomes []Outcome
	maxRec   int
}

// NewLearner creates a learning engine.
func NewLearner() *Learner {
	return &Learner{maxRec: 500}
}

// Record stores the outcome of a proposal.
func (l *Learner) Record(p resources.Proposal, state resources.ClusterState, healthy bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.outcomes = append(l.outcomes, Outcome{
		Proposal:  p,
		Healthy:   healthy,
		LearnedAt: time.Now().UTC(),
	})
	if len(l.outcomes) > l.maxRec {
		l.outcomes = l.outcomes[1:]
	}
}

// SuccessRate returns the success rate for a given resource type.
func (l *Learner) SuccessRate(resourceType string) float64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var total, success float64
	for _, o := range l.outcomes {
		if o.Proposal.Resource == resourceType || resourceType == "" {
			total++
			if o.Healthy {
				success++
			}
		}
	}
	if total == 0 {
		return 1.0
	}
	return success / total
}

// Recent returns the last N outcomes.
func (l *Learner) Recent(n int) []Outcome {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if n > len(l.outcomes) {
		n = len(l.outcomes)
	}
	result := make([]Outcome, n)
	copy(result, l.outcomes[len(l.outcomes)-n:])
	return result
}
