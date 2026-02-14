package views

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/poiley/beady/internal/models"
	"github.com/poiley/beady/internal/ui"
)

// NavigateToIssueMsg requests the app to load a detail view for the given ID.
type NavigateToIssueMsg struct {
	ID string
}

// navItem maps a rendered line to a navigable issue.
type navItem struct {
	lineIndex int    // index into d.lines
	issueID   string // issue to navigate to on enter
}

// DetailView shows full details for a single issue.
type DetailView struct {
	issue     *models.Issue
	width     int
	height    int
	scroll    int
	lines     []string // pre-rendered content lines
	statusMsg string   // temporary status bar message

	// Navigation within deps/dependents.
	navItems  []navItem // navigable lines (dependencies + dependents)
	navCursor int       // index into navItems, -1 = none selected
}

// NewDetailView creates a detail view for an issue.
func NewDetailView(issue *models.Issue) *DetailView {
	d := &DetailView{issue: issue, navCursor: -1}
	d.buildContent()
	return d
}

// SetSize sets terminal dimensions.
func (d *DetailView) SetSize(w, h int) {
	d.width = w
	d.height = h
	d.buildContent()
}

// IssueID returns the ID of the displayed issue.
func (d *DetailView) IssueID() string {
	if d.issue == nil {
		return ""
	}
	return d.issue.ID
}

// SetStatusMsg sets a temporary status bar message.
func (d *DetailView) SetStatusMsg(msg string) {
	d.statusMsg = msg
}

// UpdateIssue replaces the issue data and re-renders content while
// preserving the current scroll position (clamped to the new content length).
func (d *DetailView) UpdateIssue(issue *models.Issue) {
	d.issue = issue
	d.buildContent()
	// Clamp scroll to new content bounds
	maxScroll := max(0, len(d.lines)-d.visibleLines())
	if d.scroll > maxScroll {
		d.scroll = maxScroll
	}
}

// SelectedNavID returns the issue ID of the currently selected nav item,
// or empty string if nothing is selected.
func (d *DetailView) SelectedNavID() string {
	if d.navCursor < 0 || d.navCursor >= len(d.navItems) {
		return ""
	}
	return d.navItems[d.navCursor].issueID
}

// Update handles key messages.
func (d *DetailView) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if d.scroll < len(d.lines)-d.visibleLines() {
				d.scroll++
			}
		case "k", "up":
			if d.scroll > 0 {
				d.scroll--
			}
		case "g", "home":
			d.scroll = 0
		case "G", "end":
			d.scroll = max(0, len(d.lines)-d.visibleLines())
		case "ctrl+d":
			page := d.visibleLines() / 2
			d.scroll = min(d.scroll+page, max(0, len(d.lines)-d.visibleLines()))
		case "ctrl+u":
			page := d.visibleLines() / 2
			d.scroll = max(d.scroll-page, 0)
		case "tab":
			if len(d.navItems) > 0 {
				d.navCursor++
				if d.navCursor >= len(d.navItems) {
					d.navCursor = 0
				}
				d.scrollToNav()
			}
		case "shift+tab":
			if len(d.navItems) > 0 {
				d.navCursor--
				if d.navCursor < 0 {
					d.navCursor = len(d.navItems) - 1
				}
				d.scrollToNav()
			}
		case "enter":
			if id := d.SelectedNavID(); id != "" {
				return func() tea.Msg {
					return NavigateToIssueMsg{ID: id}
				}
			}
		}
	}
	return nil
}

// scrollToNav scrolls the view to make the current nav item visible.
func (d *DetailView) scrollToNav() {
	if d.navCursor < 0 || d.navCursor >= len(d.navItems) {
		return
	}
	line := d.navItems[d.navCursor].lineIndex
	vis := d.visibleLines()
	if line < d.scroll {
		d.scroll = line
	} else if line >= d.scroll+vis {
		d.scroll = line - vis + 1
	}
}

func (d *DetailView) visibleLines() int {
	return ui.ContentHeight(d.height, d.renderHeaderChrome(), d.renderStatusBar())
}

// View renders the detail view.
func (d *DetailView) View() string {
	vis := d.visibleLines()
	var b strings.Builder

	// Header (with scroll info that depends on vis)
	b.WriteString(d.renderHeader(vis))
	b.WriteString("\n")

	// Content area
	b.WriteString(d.renderContent())
	b.WriteString("\n")

	// Status bar
	b.WriteString(d.renderStatusBar())

	return b.String()
}

// renderHeaderChrome returns the header without scroll info, used only for
// measuring the header's height (avoids recursion with visibleLines).
func (d *DetailView) renderHeaderChrome() string {
	if d.issue == nil {
		return ui.HeaderStyle.Width(d.width).Render("(no issue)")
	}
	issue := d.issue
	left := fmt.Sprintf("%s  %s  %s  %s",
		ui.LogoStyle.Render(issue.ID),
		ui.PriorityStyle(issue.Priority).Render(issue.PriorityString()),
		ui.StatusStyle(issue.Status).Render(issue.Status),
		ui.TypeStyle(issue.IssueType).Render(issue.IssueType),
	)
	return ui.HeaderStyle.Width(d.width).Render(left)
}

