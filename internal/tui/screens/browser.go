package screens

import (
	"fmt"
	"strings"

	"github.com/mattn/go-runewidth"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/store"
	"github.com/agmonetti/relm/internal/tui/styles"
)

// SidebarWindow returns the visible window of the sidebar list.
func SidebarWindow(cursor, height int) (offset, visible int) {
	offset = 0
	if cursor >= height {
		offset = cursor - height + 1
	}
	visible = height
	return
}

// RenderSidebar renders the catalog list with a selection cursor and badges.
func RenderSidebar(b *browser.Browser, cursor, width, height int) string {
	if b == nil || width < 6 || height < 1 {
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
		name := b.Tables[i]
		badge := ""
		if i < len(b.Items) && b.Items[i].Badge != "" {
			badge = " [" + b.Items[i].Badge + "]"
		}
		itemText := truncate(name+badge, width-2)
		switch {
		case i == cursor:
			sb.WriteString(styles.StyleSidebarActive.Render("> " + itemText))
		case b.Tables[i] == b.ActiveTable:
			sb.WriteString(styles.StyleSidebarActiveTable.Render("  " + itemText))
		default:
			sb.WriteString(styles.StyleSidebarItem.Render("  " + itemText))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// TableWindow returns the visible window of a data table.
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

const colSep = " | "
const colSepW = 3

// RenderMainBrowser renders data from any DataView.
func RenderMainBrowser(b *browser.Browser, width, height int) string {
	if b == nil {
		return styles.StyleHeaderDim.Render("no connection")
	}
	if len(b.Tables) == 0 {
		noun := "items"
		if b.ItemNoun != "" {
			noun = b.ItemNoun + "s"
		}
		return styles.StyleHeaderDim.Render(fmt.Sprintf("no %s — use the editor to create data", noun))
	}
	if b.Data == nil {
		if len(b.Columns) > 0 {
			cols := make([]string, len(b.Columns))
			for i, c := range b.Columns {
				cols[i] = c.Name
			}
			if len(b.Rows) == 0 {
				return RenderDataTable(cols, nil, b.Cursor, width, 2) + "\n\n" + RenderEmptyTable(cols, width)
			}
			return RenderDataTable(cols, b.Rows, b.Cursor, width, height)
		}
		return styles.StyleHeaderDim.Render("select an item")
	}

	switch v := b.Data.(type) {
	case *store.TabularData:
		cols := v.Columns
		if len(cols) == 0 {
			cols = make([]string, len(b.Columns))
			for i, c := range b.Columns {
				cols[i] = c.Name
			}
		}
		if len(v.Rows) == 0 {
			return RenderEmptyTable(cols, width)
		}
		return RenderDataTable(cols, v.Rows, b.Cursor, width, height)

	case *store.DocumentData:
		return RenderDocumentList(v, b.Cursor, width, height)

	case *store.KeyValueData:
		return RenderKeyValue(v, b.Cursor, width, height)

	case *store.GraphData:
		return RenderGraph(v, b.Cursor, width, height)

	case *store.RawTextData:
		return RenderRawText(v, width, height)

	default:
		// Fallback for Tabular convenience fields
		if len(b.Columns) > 0 {
			cols := make([]string, len(b.Columns))
			for i, c := range b.Columns {
				cols[i] = c.Name
			}
			if len(b.Rows) == 0 {
				return RenderEmptyTable(cols, width)
			}
			return RenderDataTable(cols, b.Rows, b.Cursor, width, height)
		}
		return styles.StyleHeaderDim.Render("unsupported view format")
	}
}

// RenderDocumentList renders a list of BSON/JSON documents.
func RenderDocumentList(docs *store.DocumentData, cursor, width, height int) string {
	if len(docs.Documents) == 0 {
		return styles.StyleHeaderDim.Render("( 0 documents )")
	}
	start, visible := TableWindow(len(docs.Documents), cursor, height)
	var sb strings.Builder
	for i := 0; i < visible; i++ {
		idx := start + i
		if idx >= len(docs.Documents) {
			break
		}
		d := docs.Documents[idx]
		summary := d.Summary
		if summary == "" {
			summary = d.RawJSON
		}
		line := fmt.Sprintf("[%s] %s", d.ID, summary)
		line = truncate(line, width-4)
		if idx == cursor {
			sb.WriteString(styles.StyleCursor.Render("> "+line) + "\n")
		} else {
			sb.WriteString(styles.StyleSidebarItem.Render("  "+line) + "\n")
		}
	}
	return sb.String()
}

// RenderKeyValue renders Redis Key-Value entries.
func RenderKeyValue(kv *store.KeyValueData, cursor, width, height int) string {
	if kv.Type == "string" {
		var sb strings.Builder
		sb.WriteString(styles.StyleHeaderDim.Render(fmt.Sprintf("Key: %s · Type: string · TTL: %s", kv.Key, kv.TTL)) + "\n\n")
		val := ""
		if len(kv.Entries) > 0 {
			val = kv.Entries[0].Value
		}
		sb.WriteString(styles.StyleSidebarItem.Render(val) + "\n")
		return sb.String()
	}

	cols := []string{"INDEX", "VALUE"}
	hasExtra := false
	for _, e := range kv.Entries {
		if e.Extra != "" {
			hasExtra = true
			break
		}
	}
	if hasExtra {
		cols = append(cols, "EXTRA")
	}

	rows := make([][]string, len(kv.Entries))
	for i, e := range kv.Entries {
		if hasExtra {
			rows[i] = []string{e.Index, e.Value, e.Extra}
		} else {
			rows[i] = []string{e.Index, e.Value}
		}
	}
	header := styles.StyleHeaderDim.Render(fmt.Sprintf("Key: %s · Type: %s · TTL: %s", kv.Key, kv.Type, kv.TTL))
	table := RenderDataTable(cols, rows, cursor, width, height-2)
	return header + "\n" + table
}

// RenderGraph renders Neo4j nodes and incident relationships.
func RenderGraph(g *store.GraphData, cursor, width, height int) string {
	if len(g.Nodes) == 0 {
		return styles.StyleHeaderDim.Render("( 0 nodes )")
	}
	start, visible := TableWindow(len(g.Nodes), cursor, height)
	var sb strings.Builder
	for i := 0; i < visible; i++ {
		idx := start + i
		if idx >= len(g.Nodes) {
			break
		}
		n := g.Nodes[idx]
		labels := ":" + strings.Join(n.Labels, ":")
		props := ""
		for k, v := range n.Properties {
			props += fmt.Sprintf("%s: %q ", k, v)
		}
		edges := fmt.Sprintf("[%d edges]", len(n.Incident))
		line := fmt.Sprintf("(%s %s) %s %s", n.ID, labels, props, edges)
		line = truncate(line, width-4)
		if idx == cursor {
			sb.WriteString(styles.StyleCursor.Render("> "+line) + "\n")
		} else {
			sb.WriteString(styles.StyleSidebarItem.Render("  "+line) + "\n")
		}
	}
	return sb.String()
}

// RenderRawText renders plain diagnostic text.
func RenderRawText(raw *store.RawTextData, width, height int) string {
	lines := strings.Split(raw.Text, "\n")
	if len(lines) > height {
		lines = lines[:height]
	}
	for i, l := range lines {
		lines[i] = truncate(l, width-2)
	}
	return strings.Join(lines, "\n")
}

// RenderDataTable renders columns + rows with cursor row highlighted.
func RenderDataTable(cols []string, rows [][]string, cursor, width, height int) string {
	if len(cols) == 0 {
		return ""
	}

	widths := colWidths(cols, rows, width)
	start, visible := TableWindow(len(rows), cursor, height)

	var sb strings.Builder
	sb.WriteString(renderHeader(cols, widths) + "\n")
	sb.WriteString(renderSep(cols, widths) + "\n")

	for i := 0; i < visible; i++ {
		row := start + i
		if row >= len(rows) {
			break
		}
		sb.WriteString(renderRow(rows[row], widths, row == cursor) + "\n")
	}
	return sb.String()
}

func renderHeader(cells []string, widths []int) string {
	var sb strings.Builder
	for i, c := range cells {
		if i >= len(widths) {
			break
		}
		sb.WriteString(styles.StyleColHeader.Render(pad(truncate(c, widths[i]), widths[i])))
		if i < len(cells)-1 {
			sb.WriteString(styles.StyleBorder.Render(colSep))
		}
	}
	return sb.String()
}

func renderSep(cells []string, widths []int) string {
	var sb strings.Builder
	for i, w := range widths {
		if i >= len(cells) {
			break
		}
		sb.WriteString(strings.Repeat("-", w))
		if i < len(cells)-1 {
			sb.WriteString("-+-")
		}
	}
	return styles.StyleBorder.Render(sb.String())
}

const colMaxW = 36
const colMinW = 4

func colWidths(cols []string, rows [][]string, width int) []int {
	n := len(cols)
	widths := make([]int, n)
	for i, c := range cols {
		widths[i] = runewidth.StringWidth(c)
	}
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
	for i := range widths {
		if widths[i] > colMaxW {
			widths[i] = colMaxW
		}
	}

	total := (n - 1) * colSepW
	for _, w := range widths {
		total += w
	}
	if total <= width {
		return widths
	}

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

	total = (n - 1) * colSepW
	for _, w := range widths {
		total += w
	}
	if total > width {
		remaining := total - width
		for i := range widths {
			if remaining <= 0 {
				break
			}
			shrink := widths[i] - 1
			if shrink > remaining {
				shrink = remaining
			}
			if shrink > 0 {
				widths[i] -= shrink
				remaining -= shrink
			}
		}
	}
	return widths
}

func renderRow(cells []string, widths []int, selected bool) string {
	var sb strings.Builder
	for i, cell := range cells {
		if i >= len(widths) {
			break
		}
		sb.WriteString(renderCellText(cell, widths[i], selected))
		if i < len(cells)-1 {
			if selected {
				sb.WriteString(styles.StyleCursor.Render(colSep))
			} else {
				sb.WriteString(styles.StyleBorder.Render(colSep))
			}
		}
	}
	return sb.String()
}

func renderCellText(cell string, width int, selected bool) string {
	text := cell
	if text == "" {
		text = styles.NullCell()
	} else {
		text = truncate(text, width)
	}
	text = pad(text, width)
	if selected {
		text = styles.StyleCursor.Render(text)
	}
	return text
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

// RenderEmptyTable renders the centered ASCII-art empty state drawing.
func RenderEmptyTable(cols []string, width int) string {
	if width < 12 {
		width = 12
	}
	caption := "( 0 rows returned )"
	inner := runewidth.StringWidth(caption)
	if inner > width-3 {
		inner = width - 3
	}
	if inner < 6 {
		inner = 6
	}
	center := func(s string) string {
		pad := (width - runewidth.StringWidth(s)) / 2
		if pad < 0 {
			pad = 0
		}
		return strings.Repeat(" ", pad) + s
	}
	head := "+" + strings.Repeat("-", inner) + "+"
	mid := "|" + strings.Repeat(" ", inner) + "|"
	return styles.StyleBorder.Render(center(head)) + "\n" +
		styles.StyleBorder.Render(center(mid)) + "\n" +
		styles.StyleBorder.Render(center(head)) + "\n\n" +
		styles.StyleHeaderDim.Render(center(caption))
}

// RenderRowDetail renders a single row with every column and full value.
func RenderRowDetail(title string, cols, vals []string, scroll, width, height int) string {
	if width < 4 {
		width = 40
	}
	contentW := width - 4
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
