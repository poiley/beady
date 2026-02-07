package bd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/poiley/beady/internal/models"
)

// Client wraps the bd CLI for JSON data retrieval.
type Client struct {
	// WorkDir is the directory to run bd commands in (for --dir flag or cwd).
	WorkDir string
}

// NewClient creates a new bd CLI client.
func NewClient(workDir string) *Client {
	return &Client{WorkDir: workDir}
}

// run executes a bd command and returns stdout.
func (c *Client) run(args ...string) ([]byte, error) {
	args = append(args, "--json")
	cmd := exec.Command("bd", args...)
	if c.WorkDir != "" {
		cmd.Dir = c.WorkDir
	}
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("bd %s failed: %s\n%s", strings.Join(args, " "), err, string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("bd %s failed: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

// List returns all issues (non-closed by default).
func (c *Client) List() ([]models.Issue, error) {
	out, err := c.run("list", "--limit", "0")
	if err != nil {
		return nil, err
	}
	if len(out) == 0 || strings.TrimSpace(string(out)) == "" {
		return nil, nil
	}
	var issues []models.Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("parsing bd list output: %w", err)
	}
	return issues, nil
}

// ListAll returns all issues including closed.
func (c *Client) ListAll() ([]models.Issue, error) {
	out, err := c.run("list", "--all", "--limit", "0")
	if err != nil {
		return nil, err
	}
	if len(out) == 0 || strings.TrimSpace(string(out)) == "" {
		return nil, nil
	}
	var issues []models.Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("parsing bd list --all output: %w", err)
	}
	return issues, nil
}

// Ready returns ready (unblocked) issues.
func (c *Client) Ready() ([]models.Issue, error) {
	out, err := c.run("ready", "--limit", "0")
	if err != nil {
		return nil, err
	}
	if len(out) == 0 || strings.TrimSpace(string(out)) == "" {
		return nil, nil
	}
	var issues []models.Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("parsing bd ready output: %w", err)
	}
	return issues, nil
}

// Show returns full details for a single issue.
func (c *Client) Show(id string) (*models.Issue, error) {
	out, err := c.run("show", id)
	if err != nil {
		return nil, err
	}
	// bd show --json returns an array with one element
	var issues []models.Issue
	if err := json.Unmarshal(out, &issues); err != nil {
		return nil, fmt.Errorf("parsing bd show output: %w", err)
	}
	if len(issues) == 0 {
		return nil, fmt.Errorf("issue %s not found", id)
	}
	return &issues[0], nil
}

// Stats returns aggregate statistics.
func (c *Client) Stats() (*models.StatsSummary, error) {
	out, err := c.run("stats")
	if err != nil {
		return nil, err
	}
	var stats models.Stats
	if err := json.Unmarshal(out, &stats); err != nil {
		return nil, fmt.Errorf("parsing bd stats output: %w", err)
	}
	return &stats.Summary, nil
}

// CheckInit verifies that bd is available and the current dir has beads initialized.
func (c *Client) CheckInit() error {
	_, err := exec.LookPath("bd")
	if err != nil {
		return fmt.Errorf("bd CLI not found in PATH. Install with: brew install beads")
	}

	cmd := exec.Command("bd", "stats", "--json")
	if c.WorkDir != "" {
		cmd.Dir = c.WorkDir
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("beads not initialized in this directory. Run: bd init")
	}
	return nil
}
