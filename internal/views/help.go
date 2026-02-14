package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/poiley/beady/internal/ui"
)

// HelpView renders a help overlay.
type HelpView struct {
	width  int
	height int
}

// NewHelpView creates a help overlay.
func NewHelpView() *HelpView {
	return &HelpView{}
}

// SetSize sets terminal dimensions.
func (h *HelpView) SetSize(w, ht int) {
	h.width = w
	h.height = ht
}

// View renders the help overlay.
func (h *HelpView) View() string {
	title := ui.HelpTitleStyle.Render("Keybindings")

	sections := []struct {
		header string
		keys   []struct{ key, desc string }
	}{
		{
			header: "Navigation",
			keys: []struct{ key, desc string }{
				{"j / k / arrows", "Move up / down"},
				{"g / G", "Jump to top / bottom"},
				{"Ctrl+u / Ctrl+d", "Page up / down"},
				{"Enter", "Open issue detail / drill into dependency"},
				{"Tab / Shift+Tab", "Cycle through dependencies (detail view)"},
				{"Esc", "Back / cancel"},
			},
		},
		{
			header: "Sorting",
			keys: []struct{ key, desc string }{
				{"s", "Cycle sort: priority > created > updated > status > type > id"},
				{"S", "Reverse sort direction"},
			},
		},
		{
			header: "Filtering",
			keys: []struct{ key, desc string }{
				{"/", "Start text search (filters on title, ID, type, assignee, labels)"},
				{"Esc", "Cancel search (restore previous)"},
				{"1", "Toggle: open issues only"},
				{"2", "Toggle: in_progress only"},
				{"3", "Toggle: blocked only"},
				{"4", "Toggle: closed only"},
				{"5", "Toggle: ready (unblocked) only"},
				{"6", "Toggle: deferred only"},
				{"7", "Toggle: pinned only"},
				{"0", "Show all statuses"},
				{"c", "Toggle: show/hide closed issues (hidden by default)"},
			},
		},
		{
			header: "Detail View",
			keys: []struct{ key, desc string }{
				{"x", "Collapse / expand section under cursor"},
			},
		},
		{
			header: "Actions",
			keys: []struct{ key, desc string }{
				{"r", "Refresh data from bd"},
				{"y", "Copy issue ID to clipboard"},
				{"?", "Toggle this help screen"},
				{"q", "Quit"},
			},
		},
	}

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")

	for _, section := range sections {
		header := lipgloss.NewStyle().Bold(true).Foreground(ui.ColorCyan).Render(section.header)
		b.WriteString(header)
		b.WriteString("\n")
		for _, k := range section.keys {
			line := ui.HelpKeyStyle.Render(k.key) + " " + ui.HelpDescStyle.Render(k.desc)
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	content := b.String()

	// Center in a box
	boxWidth := min(70, h.width-4)
	boxHeight := min(strings.Count(content, "\n")+4, h.height-4)

	box := lipgloss.NewStyle().
		Width(boxWidth).
		Height(boxHeight).
		Padding(1, 2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorBlue).
		Background(ui.ColorBg).
		Render(content)

	// Center the box
	return lipgloss.Place(
		h.width, h.height,
		lipgloss.Center, lipgloss.Center,
		box,
	)
}
