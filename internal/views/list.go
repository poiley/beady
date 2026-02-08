package views

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/poiley/beady/internal/models"
	"github.com/poiley/beady/internal/ui"
)

const flashDurationValue = 2 * time.Second

// FlashDuration returns the duration before row flashes expire.
func FlashDuration() time.Duration {
	return flashDurationValue
}

// FlashExpiredMsg signals that row flashes should be cleared.
type FlashExpiredMsg struct{}

// SortField defines what field to sort by.
type SortField int

const (
	SortByPriority SortField = iota
	SortByCreated
	SortByUpdated
	SortByStatus
	SortByType
	SortByID
)

func (s SortField) String() string {
	switch s {
	case SortByPriority:
		return "priority"
	case SortByCreated:
		return "created"
	case SortByUpdated:
		return "updated"
	case SortByStatus:
		return "status"
	case SortByType:
		return "type"
	case SortByID:
		return "id"
	default:
		return "priority"
	}
}

// StatusFilter defines which statuses to show.
type StatusFilter int

const (
	FilterAll StatusFilter = iota
	FilterOpen
	FilterInProgress
	FilterBlocked
	FilterClosed
	FilterReady
)

func (f StatusFilter) String() string {
	switch f {
	case FilterAll:
		return "all"
	case FilterOpen:
		return "open"
	case FilterInProgress:
		return "in_progress"
	case FilterBlocked:
		return "blocked"
	case FilterClosed:
		return "closed"
	case FilterReady:
		return "ready"
	default:
		return "all"
	}
}

// ListView is the main list view model.
type ListView struct {
	allIssues    []models.Issue
	readyIDs     map[string]bool
	filtered     []models.Issue
	cursor       int
	offset       int
	width        int
	height       int
	sortField    SortField
	sortReverse  bool
	statusFilter StatusFilter
	hideClosed   bool
	filterInput  textinput.Model
	filtering    bool
	filterText   string
	stats        *models.StatsSummary

	// Change tracking for pulse flare on updated rows.
	prevUpdatedAt map[string]time.Time // issue ID -> UpdatedAt from last data load
	flashIDs      map[string]bool      // issue IDs currently flashing
}

// NewListView creates a new list view.
func NewListView() *ListView {
	ti := textinput.New()
	ti.Placeholder = "filter..."
	ti.CharLimit = 100
	return &ListView{
		sortField:     SortByPriority,
		statusFilter:  FilterAll,
		hideClosed:    true,
		filterInput:   ti,
		readyIDs:      make(map[string]bool),
		prevUpdatedAt: make(map[string]time.Time),
		flashIDs:      make(map[string]bool),
	}
}

// SetData updates the issue list and stats.
// Returns true if any issues changed (and row flashes were triggered).
func (l *ListView) SetData(issues []models.Issue, readyIssues []models.Issue, stats *models.StatsSummary) bool {
	// Detect changed rows by comparing UpdatedAt timestamps.
	hasFlashes := false
	if len(l.prevUpdatedAt) > 0 {
		// Build current index
		currentIDs := make(map[string]bool, len(issues))
		for _, issue := range issues {
			currentIDs[issue.ID] = true
			prev, existed := l.prevUpdatedAt[issue.ID]
			if !existed || !issue.UpdatedAt.Equal(prev) {
				l.flashIDs[issue.ID] = true
				hasFlashes = true
			}
		}
		// Flash newly removed issues? No — they won't render anyway.
		// Flash newly added issues.
		for _, issue := range issues {
			if _, existed := l.prevUpdatedAt[issue.ID]; !existed {
				l.flashIDs[issue.ID] = true
				hasFlashes = true
			}
		}
	}

	// Update the index for next comparison.
	l.prevUpdatedAt = make(map[string]time.Time, len(issues))
	for _, issue := range issues {
		l.prevUpdatedAt[issue.ID] = issue.UpdatedAt
	}

	l.allIssues = issues
	l.stats = stats
	l.readyIDs = make(map[string]bool)
	for _, ri := range readyIssues {
		l.readyIDs[ri.ID] = true
	}
	l.applyFilterAndSort()
	// Clamp cursor
	if l.cursor >= len(l.filtered) {
		l.cursor = max(0, len(l.filtered)-1)
	}
	return hasFlashes
}

