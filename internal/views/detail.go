package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/poiley/beady/internal/models"
	"github.com/poiley/beady/internal/ui"
)

// NavigateToIssueMsg requests the app to load a detail view for the given ID.
type NavigateToIssueMsg struct {
	ID string
}

// sectionKind identifies a collapsible section in the detail view.
type sectionKind int

const (
	sectionDescription sectionKind = iota
	sectionDesign
	sectionAcceptance
	sectionNotes
	sectionDeps
	sectionDependents
	sectionComments
	sectionCount // sentinel — total number of section kinds
)

// navItem maps a rendered line to a navigable issue link (for drill-down).
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

	// Navigation within parent/deps/dependents (issue links only).
	navItems  []navItem // navigable lines
	navCursor int       // index into navItems, -1 = none selected

	// Section cursor for collapse/expand (separate from nav cursor).
	// sections lists the sectionKinds present in the current content.
	// sectionLines[i] is the line index of sections[i]'s header.
	sections      []sectionKind
	sectionLines  []int
	sectionCursor int // index into sections, 0 = first section

	// Breadcrumb trail (set by app when drilling into deps).
	breadcrumbs []string // issue IDs from root to current (exclusive)

	// Collapsible sections.
	collapsed [sectionCount]bool
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

// SetBreadcrumbs sets the navigation trail shown in the header.
func (d *DetailView) SetBreadcrumbs(crumbs []string) {
	d.breadcrumbs = crumbs
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
		case "x":
			d.toggleSection()
		case "]":
			d.moveSectionCursor(1)
		case "[":
			d.moveSectionCursor(-1)
		}
	}
	return nil
}

// toggleSection toggles collapse on the section under the section cursor.
func (d *DetailView) toggleSection() {
	if len(d.sections) == 0 {
		return
	}
	if d.sectionCursor < 0 || d.sectionCursor >= len(d.sections) {
		d.sectionCursor = 0
	}

	sk := d.sections[d.sectionCursor]
	d.collapsed[sk] = !d.collapsed[sk]
	d.buildContent()

	// Scroll so the toggled section header is visible.
	d.scrollToSection()
}

// moveSectionCursor moves the section cursor by delta (+1 or -1) and scrolls
// to make the new section header visible.
func (d *DetailView) moveSectionCursor(delta int) {
	if len(d.sections) == 0 {
		return
	}
	d.sectionCursor += delta
	if d.sectionCursor >= len(d.sections) {
		d.sectionCursor = 0
	}
	if d.sectionCursor < 0 {
		d.sectionCursor = len(d.sections) - 1
	}
	d.scrollToSection()
}

