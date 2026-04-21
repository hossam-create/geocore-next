package gitops

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/geocore-next/backend/internal/cloudos/resources"
)

// Watcher monitors Git for changes and converts them to desired state.
type Watcher struct {
	mu         sync.RWMutex
	repoURL    string
	branch     string
	lastCommit string
	changes    []resources.Change
}

// NewWatcher creates a GitOps watcher.
func NewWatcher(repoURL, branch string) *Watcher {
	return &Watcher{
		repoURL: repoURL,
		branch:  branch,
	}
}

// DetectChanges polls for new changes from the Git repository.
func (w *Watcher) DetectChanges(ctx context.Context) []resources.Change {
	w.mu.Lock()
	defer w.mu.Unlock()

	// In production: git ls-remote or webhook receiver
	// Here: return empty (no changes detected in simulation)
	changes := []resources.Change{}

	if len(changes) > 0 {
		slog.Info("cloudos: git changes detected", "count", len(changes))
		w.changes = append(w.changes, changes...)
		w.lastCommit = time.Now().UTC().Format("20060102150405")
	}

	return changes
}

// LastCommit returns the most recently seen commit.
func (w *Watcher) LastCommit() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.lastCommit
}
