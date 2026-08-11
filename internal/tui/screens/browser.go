package screens

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/tui/styles"
)

// RenderBrowser renders the main browser content.
func RenderBrowser(b *browser.Browser, width, height int, showSidebar bool) string {
	if b == nil {
		return styles.StyleHeaderDim.Render("no connection")
	}

	cols := make([]string, len(b.Columns))
	for i, c := range b.Columns {
		cols[i] = c.Name
	}
	content := renderDataTable(cols, b.Rows, b.Cursor, width, height)

	if len(b.Tables) == 0 {
		content = styles.StyleHeaderDim.Render("no tables — use the editor to create one")
	} else if len(b.Rows) == 0 && b.ActiveTable != "" {
		content = styles.StyleHeaderDim.Render("empty table")
	}

	if showSidebar {
		return lipgloss.JoinHorizontal(
			lipgloss.Top,
			renderSidebar(b, width/4),
			"  ",
			content,
		)
	}
	return content
}

func renderSidebar(b *browser.Browser, width int) string {
	if width < 10 {
		return ""
	}
	var sb strings.Builder
	for _, t := range b.Tables {
		name := truncate(t, width-2)
		if t == b.ActiveTable {
			sb.WriteString(styles.StyleSidebarActive.Render("> " + name))
		} else {
			sb.WriteString(styles.StyleSidebarItem.Render("  " + name))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// renderDataTable renders columns + rows with the cursor row highlighted.
func renderDataTable(cols []string, rows [][]string, cursor, width, height int) string {
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
	sb.WriteString(renderRow(cols, widths, -1) + "\n")

	for i := 0; i < visible; i++ {
		sb.WriteString(renderRow(rows[i], widths, cursor) + "\n")
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

// renderRow renders a row. cursor == idx highlights the row.
func renderRow(cells []string, widths []int, cursor int) string {
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
		if i == cursor {
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
