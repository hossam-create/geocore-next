package reslab

import "sync"

// History stores experiment run results in-memory (last 100 runs, newest first).
type History struct {
	mu      sync.RWMutex
	results []*RunResult
}

var globalHistory = &History{}

func (h *History) Add(r *RunResult) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.results = append([]*RunResult{r}, h.results...)
	if len(h.results) > 100 {
		h.results = h.results[:100]
	}
}

func (h *History) List() []*RunResult {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]*RunResult, len(h.results))
	copy(out, h.results)
	return out
}

// Baseline returns the oldest stored result for an experiment as its baseline.
func (h *History) Baseline(experimentID string) *RunResult {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var oldest *RunResult
	for i := len(h.results) - 1; i >= 0; i-- {
		if h.results[i].ExperimentID == experimentID {
			oldest = h.results[i]
		}
	}
	return oldest
}

// Trend returns the last N overall scores for an experiment (newest first).
func (h *History) Trend(experimentID string, n int) []float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var scores []float64
	for _, r := range h.results {
		if r.ExperimentID == experimentID {
			scores = append(scores, r.Score.Overall)
			if len(scores) >= n {
				break
			}
		}
	}
	return scores
}
