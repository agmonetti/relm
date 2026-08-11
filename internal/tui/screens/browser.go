package screens

import (
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/tui/styles"
)

// RenderSidebar renders the table list with a selection cursor and the opened
// table marked. It scrolls vertically so the cursor stays visible.
func RenderSidebar(b *browser.Browser, cursor, width, height int) string {
	if width < 10 || height < 1 {
		return ""
	}
	n := len(b.Tables)
	if n == 0 {
		return ""
	}
	if cursor >= n {
		cursor = n - 1
	}
	if cursor < 0 {
		cursor = 0
	}
	offset := 0
	if cursor >= height {
		offset = cursor
	}

	var sb strings.Builder
	for i := offset; i < n && i < offset+height; i++ {
		name := truncate(b.Tables[i], width-2)
		switch {
		case i == cursor:
			sb.WriteString(styles.StyleSidebarActive.Render("> " + name))
		case b.Tables[i] == b.ActiveTable:
			sb.WriteString(styles.StyleSidebarActiveTable.Render("  " + name))
		default:
			sb.WriteString(styles.StyleSidebarItem.Render("  " + name))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// RenderDataTable renders columns + rows with the cursor row highlighted.
func RenderDataTable(cols []string, rows [][]string, cursor, width, height int) string {
	if len(cols) == 0 {
		return ""
	}

	widths := colWidths(cols, rows, width)
	visible := height - 1 // leaves room for the header
	if visible > len(rows) {
		visible = len(rows)
	}
	if visible < 0 {
		visible = 0
	}

	var sb strings.Builder
	// header
	sb.WriteString(renderHeader(cols, widths) + "\n")

	for i := 0; i < visible; i++ {
		sb.WriteString(renderRow(rows[i], widths, i == cursor) + "\n")
	}
	return sb.String()
}

// renderHeader renders the column header row with the column header style.
func renderHeader(cells []string, widths []int) string {
	var sb strings.Builder
	for i, c := range cells {
		if i >= len(widths) {
			break
		}
		sb.WriteString(styles.StyleColHeader.Render(pad(truncate(c, widths[i]), widths[i])))
		if i < len(cells)-1 {
			sb.WriteString(" ")
		}
	}
	return sb.String()
}

// colWidths computes the width of each column based on the visible content.
func colWidths(cols []string, rows [][]string, width int) []int {
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = runewidth.StringWidth(c)
	}
	// consider up to 100 rows so wide tables stay cheap
	maxRows := len(rows)
	if maxRows > 100 {
		maxRows = 100
	}
	for r := 0; r < maxRows; r++ {
		for i, cell := range rows[r] {
			if i >= len(widths) {
				break
			}
			if w := runewidth.StringWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	// adjust to the available width: shrink wide columns first
	total := len(widths) - 1 // separators
	for _, w := range widths {
		total += w
	}
	if total > width {
		for i := range widths {
			over := total - width
			if over <= 0 {
				break
			}
			shrink := widths[i] - 4 // reasonable minimum
			if shrink > over {
				shrink = over
			}
			if shrink < 0 {
				shrink = 0
			}
			widths[i] -= shrink
			total -= shrink
		}
	}
	return widths
}

// renderRow renders a row. selected highlights the whole row.
func renderRow(cells []string, widths []int, selected bool) string {
	var sb strings.Builder
	for i, cell := range cells {
		if i >= len(widths) {
			break
		}
		text := cell
		if text == "" {
			text = styles.NullCell()
		} else {
			text = truncate(text, widths[i])
		}
		text = pad(text, widths[i])
		if selected {
			text = styles.StyleCursor.Render(text)
		}
		sb.WriteString(text)
		if i < len(cells)-1 {
			sb.WriteString(" ")
		}
	}
	return sb.String()
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= max {
		return s
	}
	trimmed := s
	for runewidth.StringWidth(trimmed) > max-1 {
		trimmed = trimLastRune(trimmed)
	}
	return trimmed + "…"
}

func trimLastRune(s string) string {
	if s == "" {
		return ""
	}
	r := []rune(s)
	return string(r[:len(r)-1])
}

func pad(s string, w int) string {
	if d := w - runewidth.StringWidth(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}
