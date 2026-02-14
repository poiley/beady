package models

import (
	"fmt"
	"math"
	"time"
)

// Issue represents a bead issue as returned by bd --json.
type Issue struct {
	ID                 string     `json:"id"`
	Title              string     `json:"title"`
	Description        string     `json:"description,omitempty"`
	Design             string     `json:"design,omitempty"`
	AcceptanceCriteria string     `json:"acceptance_criteria,omitempty"`
	Notes              string     `json:"notes,omitempty"`
	Status             string     `json:"status,omitempty"`
	Priority           int        `json:"priority"`
	IssueType          string     `json:"issue_type,omitempty"`
	Assignee           string     `json:"assignee,omitempty"`
	Owner              string     `json:"owner,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	CreatedBy          string     `json:"created_by,omitempty"`
	UpdatedAt          time.Time  `json:"updated_at"`
	ClosedAt           *time.Time `json:"closed_at,omitempty"`
	CloseReason        string     `json:"close_reason,omitempty"`
	DueAt              *time.Time `json:"due_at,omitempty"`
	DeferUntil         *time.Time `json:"defer_until,omitempty"`
	Pinned             bool       `json:"pinned,omitempty"`

	// Counts from bd list
	DependencyCount int `json:"dependency_count"`
	DependentCount  int `json:"dependent_count"`
	CommentCount    int `json:"comment_count"`

	// Relational data from bd show
	Labels       []string            `json:"labels,omitempty"`
	Dependencies []*IssueWithDepType `json:"dependencies,omitempty"`
	Dependents   []*IssueWithDepType `json:"dependents,omitempty"`
	Comments     []*Comment          `json:"comments,omitempty"`
	Parent       *string             `json:"parent,omitempty"`
}

// IssueWithDepType is an issue with a dependency type annotation.
// It handles two JSON formats:
//   - bd show: full issue fields with "id" and "dependency_type"
//   - bd list: raw dependency record with "depends_on_id" and "type"
type IssueWithDepType struct {
	Issue
	DependencyType string `json:"dependency_type"`

	// Raw dependency fields from bd list JSON format.
	DependsOnID string `json:"depends_on_id,omitempty"`
	DepType     string `json:"type,omitempty"`
}

// ParentID returns the parent issue ID from whichever JSON format was used.
// For bd show dependents, the parent is the issue that was shown (not in this struct).
// For bd list dependencies, the parent is in DependsOnID.
func (d *IssueWithDepType) ParentID() string {
	if d.DependsOnID != "" {
		return d.DependsOnID
	}
	return d.ID
}

// DepTypeValue returns the dependency type from whichever JSON format was used.
func (d *IssueWithDepType) DepTypeValue() string {
	if d.DepType != "" {
		return d.DepType
	}
	return d.DependencyType
}

// Comment represents a comment on an issue.
type Comment struct {
	ID        int64     `json:"id"`
	IssueID   string    `json:"issue_id"`
	Author    string    `json:"author"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"created_at"`
}

// Stats represents aggregate statistics from bd stats.
type Stats struct {
	Summary StatsSummary `json:"summary"`
}

// StatsSummary is the summary block inside bd stats output.
type StatsSummary struct {
	TotalIssues      int     `json:"total_issues"`
	OpenIssues       int     `json:"open_issues"`
	InProgressIssues int     `json:"in_progress_issues"`
	ClosedIssues     int     `json:"closed_issues"`
	BlockedIssues    int     `json:"blocked_issues"`
	DeferredIssues   int     `json:"deferred_issues"`
	ReadyIssues      int     `json:"ready_issues"`
	TombstoneIssues  int     `json:"tombstone_issues"`
	PinnedIssues     int     `json:"pinned_issues"`
	AvgLeadTimeHours float64 `json:"average_lead_time_hours"`
}

// PriorityString returns "P0", "P1", etc.
func (i *Issue) PriorityString() string {
	return fmt.Sprintf("P%d", i.Priority)
}

// RelativeAge returns a human-readable relative time string.
func RelativeAge(t time.Time) string {
	d := time.Since(t)
	if d < 0 {
		d = -d
	}

	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		m := int(math.Round(d.Minutes()))
		return fmt.Sprintf("%dm", m)
	case d < 24*time.Hour:
		h := int(math.Round(d.Hours()))
		return fmt.Sprintf("%dh", h)
	case d < 30*24*time.Hour:
		days := int(math.Round(d.Hours() / 24))
		return fmt.Sprintf("%dd", days)
	case d < 365*24*time.Hour:
		months := int(math.Round(d.Hours() / (24 * 30)))
		return fmt.Sprintf("%dmo", months)
	default:
		years := int(math.Round(d.Hours() / (24 * 365)))
		return fmt.Sprintf("%dy", years)
	}
}
