package ui

import "github.com/charmbracelet/lipgloss"

// k9s-inspired color palette
var (
	ColorBlue     = lipgloss.Color("#4FC3F7")
	ColorDarkBlue = lipgloss.Color("#1565C0")
	ColorGreen    = lipgloss.Color("#66BB6A")
	ColorYellow   = lipgloss.Color("#FFD54F")
	ColorRed      = lipgloss.Color("#EF5350")
	ColorMagenta  = lipgloss.Color("#CE93D8")
	ColorCyan     = lipgloss.Color("#4DD0E1")
	ColorGray     = lipgloss.Color("#757575")
	ColorDimGray  = lipgloss.Color("#424242")
	ColorWhite    = lipgloss.Color("#EEEEEE")
	ColorBg       = lipgloss.Color("#1A1A2E")
	ColorHeaderBg = lipgloss.Color("#16213E")
	ColorSelectBg = lipgloss.Color("#0F3460")
	ColorFlashBg  = lipgloss.Color("#3E3516") // subtle gold tint for changed rows
	ColorBorder   = lipgloss.Color("#2C3E6D")
)

// Reusable styles
var (
	// Header / title bar
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBlue).
			Background(ColorHeaderBg).
			Padding(0, 1)

	// Logo style
	LogoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBlue)

	// Table header row
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorBlue).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(ColorBorder)

	// Normal row
	RowStyle = lipgloss.NewStyle().
			Foreground(ColorWhite)

	// Selected row
	SelectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Background(ColorSelectBg).
				Foreground(ColorWhite)

	// Flash row (recently changed, k9s-style pulse)
	FlashRowStyle = lipgloss.NewStyle().
			Background(ColorFlashBg).
			Foreground(ColorYellow)

	// Status bar at bottom
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorGray).
			Background(ColorHeaderBg).
			Padding(0, 1)

	// Status bar key hints
	KeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBlue)

	KeyDescStyle = lipgloss.NewStyle().
			Foreground(ColorGray)

	// Detail view section headers
	SectionHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorBlue).
				MarginTop(1)

	// Detail view field labels
	FieldLabelStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorCyan).
			Width(14)

	FieldValueStyle = lipgloss.NewStyle().
			Foreground(ColorWhite)

	// Help overlay
	HelpTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorBlue).
			MarginBottom(1)

	HelpKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorYellow).
			Width(14)

	HelpDescStyle = lipgloss.NewStyle().
			Foreground(ColorWhite)

	// Border style for panels
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder)

	// Error style
	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorRed)

	// Filter input
	FilterPromptStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorYellow)

	FilterInputStyle = lipgloss.NewStyle().
				Foreground(ColorWhite)
)

// PriorityStyle returns a style colored by priority level.
func PriorityStyle(priority int) lipgloss.Style {
	switch priority {
	case 0:
		return lipgloss.NewStyle().Bold(true).Foreground(ColorRed)
	case 1:
		return lipgloss.NewStyle().Foreground(ColorYellow)
	case 2:
		return lipgloss.NewStyle().Foreground(ColorWhite)
	case 3:
		return lipgloss.NewStyle().Foreground(ColorGray)
	default:
		return lipgloss.NewStyle().Foreground(ColorDimGray)
	}
}

// StatusStyle returns a style colored by status.
func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "open":
		return lipgloss.NewStyle().Foreground(ColorGreen)
	case "in_progress":
		return lipgloss.NewStyle().Foreground(ColorCyan)
	case "blocked":
		return lipgloss.NewStyle().Bold(true).Foreground(ColorRed)
	case "deferred":
		return lipgloss.NewStyle().Foreground(ColorMagenta)
	case "closed":
		return lipgloss.NewStyle().Foreground(ColorGray)
	case "pinned":
		return lipgloss.NewStyle().Foreground(ColorYellow)
	default:
		return lipgloss.NewStyle().Foreground(ColorDimGray)
	}
}

// TypeStyle returns a style colored by issue type.
func TypeStyle(issueType string) lipgloss.Style {
	switch issueType {
	case "epic":
		return lipgloss.NewStyle().Bold(true).Foreground(ColorMagenta)
	case "bug":
		return lipgloss.NewStyle().Foreground(ColorRed)
	case "feature":
		return lipgloss.NewStyle().Foreground(ColorGreen)
	case "task":
		return lipgloss.NewStyle().Foreground(ColorCyan)
	case "chore":
		return lipgloss.NewStyle().Foreground(ColorGray)
	default:
		return lipgloss.NewStyle().Foreground(ColorWhite)
	}
}
