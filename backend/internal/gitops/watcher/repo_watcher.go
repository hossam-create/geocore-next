package watcher

import (
	"context"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Change represents a detected change in the Git repository.
type Change struct {
	Service   string    `json:"service"`
	Version   string    `json:"version"`
	Env       string    `json:"env"`
	CommitSHA string    `json:"commit_sha"`
	Author    string    `json:"author"`
	Message   string    `json:"message"`
	DetectedAt time.Time `json:"detected_at"`
}

// RepoWatcher monitors a Git repository for changes.
type RepoWatcher struct {
	mu         sync.RWMutex
	repoURL    string
	branch     string
	lastCommit string
	changes    []Change
}

// NewRepoWatcher creates a watcher for the given repo and branch.
func NewRepoWatcher(repoURL, branch string) *RepoWatcher {
	return &RepoWatcher{
		repoURL: repoURL,
		branch:  branch,
	}
}

// DetectChanges polls the Git repo for new commits and returns changes.
func (w *RepoWatcher) DetectChanges(ctx context.Context) []Change {
	w.mu.Lock()
	defer w.mu.Unlock()

	latestCommit := w.getLatestCommit(ctx)
	if latestCommit == "" {
		return nil
	}

	// No new commits
	if latestCommit == w.lastCommit {
		return nil
	}

	// Parse changes from diff
	changes := w.parseDiff(ctx, w.lastCommit, latestCommit)
	if len(changes) > 0 {
		slog.Info("gitops: new changes detected",
			"commits", latestCommit[:8],
			"changes", len(changes))
	}

	w.lastCommit = latestCommit
	w.changes = append(w.changes, changes...)
	if len(w.changes) > 100 {
		w.changes = w.changes[1:]
	}

	return changes
}

// LastCommit returns the most recently seen commit SHA.
func (w *RepoWatcher) LastCommit() string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.lastCommit
}

func (w *RepoWatcher) getLatestCommit(ctx context.Context) string {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", w.repoURL, "refs/heads/"+w.branch)
	output, err := cmd.Output()
	if err != nil {
		slog.Debug("gitops: could not fetch remote", "error", err)
		return ""
	}
	parts := strings.Fields(string(output))
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func (w *RepoWatcher) parseDiff(ctx context.Context, from, to string) []Change {
	// In production: use go-git to diff commits and extract service/version changes
	// Here: detect from commit message or kustomize/helm overlays
	var changes []Change

	// If no previous commit, treat as initial sync
	if from == "" {
		changes = append(changes, Change{
			Service:   "api",
			Version:   "latest",
			Env:       "prod",
			CommitSHA: to,
			Author:    "system",
			Message:   "initial sync",
			DetectedAt: time.Now().UTC(),
		})
		return changes
	}

	// Parse changed services from diff
	// Production: inspect kustomize/k8s manifests for image tag changes
	changes = append(changes, Change{
		Service:   "api",
		Version:   to[:8], // short SHA as version
		Env:       "prod",
		CommitSHA: to,
		Author:    "git",
		Message:   "detected from diff " + from[:8] + ".." + to[:8],
		DetectedAt: time.Now().UTC(),
	})

	return changes
}
