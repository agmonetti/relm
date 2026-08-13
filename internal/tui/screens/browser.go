package screens

import (
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/tui/styles"
)

// SidebarWindow returns the visible window of the sidebar list: `offset` is
// the index of the first shown table and `visible` how many are shown.
func SidebarWindow(cursor, height int) (offset, visible int) {
	offset = 0
	if cursor >= height {
		offset = cursor
	}
	visible = height
	return
}

// RenderSidebar renders the table list with a selection cursor and the opened
// table marked. It scrolls vertically so the cursor stays visible.
func RenderSidebar(b *browser.Browser, cursor, width, height int) string {
	if b == nil || width < 10 || height < 1 {
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
	offset, _ := SidebarWindow(cursor, height)

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

// TableWindow returns the visible window of a data table: `start` is the index
// of the first visible row and `visible` the number of rows shown, keeping the
// cursor row on screen.
func TableWindow(rows, cursor, height int) (start, visible int) {
	visible = height - 1 // leaves room for the header
	if visible > rows {
		visible = rows
	}
	if visible < 0 {
		visible = 0
	}
	start = 0
	if cursor >= visible {
		start = cursor - visible + 1
	}
	return
}

// RenderDataTable renders columns + rows with the cursor row highlighted. The
// rows scroll vertically so the cursor row always stays visible.
func RenderDataTable(cols []string, rows [][]string, cursor, width, height int) string {
	if len(cols) == 0 {
		return ""
	}

	widths := colWidths(cols, rows, width)
	start, visible := TableWindow(len(rows), cursor, height)

	var sb strings.Builder
	// header
	sb.WriteString(renderHeader(cols, widths) + "\n")

	for i := 0; i < visible; i++ {
		row := start + i
		if row >= len(rows) {
			break
		}
		sb.WriteString(renderRow(rows[row], widths, row == cursor) + "\n")
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

// colMaxW caps the width of any single column so one very long value does not
// starve the rest of the table. The full value is available in the detail view.
const colMaxW = 36

// colMinW is the smallest width a column is allowed to shrink to.
const colMinW = 4

// colWidths computes the width of each column based on the visible content. A
// column is capped at colMaxW and, when the row overflows the pane, the excess
// is distributed proportionally to each column's headroom instead of shrinking
// the first columns to the floor.
func colWidths(cols []string, rows [][]string, width int) []int {
	n := len(cols)
	widths := make([]int, n)
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
			if i >= n {
				break
			}
			if w := runewidth.StringWidth(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}
	// cap wide columns so a single long value does not blow the layout
	for i := range widths {
		if widths[i] > colMaxW {
			widths[i] = colMaxW
		}
	}

	total := n - 1 // separators
	for _, w := range widths {
		total += w
	}
	if total <= width {
		return widths
	}

	// shrink proportionally to the headroom of each column (above the floor),
	// then absorb any remainder greedily
	over := total - width
	giveable := 0
	for _, w := range widths {
		if g := w - colMinW; g > 0 {
			giveable += g
		}
	}
	if giveable > 0 {
		remaining := over
		for i := range widths {
			if remaining <= 0 {
				break
			}
			g := widths[i] - colMinW
			if g <= 0 {
				continue
			}
			share := g * over / giveable
			if share > g {
				share = g
			}
			if share > remaining {
				share = remaining
			}
			if share < 0 {
				share = 0
			}
			widths[i] -= share
			remaining -= share
		}
		// absorb any remainder greedily without going below the floor
		for remaining > 0 {
			progress := false
			for i := range widths {
				if remaining <= 0 {
					break
				}
				shrink := widths[i] - colMinW
				if shrink > remaining {
					shrink = remaining
				}
				if shrink < 0 {
					shrink = 0
				}
				if shrink > 0 {
					widths[i] -= shrink
					remaining -= shrink
					progress = true
				}
			}
			if !progress {
				break
			}
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

// RenderRowDetail renders a single row with every column and its full value,
// wrapped to the pane width and scrollable. Used by the detail view ("v").
func RenderRowDetail(title string, cols, vals []string, scroll, width, height int) string {
	if width < 4 {
		width = 40
	}
	contentW := width - 4 // border (2) + padding (2)
	if contentW < 4 {
		contentW = 4
	}

	var b strings.Builder
	b.WriteString(styles.StyleHeader.Render(title))
	b.WriteString("\n")
	if len(cols) == 0 {
		b.WriteString(styles.StyleHeaderDim.Render("  no columns"))
		return b.String()
	}

	var lines []string
	lines = append(lines, "")
	for i := range cols {
		name := cols[i]
		val := ""
		if i < len(vals) {
			val = vals[i]
		}
		if val == "" {
			val = styles.NullCell()
		}
		lines = append(lines, styles.StyleColHeader.Render(name)+":")
		wrapped := wrapCells(val, contentW)
		for _, wl := range wrapped {
			lines = append(lines, "  "+wl)
		}
		lines = append(lines, "")
	}

	if scroll < 0 {
		scroll = 0
	}
	if scroll > len(lines) {
		scroll = len(lines)
	}
	end := scroll + height
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[scroll:end], "\n")
}

// wrapCells hard-wraps a string at max cells without splitting multibyte runes.
func wrapCells(s string, max int) []string {
	if max <= 0 {
		return []string{""}
	}
	var out []string
	line := ""
	lineW := 0
	for _, r := range s {
		w := runewidth.RuneWidth(r)
		if lineW+w > max && lineW > 0 {
			out = append(out, line)
			line = ""
			lineW = 0
		}
		line += string(r)
		lineW += w
	}
	if line != "" {
		out = append(out, line)
	}
	if len(out) == 0 {
		out = []string{""}
	}
	return out
}