// scrollToSection scrolls the viewport so the current section header is visible.
func (d *DetailView) scrollToSection() {
	if d.sectionCursor < 0 || d.sectionCursor >= len(d.sectionLines) {
		return
	}
	line := d.sectionLines[d.sectionCursor]
	vis := d.visibleLines()
	if line < d.scroll {
		d.scroll = line
	} else if line >= d.scroll+vis {
		d.scroll = line - vis + 1
	}
	maxScroll := max(0, len(d.lines)-d.visibleLines())
	if d.scroll > maxScroll {
		d.scroll = maxScroll
	}
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

	// Breadcrumb trail: show navigation path when drilled into deps.
	if len(d.breadcrumbs) > 0 {
		crumbStyle := lipgloss.NewStyle().Foreground(ui.ColorGray)
		sepStyle := lipgloss.NewStyle().Foreground(ui.ColorDimGray)
		var crumbs []string
		for _, c := range d.breadcrumbs {
			crumbs = append(crumbs, crumbStyle.Render(c))
		}
		crumbs = append(crumbs, ui.LogoStyle.Render(issue.ID))
		trail := strings.Join(crumbs, sepStyle.Render(" > "))
		left = trail + "  " + pri + "  " + status + "  " + itype
	}

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
		d.sections = nil
		d.sectionLines = nil
		return
	}
	issue := d.issue
	contentWidth := max(20, d.width-4)

	var lines []string
	var navItems []navItem
	var sections []sectionKind
	var sectionLines []int

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
	if est := issue.EstimateString(); est != "" {
		field("Estimate", est)
	}
	if lt := issue.LeadTime(); lt > 0 {
		field("Lead Time", formatDuration(lt))
	}

	// Parent — navigable link to drill up the hierarchy.
	if issue.Parent != nil {
		parentLine := ui.FieldLabelStyle.Render("Parent") +
			lipgloss.NewStyle().Foreground(ui.ColorBlue).Bold(true).Render(*issue.Parent) +
			lipgloss.NewStyle().Foreground(ui.ColorDimGray).Render("  (enter to view)")
		navItems = append(navItems, navItem{lineIndex: len(lines), issueID: *issue.Parent})
		add(parentLine)
	}

	// Labels
	if len(issue.Labels) > 0 {
		field("Labels", strings.Join(issue.Labels, ", "))
	}

	// --- Collapsible sections ---

	// Helper to add a section header + divider. Registers the section
	// in the sections list so the section cursor can navigate to it.
	addSectionHeader := func(sk sectionKind, title string) {
		addBlank()
		indicator := d.collapseIndicator(sk)
		sections = append(sections, sk)
		sectionLines = append(sectionLines, len(lines))
		add(ui.SectionHeaderStyle.Render(fmt.Sprintf("%s %s", indicator, title)))
		add(ui.TableHeaderStyle.Width(contentWidth).Render(""))
	}

	// Description
	if issue.Description != "" {
		addSectionHeader(sectionDescription, "DESCRIPTION")
		if !d.collapsed[sectionDescription] {
			for _, line := range wrapText(issue.Description, contentWidth) {
				add(line)
			}
		}
	}

	// Design
	if issue.Design != "" {
		addSectionHeader(sectionDesign, "DESIGN")
		if !d.collapsed[sectionDesign] {
			for _, line := range wrapText(issue.Design, contentWidth) {
				add(line)
			}
		}
	}

	// Acceptance Criteria
	if issue.AcceptanceCriteria != "" {
		addSectionHeader(sectionAcceptance, "ACCEPTANCE CRITERIA")
		if !d.collapsed[sectionAcceptance] {
			for _, line := range wrapText(issue.AcceptanceCriteria, contentWidth) {
				add(line)
			}
		}
	}

	// Notes
	if issue.Notes != "" {
		addSectionHeader(sectionNotes, "NOTES")
		if !d.collapsed[sectionNotes] {
			for _, line := range wrapText(issue.Notes, contentWidth) {
				add(line)
			}
		}
	}

	// Dependencies (excluding parent-child, since parent is shown above)
	nonParentDeps := d.nonParentDeps()
	if len(nonParentDeps) > 0 {
		addSectionHeader(sectionDeps, fmt.Sprintf("DEPENDENCIES (%d)", len(nonParentDeps)))
		if !d.collapsed[sectionDeps] {
			for i, dep := range nonParentDeps {
				prefix := "  ├─ "
				if i == len(nonParentDeps)-1 {
					prefix = "  └─ "
				}
				depLine := fmt.Sprintf("%s%s  %s  %s  %s",
					prefix,
					dep.ID,
					ui.StatusBadge(dep.Status),
					ui.PriorityStyle(dep.Priority).Render(dep.PriorityString()),
					dep.Title,
				)
				navItems = append(navItems, navItem{lineIndex: len(lines), issueID: dep.ID})
				add(depLine)
			}
		}
	}

	// Dependents
	if len(issue.Dependents) > 0 {
		addSectionHeader(sectionDependents, fmt.Sprintf("DEPENDENTS (%d)", len(issue.Dependents)))
		if !d.collapsed[sectionDependents] {
			for i, dep := range issue.Dependents {
				prefix := "  ├─ "
				if i == len(issue.Dependents)-1 {
					prefix = "  └─ "
				}
				depLine := fmt.Sprintf("%s%s  %s  %s  %s",
					prefix,
					dep.ID,
					ui.StatusBadge(dep.Status),
					ui.PriorityStyle(dep.Priority).Render(dep.PriorityString()),
					dep.Title,
				)
				navItems = append(navItems, navItem{lineIndex: len(lines), issueID: dep.ID})
				add(depLine)
			}
		}
	}

	// Comments
	if len(issue.Comments) > 0 {
		addSectionHeader(sectionComments, fmt.Sprintf("COMMENTS (%d)", len(issue.Comments)))
		if !d.collapsed[sectionComments] {
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
	}

	d.lines = lines
	d.sections = sections
	d.sectionLines = sectionLines
	d.navItems = navItems
	// Clamp nav cursor
	if d.navCursor >= len(d.navItems) {
		d.navCursor = max(-1, len(d.navItems)-1)
	}
	// Clamp section cursor
	if len(d.sections) > 0 {
		if d.sectionCursor >= len(d.sections) {
			d.sectionCursor = len(d.sections) - 1
		}
	} else {
		d.sectionCursor = 0
	}
}

// collapseIndicator returns a visual indicator for collapsed/expanded state.
func (d *DetailView) collapseIndicator(kind sectionKind) string {
	if d.collapsed[kind] {
		return lipgloss.NewStyle().Foreground(ui.ColorGray).Render("▶")
	}
	return lipgloss.NewStyle().Foreground(ui.ColorGray).Render("▼")
}

// nonParentDeps returns dependencies that aren't the parent-child link
// (since parent is shown separately as a navigable field).
func (d *DetailView) nonParentDeps() []*models.IssueWithDepType {
	if d.issue == nil {
		return nil
	}
	var result []*models.IssueWithDepType
	for _, dep := range d.issue.Dependencies {
		if dep.DependencyType == "parent-child" && d.issue.Parent != nil && dep.ID == *d.issue.Parent {
			continue
		}
		result = append(result, dep)
	}
	return result
}

func (d *DetailView) renderContent() string {
	vis := d.visibleLines()
	end := min(d.scroll+vis, len(d.lines))
	start := d.scroll
	if start >= len(d.lines) {
		start = max(0, len(d.lines)-1)
	}

	// Build set of highlighted line indices.
	navHighlight := -1
	if d.navCursor >= 0 && d.navCursor < len(d.navItems) {
		navHighlight = d.navItems[d.navCursor].lineIndex
	}
	sectionHighlight := -1
	if d.sectionCursor >= 0 && d.sectionCursor < len(d.sectionLines) {
		sectionHighlight = d.sectionLines[d.sectionCursor]
	}

	// Horizontal padding applied per-line to avoid lipgloss re-wrapping
	// content that's already been sized to contentWidth.
	pad := "  "
	visible := make([]string, 0, vis)
	for i := start; i < end; i++ {
		line := d.lines[i]
		if i == navHighlight {
			line = ui.SelectedRowStyle.Width(d.width - 4).Render(line)
		} else if i == sectionHighlight {
			line = ui.SectionCursorStyle.Width(d.width - 4).Render(line)
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
		{"[/]", "section"},
		{"x", "collapse"},
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

// formatDuration formats a duration as a human-readable string like "2d 4h".
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	if hours < 1 {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if hours < 24 {
		return fmt.Sprintf("%dh", hours)
	}
	days := hours / 24
	rh := hours % 24
	if rh > 0 {
		return fmt.Sprintf("%dd %dh", days, rh)
	}
	return fmt.Sprintf("%dd", days)
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