func (d *DetailView) renderHeader(vis int) string {
	if d.issue == nil {
		return ui.HeaderStyle.Width(d.width).Render("(no issue)")
	}
	issue := d.issue

	id := ui.LogoStyle.Render(issue.ID)
	pri := ui.PriorityStyle(issue.Priority).Render(issue.PriorityString())
	status := ui.StatusStyle(issue.Status).Render(issue.Status)
	itype := ui.TypeStyle(issue.IssueType).Render(issue.IssueType)

	left := fmt.Sprintf("%s  %s  %s  %s", id, pri, status, itype)

	scrollInfo := ""
	if len(d.lines) > vis {
		pct := 0
		maxScroll := len(d.lines) - vis
		if maxScroll > 0 {
			pct = d.scroll * 100 / maxScroll
		}
		scrollInfo = ui.KeyDescStyle.Render(fmt.Sprintf("[%d%%]", pct))
	}

	gap := max(0, d.width-lipgloss.Width(left)-lipgloss.Width(scrollInfo)-2)
	header := left + strings.Repeat(" ", gap) + scrollInfo

	return ui.HeaderStyle.Width(d.width).Render(header)
}

func (d *DetailView) buildContent() {
	if d.issue == nil {
		d.lines = []string{"(no issue data)"}
		d.navItems = nil
		return
	}
	issue := d.issue
	contentWidth := max(20, d.width-4)

	var lines []string
	var navItems []navItem
	add := func(s string) {
		lines = append(lines, s)
	}
	addBlank := func() {
		lines = append(lines, "")
	}

	// Title
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorWhite)
	add(titleStyle.Render(issue.Title))
	addBlank()

	// Metadata fields
	field := func(label, value string) {
		if value == "" {
			return
		}
		add(ui.FieldLabelStyle.Render(label) + ui.FieldValueStyle.Render(value))
	}

	field("ID", issue.ID)
	field("Priority", ui.PriorityStyle(issue.Priority).Render(issue.PriorityString()))
	field("Status", ui.StatusStyle(issue.Status).Render(issue.Status))
	field("Type", ui.TypeStyle(issue.IssueType).Render(issue.IssueType))
	field("Assignee", issue.Assignee)
	field("Owner", issue.Owner)
	field("Created By", issue.CreatedBy)
	field("Created", issue.CreatedAt.Format("2006-01-02 15:04")+"  ("+models.RelativeAge(issue.CreatedAt)+" ago)")
	field("Updated", issue.UpdatedAt.Format("2006-01-02 15:04")+"  ("+models.RelativeAge(issue.UpdatedAt)+" ago)")
	if issue.ClosedAt != nil {
		field("Closed", issue.ClosedAt.Format("2006-01-02 15:04")+"  ("+models.RelativeAge(*issue.ClosedAt)+" ago)")
	}
	if issue.CloseReason != "" {
		field("Close Reason", issue.CloseReason)
	}
	if issue.DueAt != nil {
		field("Due", issue.DueAt.Format("2006-01-02 15:04"))
	}
	if issue.DeferUntil != nil {
		field("Defer Until", issue.DeferUntil.Format("2006-01-02 15:04"))
	}
	if issue.Parent != nil {
		field("Parent", *issue.Parent)
	}

	// Labels
	if len(issue.Labels) > 0 {
		field("Labels", strings.Join(issue.Labels, ", "))
	}

	// Description
	if issue.Description != "" {
		addBlank()
		add(ui.SectionHeaderStyle.Render("DESCRIPTION"))
		add(ui.TableHeaderStyle.Width(contentWidth).Render(""))
		for _, line := range wrapText(issue.Description, contentWidth) {
			add(line)
		}
	}

	// Design
	if issue.Design != "" {
		addBlank()
		add(ui.SectionHeaderStyle.Render("DESIGN"))
		add(ui.TableHeaderStyle.Width(contentWidth).Render(""))
		for _, line := range wrapText(issue.Design, contentWidth) {
			add(line)
		}
	}

	// Acceptance Criteria
	if issue.AcceptanceCriteria != "" {
		addBlank()
		add(ui.SectionHeaderStyle.Render("ACCEPTANCE CRITERIA"))
		add(ui.TableHeaderStyle.Width(contentWidth).Render(""))
		for _, line := range wrapText(issue.AcceptanceCriteria, contentWidth) {
			add(line)
		}
	}

	// Notes
	if issue.Notes != "" {
		addBlank()
		add(ui.SectionHeaderStyle.Render("NOTES"))
		add(ui.TableHeaderStyle.Width(contentWidth).Render(""))
		for _, line := range wrapText(issue.Notes, contentWidth) {
			add(line)
		}
	}

	// Dependencies
	if len(issue.Dependencies) > 0 {
		addBlank()
		add(ui.SectionHeaderStyle.Render(fmt.Sprintf("DEPENDENCIES (%d)", len(issue.Dependencies))))
		add(ui.TableHeaderStyle.Width(contentWidth).Render(""))
		for i, dep := range issue.Dependencies {
			prefix := "  ├─ "
			if i == len(issue.Dependencies)-1 {
				prefix = "  └─ "
			}
			depLine := fmt.Sprintf("%s%s [%s] %s  %s  %s",
				prefix,
				dep.ID,
				dep.DependencyType,
				dep.Title,
				ui.StatusStyle(dep.Status).Render(dep.Status),
				ui.PriorityStyle(dep.Priority).Render(dep.PriorityString()),
			)
			navItems = append(navItems, navItem{lineIndex: len(lines), issueID: dep.ID})
			add(depLine)
		}
	}

	// Dependents
	if len(issue.Dependents) > 0 {
		addBlank()
		add(ui.SectionHeaderStyle.Render(fmt.Sprintf("DEPENDENTS (%d)", len(issue.Dependents))))
		add(ui.TableHeaderStyle.Width(contentWidth).Render(""))
		for i, dep := range issue.Dependents {
			prefix := "  ├─ "
			if i == len(issue.Dependents)-1 {
				prefix = "  └─ "
			}
			depLine := fmt.Sprintf("%s%s [%s] %s  %s  %s",
				prefix,
				dep.ID,
				dep.DependencyType,
				dep.Title,
				ui.StatusStyle(dep.Status).Render(dep.Status),
				ui.PriorityStyle(dep.Priority).Render(dep.PriorityString()),
			)
			navItems = append(navItems, navItem{lineIndex: len(lines), issueID: dep.ID})
			add(depLine)
		}
	}

	// Comments
	if len(issue.Comments) > 0 {
		addBlank()
		add(ui.SectionHeaderStyle.Render(fmt.Sprintf("COMMENTS (%d)", len(issue.Comments))))
		add(ui.TableHeaderStyle.Width(contentWidth).Render(""))
		for _, c := range issue.Comments {
			author := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorCyan).Render(c.Author)
			age := models.RelativeAge(c.CreatedAt)
			add(fmt.Sprintf("  %s (%s ago):", author, age))
			for _, line := range wrapText(c.Text, contentWidth-4) {
				add("    " + line)
			}
			addBlank()
		}
	}

	d.lines = lines
	d.navItems = navItems
	// Clamp nav cursor
	if d.navCursor >= len(d.navItems) {
		d.navCursor = max(-1, len(d.navItems)-1)
	}
}

