package git

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

// Client provides Git operations for the GitOps controller.
type Client struct {
	repoURL string
	branch  string
	dir     string // local clone directory
}

// NewClient creates a Git client for the given repository.
func NewClient(repoURL, branch, dir string) *Client {
	return &Client{repoURL: repoURL, branch: branch, dir: dir}
}

// Clone performs a git clone of the repository.
func (c *Client) Clone(ctx context.Context) error {
	slog.Info("git: cloning repository", "url", c.repoURL, "branch", c.branch)
	cmd := exec.CommandContext(ctx, "git", "clone", "-b", c.branch, c.repoURL, c.dir)
	return cmd.Run()
}

// Pull fetches the latest changes from the remote.
func (c *Client) Pull(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "-C", c.dir, "pull", "--ff-only")
	return cmd.Run()
}

// LatestCommit returns the latest commit SHA on the current branch.
func (c *Client) LatestCommit(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", c.dir, "rev-parse", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// Diff returns the list of changed files between two commits.
func (c *Client) Diff(ctx context.Context, from, to string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", c.dir, "diff", "--name-only", from, to)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	files := strings.Split(strings.TrimSpace(string(output)), "\n")
	return files, nil
}

// Log returns the last N commit messages.
func (c *Client) Log(ctx context.Context, n int) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", c.dir, "log", "--oneline", "-n", fmt.Sprintf("%d", n))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimSpace(string(output)), "\n"), nil
}
