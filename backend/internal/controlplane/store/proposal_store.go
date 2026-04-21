package store

import (
	"sync"
	"time"

	"github.com/geocore-next/backend/internal/controlplane/planner"
)

// ProposalRecord tracks the lifecycle of a proposal.
type ProposalRecord struct {
	Proposal   planner.Proposal `json:"proposal"`
	Status     string           `json:"status"` // proposed, simulated, approved, executed, rejected, rolled_back
	CreatedAt  time.Time        `json:"created_at"`
	AppliedAt  *time.Time       `json:"applied_at,omitempty"`
	RejectedAt *time.Time       `json:"rejected_at,omitempty"`
	Reason     string           `json:"reason,omitempty"`
}

// ProposalStore records all proposals and their outcomes for learning.
type ProposalStore struct {
	mu       sync.RWMutex
	records  []ProposalRecord
	maxRec   int
}

// NewProposalStore creates a proposal store.
func NewProposalStore() *ProposalStore {
	return &ProposalStore{
		maxRec: 500,
	}
}

// Add records a new proposal.
func (ps *ProposalStore) Add(p planner.Proposal, status, reason string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	record := ProposalRecord{
		Proposal:  p,
		Status:    status,
		CreatedAt: time.Now().UTC(),
		Reason:    reason,
	}
	if status == "executed" {
		now := time.Now().UTC()
		record.AppliedAt = &now
	}
	if status == "rejected" {
		now := time.Now().UTC()
		record.RejectedAt = &now
	}

	ps.records = append(ps.records, record)
	if len(ps.records) > ps.maxRec {
		ps.records = ps.records[1:]
	}
}

// Recent returns the last N proposal records.
func (ps *ProposalStore) Recent(n int) []ProposalRecord {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if n > len(ps.records) {
		n = len(ps.records)
	}
	result := make([]ProposalRecord, n)
	copy(result, ps.records[len(ps.records)-n:])
	return result
}

// CountByStatus returns the count of proposals per status.
func (ps *ProposalStore) CountByStatus() map[string]int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	counts := make(map[string]int)
	for _, r := range ps.records {
		counts[r.Status]++
	}
	return counts
}