// ClearFlashes removes all active row flashes.
func (l *ListView) ClearFlashes() {
	l.flashIDs = make(map[string]bool)
}

// HasFlashes returns whether any rows are currently flashing.
func (l *ListView) HasFlashes() bool {
	return len(l.flashIDs) > 0
}

// SetSize sets the terminal dimensions.
func (l *ListView) SetSize(w, h int) {
	l.width = w
	l.height = h
}

// SelectedIssue returns the currently selected issue, or nil.
func (l *ListView) SelectedIssue() *models.Issue {
	if len(l.filtered) == 0 || l.cursor >= len(l.filtered) {
		return nil
	}
	return &l.filtered[l.cursor]
}

// IsFiltering returns whether the filter input is active.
func (l *ListView) IsFiltering() bool {
	return l.filtering
}

// Update handles key messages for the list view.
func (l *ListView) Update(msg tea.Msg) tea.Cmd {
	if l.filtering {
		return l.updateFiltering(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if l.cursor < len(l.filtered)-1 {
				l.cursor++
				l.ensureVisible()
			}
		case "k", "up":
			if l.cursor > 0 {
				l.cursor--
				l.ensureVisible()
			}
		case "g", "home":
			l.cursor = 0
			l.offset = 0
		case "G", "end":
			l.cursor = max(0, len(l.filtered)-1)
			l.ensureVisible()
		case "ctrl+d":
			pageSize := l.visibleRows() / 2
			l.cursor = min(l.cursor+pageSize, max(0, len(l.filtered)-1))
			l.ensureVisible()
		case "ctrl+u":
			pageSize := l.visibleRows() / 2
			l.cursor = max(l.cursor-pageSize, 0)
			l.ensureVisible()
		case "s":
			l.sortField = (l.sortField + 1) % 6
			l.applyFilterAndSort()
		case "S":
			l.sortReverse = !l.sortReverse
			l.applyFilterAndSort()
		case "1":
			l.toggleStatusFilter(FilterOpen)
		case "2":
			l.toggleStatusFilter(FilterInProgress)
		case "3":
			l.toggleStatusFilter(FilterBlocked)
		case "4":
			l.toggleStatusFilter(FilterClosed)
		case "5":
			l.toggleStatusFilter(FilterReady)
		case "0":
			l.statusFilter = FilterAll
			l.applyFilterAndSort()
		case "c":
			l.hideClosed = !l.hideClosed
			l.applyFilterAndSort()
		case "/":
			l.filtering = true
			l.filterInput.Focus()
			return textinput.Blink
		}
	}
	return nil
}

func (l *ListView) toggleStatusFilter(f StatusFilter) {
	if l.statusFilter == f {
		l.statusFilter = FilterAll
	} else {
		l.statusFilter = f
	}
	l.applyFilterAndSort()
}

func (l *ListView) updateFiltering(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			l.filterText = l.filterInput.Value()
			l.filtering = false
			l.filterInput.Blur()
			l.applyFilterAndSort()
			return nil
		case "esc":
			l.filtering = false
			l.filterInput.Blur()
			l.filterInput.SetValue(l.filterText)
			return nil
		}
	}
	var cmd tea.Cmd
	l.filterInput, cmd = l.filterInput.Update(msg)
	// Live filter as user types
	l.filterText = l.filterInput.Value()
	l.applyFilterAndSort()
	return cmd
}

func (l *ListView) applyFilterAndSort() {
	// Filter
	var filtered []models.Issue
	for _, issue := range l.allIssues {
		// Hide closed by default unless toggled or explicitly filtering for closed
		if l.hideClosed && l.statusFilter != FilterClosed && issue.Status == "closed" {
			continue
		}
		if !l.matchesStatusFilter(issue) {
			continue
		}
		if !l.matchesTextFilter(issue) {
			continue
		}
		filtered = append(filtered, issue)
	}

	// Sort
	sort.SliceStable(filtered, func(i, j int) bool {
		cmp := l.compareIssues(filtered[i], filtered[j])
		if l.sortReverse {
			return cmp > 0
		}
		return cmp < 0
	})

	l.filtered = filtered
}