func (d *DetailView) renderContent() string {
	vis := d.visibleLines()
	end := min(d.scroll+vis, len(d.lines))
	start := d.scroll
	if start >= len(d.lines) {
		start = max(0, len(d.lines)-1)
	}

	// Build set of highlighted line indices.
	highlightLine := -1
	if d.navCursor >= 0 && d.navCursor < len(d.navItems) {
		highlightLine = d.navItems[d.navCursor].lineIndex
	}

	// Horizontal padding applied per-line to avoid lipgloss re-wrapping
	// content that's already been sized to contentWidth.
	pad := "  "
	visible := make([]string, 0, vis)
	for i := start; i < end; i++ {
		line := d.lines[i]
		if i == highlightLine {
			line = ui.SelectedRowStyle.Width(d.width - 4).Render(line)
		}
		visible = append(visible, pad+line)
	}
	// Pad remaining space with empty lines.
	for len(visible) < vis {
		visible = append(visible, "")
	}
	return strings.Join(visible, "\n")
}

func (d *DetailView) renderStatusBar() string {
	if d.statusMsg != "" {
		return ui.StatusBarStyle.Width(d.width).Render(
			lipgloss.NewStyle().Foreground(ui.ColorGreen).Render(d.statusMsg),
		)
	}

	keys := []struct{ key, desc string }{
		{"esc", "back"},
		{"j/k", "scroll"},
		{"tab", "next dep"},
		{"enter", "drill in"},
		{"g/G", "top/bottom"},
		{"r", "refresh"},
		{"y", "copy ID"},
		{"?", "help"},
		{"q", "quit"},
	}
	var parts []string
	for _, k := range keys {
		parts = append(parts, ui.KeyStyle.Render(k.key)+" "+ui.KeyDescStyle.Render(k.desc))
	}
	bar := strings.Join(parts, "  ")
	return ui.StatusBarStyle.Width(d.width).Render(bar)
}

// wrapText splits text into lines, preserving existing newlines
// and wrapping long lines at word boundaries.
func wrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		maxWidth = 80
	}
	var result []string
	for _, paragraph := range strings.Split(text, "\n") {
		if paragraph == "" {
			result = append(result, "")
			continue
		}
		result = append(result, wrapLine(paragraph, maxWidth)...)
	}
	return result
}

func wrapLine(line string, maxWidth int) []string {
	if ui.StringWidth(line) <= maxWidth {
		return []string{line}
	}
	var result []string
	words := strings.Fields(line)
	current := ""
	currentWidth := 0
	for _, word := range words {
		wordWidth := ui.StringWidth(word)
		if current == "" {
			current = word
			currentWidth = wordWidth
		} else if currentWidth+1+wordWidth <= maxWidth {
			current += " " + word
			currentWidth += 1 + wordWidth
		} else {
			result = append(result, current)
			current = word
			currentWidth = wordWidth
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
