package store

import (
	"sync"
	"time"

	"github.com/geocore-next/backend/internal/controlplane/analyzer"
)

// StateStore holds the current and historical system state.
type StateStore struct {
	mu       sync.RWMutex
	current  analyzer.Metrics
	history  []analyzer.Metrics
	maxHist  int
}

// NewStateStore creates a state store.
func NewStateStore() *StateStore {
	return &StateStore{
		maxHist: 1000,
	}
}

// Set updates the current state.
func (s *StateStore) Set(m analyzer.Metrics) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m.Timestamp = time.Now().UTC()
	s.current = m
	s.history = append(s.history, m)
	if len(s.history) > s.maxHist {
		s.history = s.history[1:]
	}
}

// Get returns the current state.
func (s *StateStore) Get() analyzer.Metrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current
}

// History returns the last N states.
func (s *StateStore) History(n int) []analyzer.Metrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if n > len(s.history) {
		n = len(s.history)
	}
	result := make([]analyzer.Metrics, n)
	copy(result, s.history[len(s.history)-n:])
	return result
}