func (l *ListView) matchesStatusFilter(issue models.Issue) bool {
	switch l.statusFilter {
	case FilterAll:
		return true
	case FilterOpen:
		return issue.Status == "open"
	case FilterInProgress:
		return issue.Status == "in_progress"
	case FilterBlocked:
		return issue.Status == "blocked"
	case FilterClosed:
		return issue.Status == "closed"
	case FilterReady:
		return l.readyIDs[issue.ID]
	default:
		return true
	}
}

func (l *ListView) matchesTextFilter(issue models.Issue) bool {
	if l.filterText == "" {
		return true
	}
	needle := strings.ToLower(l.filterText)
	return strings.Contains(strings.ToLower(issue.ID), needle) ||
		strings.Contains(strings.ToLower(issue.Title), needle) ||
		strings.Contains(strings.ToLower(issue.IssueType), needle) ||
		strings.Contains(strings.ToLower(issue.Assignee), needle)
}

func (l *ListView) compareIssues(a, b models.Issue) int {
	switch l.sortField {
	case SortByPriority:
		if a.Priority != b.Priority {
			return a.Priority - b.Priority
		}
		// Secondary: newer first
		if a.CreatedAt.After(b.CreatedAt) {
			return -1
		}
		return 1
	case SortByCreated:
		if a.CreatedAt.After(b.CreatedAt) {
			return -1
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return 1
		}
		return 0
	case SortByUpdated:
		if a.UpdatedAt.After(b.UpdatedAt) {
			return -1
		}
		if a.UpdatedAt.Before(b.UpdatedAt) {
			return 1
		}
		return 0
	case SortByStatus:
		return statusOrder(a.Status) - statusOrder(b.Status)
	case SortByType:
		return strings.Compare(a.IssueType, b.IssueType)
	case SortByID:
		return strings.Compare(a.ID, b.ID)
	default:
		return 0
	}
}

func statusOrder(s string) int {
	switch s {
	case "in_progress":
		return 0
	case "open":
		return 1
	case "blocked":
		return 2
	case "deferred":
		return 3
	case "pinned":
		return 4
	case "closed":
		return 5
	default:
		return 6
	}
}

func (l *ListView) visibleRows() int {
	// header(3) + table header(1) + status bar(1) + filter bar if active(1)
	overhead := 5
	if l.filtering {
		overhead++
	}
	return max(1, l.height-overhead)
}

func (l *ListView) ensureVisible() {
	vis := l.visibleRows()
	if l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+vis {
		l.offset = l.cursor - vis + 1
	}
}

