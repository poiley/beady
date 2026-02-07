package ui

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// Align controls text alignment within a column.
type Align int

const (
	AlignLeft Align = iota
	AlignRight
)

// SizeMode controls how a column determines its width.
type SizeMode int

const (
	// SizeFixed uses exactly Column.Fixed characters.
	SizeFixed SizeMode = iota

	// SizeFit scans data to find the widest value, clamped to [Min, Max].
	SizeFit

	// SizeFlex absorbs remaining terminal width after all other columns
	// are resolved. If multiple flex columns exist, they split the space.
	// Clamped to [Min, Max] if set.
	SizeFlex
)

// Column defines a table column.
type Column struct {
	Header string
	Size   SizeMode
	Align  Align
	Fixed  int // used when Size == SizeFixed
	Min    int // minimum width (Fit and Flex)
	Max    int // maximum width (Fit and Flex); 0 means unlimited

	// resolved width, set by Resolve()
	Width int
}

// Table is a generic column layout engine.
type Table struct {
	Columns []*Column
	Gap     int // space between columns (default 1)
}

// NewTable creates a table with the given columns and a default gap of 1.
func NewTable(cols ...*Column) *Table {
	return &Table{Columns: cols, Gap: 1}
}

// Resolve computes the Width for every column given the available terminal
// width and per-column data widths.
//
// dataWidths[i] is the width of the widest value in column i (only used for
// SizeFit columns; ignored for Fixed and Flex).
func (t *Table) Resolve(totalWidth int, dataWidths []int) {
	gap := t.Gap
	n := len(t.Columns)
	if n == 0 {
		return
	}

	// Pass 1: resolve Fixed and Fit columns.
	usedWidth := 0
	flexCount := 0
	for i, col := range t.Columns {
		switch col.Size {
		case SizeFixed:
			col.Width = col.Fixed
		case SizeFit:
			w := 0
			if i < len(dataWidths) {
				w = dataWidths[i]
			}
			// Also account for header text display width
			if hw := runewidth.StringWidth(col.Header); hw > w {
				w = hw
			}
			if col.Min > 0 && w < col.Min {
				w = col.Min
			}
			if col.Max > 0 && w > col.Max {
				w = col.Max
			}
			col.Width = w
		case SizeFlex:
			flexCount++
			continue // defer to pass 2
		}
		usedWidth += col.Width
	}

	// Account for gaps between columns
	totalGaps := (n - 1) * gap
	remaining := totalWidth - usedWidth - totalGaps

	// Pass 2: distribute remaining space to Flex columns.
	if flexCount > 0 && remaining > 0 {
		perFlex := remaining / flexCount
		extra := remaining % flexCount
		for _, col := range t.Columns {
			if col.Size != SizeFlex {
				continue
			}
			w := perFlex
			if extra > 0 {
				w++
				extra--
			}
			if col.Min > 0 && w < col.Min {
				w = col.Min
			}
			if col.Max > 0 && w > col.Max {
				w = col.Max
			}
			col.Width = w
		}
	} else if flexCount > 0 {
		// No space left — give flex columns their minimum.
		for _, col := range t.Columns {
			if col.Size != SizeFlex {
				continue
			}
			col.Width = col.Min
		}
	}
}

// RenderRow renders a single row of cell values into a fixed-width string.
// It pads/truncates each cell to the resolved column width and applies
// column alignment. The returned string has no trailing newline.
//
// If the last N columns are all AlignRight, they are treated as a
// right-justified group: the preceding (left) columns are rendered
// at their natural widths, a dynamic gap fills the remaining space,
// and the right group is rendered flush to the right edge. This
// prevents a Flex column from pushing right-aligned columns off-screen.
//
// styleFn is called per-cell to wrap the already-padded text in ANSI styles.
// Pass nil to skip styling.
func (t *Table) RenderRow(cells []string, styleFn func(col int, padded string) string) string {
	n := len(t.Columns)
	if n == 0 {
		return ""
	}

	// Find where the trailing right-aligned group starts.
	rightStart := n // no right group by default
	for i := n - 1; i >= 0; i-- {
		if t.Columns[i].Align == AlignRight {
			rightStart = i
		} else {
			break
		}
	}

	var left strings.Builder
	leftWidth := 0 // visible chars written in left section

	// Render left columns (up to rightStart).
	for i := 0; i < rightStart; i++ {
		col := t.Columns[i]
		if i > 0 {
			left.WriteString(strings.Repeat(" ", t.Gap))
			leftWidth += t.Gap
		}
		val := ""
		if i < len(cells) {
			val = cells[i]
		}
		val = Truncate(val, col.Width)
		val = PadStr(val, col.Width)
		if styleFn != nil {
			val = styleFn(i, val)
		}
		left.WriteString(val)
		leftWidth += col.Width
	}

	// If there are no right-aligned columns, we're done.
	if rightStart == n {
		return left.String()
	}

	// Render right-aligned columns into a separate buffer.
	var right strings.Builder
	rightWidth := 0
	for i := rightStart; i < n; i++ {
		col := t.Columns[i]
		if i > rightStart {
			right.WriteString(strings.Repeat(" ", t.Gap))
			rightWidth += t.Gap
		}
		val := ""
		if i < len(cells) {
			val = cells[i]
		}
		val = Truncate(val, col.Width)
		val = PadLeft(val, col.Width)
		if styleFn != nil {
			val = styleFn(i, val)
		}
		right.WriteString(val)
		rightWidth += col.Width
	}

	// Compute the total width from Resolve to determine the gap.
	totalResolved := 0
	for _, col := range t.Columns {
		totalResolved += col.Width
	}
	totalResolved += (n - 1) * t.Gap

	// The gap between left and right sections absorbs remaining space.
	gap := totalResolved - leftWidth - rightWidth
	if gap < t.Gap {
		gap = t.Gap
	}

	var b strings.Builder
	b.WriteString(left.String())
	b.WriteString(strings.Repeat(" ", gap))
	b.WriteString(right.String())
	return b.String()
}

// StringWidth returns the visible display width of s, handling wide
// characters (CJK, em-dash, etc.) correctly.
func StringWidth(s string) int {
	return runewidth.StringWidth(s)
}

// Truncate cuts a string to maxLen visible display columns, adding "..."
// if truncated. It operates on rune boundaries and accounts for wide chars.
func Truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= maxLen {
		return s
	}
	return runewidth.Truncate(s, maxLen, "...")
}

// PadStr left-aligns s within width by appending spaces.
// Width is measured in visible display columns.
func PadStr(s string, width int) string {
	sw := runewidth.StringWidth(s)
	if sw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-sw)
}

// PadLeft right-aligns s within width by prepending spaces.
// Width is measured in visible display columns.
func PadLeft(s string, width int) string {
	sw := runewidth.StringWidth(s)
	if sw >= width {
		return s
	}
	return strings.Repeat(" ", width-sw) + s
}