// View renders the list view.
func (l *ListView) View() string {
	var b strings.Builder

	// Header bar
	b.WriteString(l.renderHeader())
	b.WriteString("\n")

	// Table
	b.WriteString(l.renderTable())

	// Filter bar (if filtering)
	if l.filtering {
		b.WriteString("\n")
		b.WriteString(ui.FilterPromptStyle.Render("/") + " " + l.filterInput.View())
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(l.renderStatusBar())

	return b.String()
}

func (l *ListView) renderHeader() string {
	logo := ui.LogoStyle.Render("bdy")

	var parts []string
	parts = append(parts, fmt.Sprintf("%d issues", len(l.filtered)))

	if l.stats != nil {
		s := l.stats
		if s.OpenIssues > 0 {
			parts = append(parts, ui.StatusStyle("open").Render(fmt.Sprintf("%d open", s.OpenIssues)))
		}
		if s.InProgressIssues > 0 {
			parts = append(parts, ui.StatusStyle("in_progress").Render(fmt.Sprintf("%d in_progress", s.InProgressIssues)))
		}
		if s.BlockedIssues > 0 {
			parts = append(parts, ui.StatusStyle("blocked").Render(fmt.Sprintf("%d blocked", s.BlockedIssues)))
		}
		if s.ClosedIssues > 0 {
			parts = append(parts, ui.StatusStyle("closed").Render(fmt.Sprintf("%d closed", s.ClosedIssues)))
		}
		if s.ReadyIssues > 0 {
			parts = append(parts, lipgloss.NewStyle().Foreground(ui.ColorGreen).Render(fmt.Sprintf("%d ready", s.ReadyIssues)))
		}
	}

	info := strings.Join(parts, "  ")
	sortInfo := ui.KeyStyle.Render("sort:") + " " + ui.KeyDescStyle.Render(l.sortField.String())
	if l.sortReverse {
		sortInfo += ui.KeyDescStyle.Render(" (rev)")
	}
	filterInfo := ""
	if l.statusFilter != FilterAll {
		filterInfo = "  " + ui.KeyStyle.Render("filter:") + " " + ui.KeyDescStyle.Render(l.statusFilter.String())
	}
	if l.filterText != "" {
		filterInfo += "  " + ui.KeyStyle.Render("search:") + " " + ui.KeyDescStyle.Render(l.filterText)
	}
	if !l.hideClosed {
		filterInfo += "  " + ui.KeyDescStyle.Render("+closed")
	}

	left := logo + "  " + info
	right := sortInfo + filterInfo
	gap := max(0, l.width-lipgloss.Width(left)-lipgloss.Width(right)-2)
	header := left + strings.Repeat(" ", gap) + right

	return ui.HeaderStyle.Width(l.width).Render(header)
}

// colIndex constants for referencing columns in styleFn.
const (
	colIdxID       = 0
	colIdxPri      = 1
	colIdxStatus   = 2
	colIdxType     = 3
	colIdxTitle    = 4
	colIdxAssignee = 5
	colIdxAge      = 6
	colIdxDeps     = 7
)

func (l *ListView) renderTable() string {
	if len(l.filtered) == 0 {
		msg := "No issues found."
		if l.statusFilter != FilterAll || l.filterText != "" {
			msg += " Try clearing filters (press 0 or Esc)."
		}
		emptyHeight := max(1, l.height-6)
		spacer := strings.Repeat("\n", emptyHeight/2)
		return spacer + lipgloss.NewStyle().
			Width(l.width).
			Align(lipgloss.Center).
			Foreground(ui.ColorGray).
			Render(msg)
	}

	// Define columns using the generic table engine.
	tbl := ui.NewTable(
		&ui.Column{Header: "ID", Size: ui.SizeFit, Align: ui.AlignLeft, Min: 4, Max: 20},
		&ui.Column{Header: "PRI", Size: ui.SizeFixed, Align: ui.AlignLeft, Fixed: 3},
		&ui.Column{Header: "STATUS", Size: ui.SizeFit, Align: ui.AlignLeft, Min: 6, Max: 12},
		&ui.Column{Header: "TYPE", Size: ui.SizeFit, Align: ui.AlignLeft, Min: 4, Max: 10},
		&ui.Column{Header: "TITLE", Size: ui.SizeFlex, Align: ui.AlignLeft, Min: 10},
		&ui.Column{Header: "ASSIGNEE", Size: ui.SizeFit, Align: ui.AlignRight, Min: 1, Max: 14},
		&ui.Column{Header: "AGE", Size: ui.SizeFit, Align: ui.AlignRight, Min: 3, Max: 5},
		&ui.Column{Header: "↑/↓", Size: ui.SizeFit, Align: ui.AlignRight, Min: 4, Max: 7},
	)
	tbl.Gap = 1

	// Scan data to compute max display widths per column (for SizeFit columns).
	// Uses ui.StringWidth to correctly handle wide/multi-byte characters.
	dataWidths := make([]int, 8)
	for _, issue := range l.filtered {
		if n := ui.StringWidth(issue.ID); n > dataWidths[colIdxID] {
			dataWidths[colIdxID] = n
		}
		// PRI is fixed, no scan needed.
		if n := ui.StringWidth(issue.Status); n > dataWidths[colIdxStatus] {
			dataWidths[colIdxStatus] = n
		}
		if n := ui.StringWidth(issue.IssueType); n > dataWidths[colIdxType] {
			dataWidths[colIdxType] = n
		}
		// TITLE is flex, no scan needed.
		if n := ui.StringWidth(issue.Assignee); n > dataWidths[colIdxAssignee] {
			dataWidths[colIdxAssignee] = n
		}
		age := models.RelativeAge(issue.CreatedAt)
		if n := ui.StringWidth(age); n > dataWidths[colIdxAge] {
			dataWidths[colIdxAge] = n
		}
		deps := ""
		if issue.DependencyCount > 0 || issue.DependentCount > 0 {
			deps = fmt.Sprintf("%d/%d", issue.DependencyCount, issue.DependentCount)
		}
		if n := ui.StringWidth(deps); n > dataWidths[colIdxDeps] {
			dataWidths[colIdxDeps] = n
		}
	}
	// Ensure assignee column is wide enough for the "-" placeholder.
	if dataWidths[colIdxAssignee] < 1 {
		dataWidths[colIdxAssignee] = 1
	}

	// Reserve 2 chars for the cursor prefix ("  " or "> ").
	cursorWidth := 2
	tbl.Resolve(l.width-cursorWidth, dataWidths)

	// Render header row.
	headers := make([]string, 8)
	for i, col := range tbl.Columns {
		headers[i] = col.Header
	}
	hdr := "  " + tbl.RenderRow(headers, nil)
	headerRow := ui.TableHeaderStyle.Width(l.width).Render(hdr)

	// Data rows.
	vis := l.visibleRows()
	end := min(l.offset+vis, len(l.filtered))
	var rows []string
	rows = append(rows, headerRow)

	for i := l.offset; i < end; i++ {
		issue := l.filtered[i]
		selected := i == l.cursor

		cursor := "  "
		if selected {
			cursor = "> "
		}

		assignee := issue.Assignee
		if assignee == "" {
			assignee = "-"
		}
		deps := ""
		if issue.DependencyCount > 0 || issue.DependentCount > 0 {
			deps = fmt.Sprintf("%d/%d", issue.DependencyCount, issue.DependentCount)
		}

		cells := []string{
			issue.ID,
			issue.PriorityString(),
			issue.Status,
			issue.IssueType,
			issue.Title,
			assignee,
			models.RelativeAge(issue.CreatedAt),
			deps,
		}

		// Style function: pad happens first inside RenderRow, then this
		// wraps the already-padded plain text in ANSI colors.
		styleFn := func(col int, padded string) string {
			switch col {
			case colIdxPri:
				return ui.PriorityStyle(issue.Priority).Render(padded)
			case colIdxStatus:
				return ui.StatusStyle(issue.Status).Render(padded)
			case colIdxType:
				return ui.TypeStyle(issue.IssueType).Render(padded)
			default:
				return padded
			}
		}

		row := cursor + tbl.RenderRow(cells, styleFn)

		if selected {
			row = ui.SelectedRowStyle.Width(l.width).Render(row)
		} else if l.flashIDs[issue.ID] {
			row = ui.FlashRowStyle.Width(l.width).Render(row)
		}
		rows = append(rows, row)
	}

	// Pad remaining space with empty rows.
	rendered := len(rows) - 1 // subtract header
	for rendered < vis {
		rows = append(rows, strings.Repeat(" ", l.width))
		rendered++
	}

	return strings.Join(rows, "\n")
}

func (l *ListView) renderStatusBar() string {
	closedLabel := "show closed"
	if !l.hideClosed {
		closedLabel = "hide closed"
	}
	keys := []struct{ key, desc string }{
		{"enter", "view"},
		{"/", "filter"},
		{"s", "sort"},
		{"S", "reverse"},
		{"1-5", "status"},
		{"0", "all"},
		{"c", closedLabel},
		{"r", "refresh"},
		{"?", "help"},
		{"q", "quit"},
	}
	var parts []string
	for _, k := range keys {
		parts = append(parts, ui.KeyStyle.Render(k.key)+" "+ui.KeyDescStyle.Render(k.desc))
	}
	bar := strings.Join(parts, "  ")
	return ui.StatusBarStyle.Width(l.width).Render(bar)
}
