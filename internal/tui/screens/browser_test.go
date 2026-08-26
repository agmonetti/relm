package screens

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/store"
)

func init() {
	// The tests run without a TTY; force a color profile so lipgloss emits
	// ANSI escapes and the cursor highlight is detectable in the output.
	lipgloss.SetColorProfile(termenv.TrueColor)
	lipgloss.SetHasDarkBackground(true)
}

func TestRenderDataTable_HighlightsOnlyCursorRow(t *testing.T) {
	cols := []string{"a", "b", "c"}
	rows := [][]string{{"1", "2", "3"}, {"4", "5", "6"}, {"7", "8", "9"}}

	out := RenderDataTable(cols, rows, 1, 40, 10)
	lines := strings.Split(out, "\n")

	// lines[0] = header, lines[1] = separator, lines[2..4] = the three rows
	if hasBackground(lines[2]) {
		t.Errorf("row 0 must not be highlighted: %q", lines[2])
	}
	if !hasBackground(lines[3]) {
		t.Errorf("row 1 (cursor) must be highlighted: %q", lines[3])
	}
	if hasBackground(lines[4]) {
		t.Errorf("row 2 must not be highlighted: %q", lines[4])
	}
}

func TestRenderDataTable_NoSelectionHighlightsNothing(t *testing.T) {
	cols := []string{"a", "b", "c"}
	rows := [][]string{{"1", "2", "3"}, {"4", "5", "6"}}

	out := RenderDataTable(cols, rows, -1, 40, 10)
	// lines[0] is the column header (styled by design); the separator is a
	// foreground-only line; the data rows must not carry a selection background
	for i, line := range strings.Split(out, "\n") {
		if i <= 1 {
			continue
		}
		if hasBackground(line) {
			t.Errorf("line %d must not be highlighted with cursor=-1: %q", i, line)
		}
	}
}

// hasBackground reports whether an ANSI string sets a background color.
func hasBackground(s string) bool {
	return strings.Contains(s, "48;")
}

func TestRenderMainBrowser_EmptyTabularDataShowsColumns(t *testing.T) {
	// The real browse flow always sets Data to a TabularData, even when the
	// table has zero rows: the empty table must still draw its column header.
	b := &browser.Browser{
		Tables:      []string{"users"},
		ActiveTable: "users",
		Data: &store.TabularData{
			Columns: []string{"id", "name", "email"},
			Rows:    nil,
		},
		Columns: []store.Column{
			{Name: "id", Type: "INTEGER", PK: true},
			{Name: "name", Type: "TEXT"},
			{Name: "email", Type: "TEXT"},
		},
		PageSize: 50,
	}

	out := RenderMainBrowser(b, 96, 20)
	foundHeader := false
	foundHint := false
	for _, l := range strings.Split(out, "\n") {
		s := ansi.Strip(l)
		if strings.Contains(s, "id") && strings.Contains(s, "name") && strings.Contains(s, "email") {
			foundHeader = true
		}
		if strings.Contains(s, "0 rows returned") {
			foundHint = true
		}
	}
	if !foundHeader {
		t.Error("empty table must still show the column headers")
	}
	if !foundHint {
		t.Error("empty table must show the '( 0 rows returned )' hint")
	}
}

func TestRenderMainBrowser_EmptyTabularDataFallsBackToSchemaColumns(t *testing.T) {
	// Cassandra SELECT * on an empty table can come back without column
	// metadata; the renderer must fall back to the schema columns (b.Columns).
	b := &browser.Browser{
		Tables:      []string{"users"},
		ActiveTable: "users",
		Data: &store.TabularData{
			Columns: nil, // empty result set carried no column metadata
			Rows:    nil,
		},
		Columns: []store.Column{
			{Name: "id", Type: "INT", PK: true},
			{Name: "name", Type: "TEXT"},
		},
		PageSize: 50,
	}

	out := RenderMainBrowser(b, 96, 20)
	foundHeader := false
	foundHint := false
	for _, l := range strings.Split(out, "\n") {
		s := ansi.Strip(l)
		if strings.Contains(s, "id") && strings.Contains(s, "name") {
			foundHeader = true
		}
		if strings.Contains(s, "0 rows returned") {
			foundHint = true
		}
	}
	if !foundHeader {
		t.Error("empty table without result metadata must still show the schema column headers")
	}
	if !foundHint {
		t.Error("empty table must show the '( 0 rows returned )' hint")
	}
}

func TestRenderMainBrowser_EmptyTabularDataWithoutAnyColumnsShowsOnlyHint(t *testing.T) {
	// No schema available (e.g. a view that has neither result metadata nor
	// an inspection view): only the empty-state box is drawn, no dead header.
	b := &browser.Browser{
		Tables:      []string{"users"},
		ActiveTable: "users",
		Data: &store.TabularData{
			Columns: nil,
			Rows:    nil,
		},
		PageSize: 50,
	}

	out := RenderMainBrowser(b, 96, 20)
	if !strings.Contains(out, "0 rows returned") {
		t.Error("empty table must show the '( 0 rows returned )' hint")
	}
}

func TestRenderSidebar_ScrollsToCursor(t *testing.T) {
	b := &browser.Browser{ActiveTable: "table_005"}
	for i := 0; i < 200; i++ {
		b.Tables = append(b.Tables, fmt.Sprintf("table_%03d", i))
	}

	// cursor near the top: no scrolling, window 0..4, cursor at bottom of window
	out := RenderSidebar(b, 5, 20, 5)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("lines = %d, want 5", len(lines))
	}
	if !strings.Contains(lines[4], ">") || !strings.Contains(lines[4], "table_005") {
		t.Errorf("cursor line = %q, want >table_005 at last visible line", lines[4])
	}
	if !strings.Contains(lines[0], "table_001") {
		t.Errorf("first visible line = %q, want table_001", lines[0])
	}

	// cursor far away: scroll so it stays visible at bottom of window (146..150)
	out = RenderSidebar(b, 150, 20, 5)
	lines = strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("lines = %d, want 5 (scrolled)", len(lines))
	}
	if !strings.Contains(lines[4], ">") || !strings.Contains(lines[4], "table_150") {
		t.Errorf("last visible line = %q, want >table_150", lines[4])
	}
	if !strings.Contains(lines[0], "table_146") {
		t.Errorf("first visible line = %q, want table_146", lines[0])
	}

	// cursor at the very end: window shows last 5 tables, cursor at bottom
	out = RenderSidebar(b, 199, 20, 5)
	lines = strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 5 || !strings.Contains(lines[4], "table_199") || !strings.Contains(lines[4], ">") {
		t.Errorf("last table lines = %q, want 5 lines ending in >table_199", strings.Join(lines, "|"))
	}
	if !strings.Contains(lines[0], "table_195") {
		t.Errorf("first visible line = %q, want table_195", lines[0])
	}
}

func TestRenderSidebar_MarksOpenedTable(t *testing.T) {
	b := &browser.Browser{ActiveTable: "table_002"}
	for i := 0; i < 10; i++ {
		b.Tables = append(b.Tables, fmt.Sprintf("table_%03d", i))
	}

	// cursor on 0, opened table 002: it must be shown without the ">"
	out := RenderSidebar(b, 0, 20, 10)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if !strings.Contains(lines[0], ">") {
		t.Errorf("cursor line must have '>': %q", lines[0])
	}
	if strings.Contains(lines[2], ">") {
		t.Errorf("opened table line must not have '>': %q", lines[2])
	}
}

func TestRenderDataTable_CursorStaysVisible(t *testing.T) {
	cols := []string{"id"}
	rows := make([][]string, 0, 10)
	for i := 0; i < 10; i++ {
		rows = append(rows, []string{string(rune('a' + i))})
	}

	// height=6 -> 4 visible rows (6 minus 2 for header/separator). Total lines produced must be 6.
	out := RenderDataTable(cols, rows, 0, 20, 6)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	if len(lines) != 6 {
		t.Fatalf("total lines = %d, want 6", len(lines))
	}
	data := dataLines(out, 6)
	if len(data) != 4 {
		t.Fatalf("data rows = %d, want 4", len(data))
	}
	if data[0] != "a" || data[3] != "d" {
		t.Errorf("cursor 0 should show rows a-d only: %v", data)
	}

	// Cursor 3 (last visible row in initial window): window 0..3 (rows a-d)
	out = RenderDataTable(cols, rows, 3, 20, 6)
	data = dataLines(out, 6)
	if len(data) != 4 {
		t.Fatalf("data rows = %d, want 4", len(data))
	}
	if data[0] != "a" || data[3] != "d" {
		t.Errorf("cursor 3 should show rows a-d: %v", data)
	}

	// Cursor 4: window must displace by 1 so row 4 (e) is visible at bottom (rows b-e)
	out = RenderDataTable(cols, rows, 4, 20, 6)
	data = dataLines(out, 6)
	if len(data) != 4 {
		t.Fatalf("data rows = %d, want 4", len(data))
	}
	if data[0] != "b" || data[3] != "e" {
		t.Errorf("cursor 4 should show rows b-e: %v", data)
	}

	// Cursor 7: window must scroll so row 7 (h) is visible (rows e-h).
	out = RenderDataTable(cols, rows, 7, 20, 6)
	data = dataLines(out, 6)
	if len(data) != 4 {
		t.Fatalf("data rows = %d, want 4", len(data))
	}
	if data[0] != "e" || data[3] != "h" {
		t.Errorf("cursor 7 should show rows e-h: %v", data)
	}
}

func TestColWidths_ShrinksToFitManyColumns(t *testing.T) {
	cols := make([]string, 0, 12)
	for i := 0; i < 12; i++ {
		cols = append(cols, "col012345") // 9 wide
	}
	rows := [][]string{cols} // each cell as wide as its header

	widths := colWidths(cols, rows, 70)
	total := (len(widths) - 1) * colSepW
	for _, w := range widths {
		total += w
	}
	if total > 70 {
		t.Errorf("widths %v sum to %d, want <= 70", widths, total)
	}
	for _, w := range widths {
		if w < 1 {
			t.Errorf("width %d below the 1-char minimum", w)
		}
	}

	// with a tiny width every column keeps shrinking to a 1-cell minimum; the
	// table may still not fit when even n + (n-1)*3 separators exceeds the pane
	widths = colWidths(cols, rows, 40)
	total = (len(widths) - 1) * colSepW
	for _, w := range widths {
		total += w
	}
	minTotal := len(widths) + (len(widths)-1)*colSepW
	if total != minTotal {
		t.Errorf("widths %v sum to %d, want the 1-cell minimum %d", widths, total, minTotal)
	}
	for _, w := range widths {
		if w < 1 {
			t.Errorf("width %d below the 1-char minimum", w)
		}
	}
}

func dataLines(out string, height int) []string {
	lines := ansiStripLines(strings.TrimSuffix(out, "\n"))
	data := make([]string, 0, len(lines))
	for i, l := range lines {
		if i == 0 || i == 1 {
			continue // header + separator
		}
		data = append(data, strings.TrimSpace(l))
	}
	return data
}

func TestColWidths_KeepsNaturalWhenItFits(t *testing.T) {
	cols := []string{"id", "name"}
	rows := [][]string{{"1", "alice"}}
	widths := colWidths(cols, rows, 40)
	if widths[0] != 2 || widths[1] != 5 {
		t.Errorf("widths = %v, want [2 5]", widths)
	}
}

func TestColWidths_CapsAndDistributes(t *testing.T) {
	cols := []string{"id", "name", "body"}
	rows := [][]string{{"1", "alice", strings.Repeat("x", 100)}}
	widths := colWidths(cols, rows, 40)
	if len(widths) != 3 {
		t.Fatalf("widths = %v", widths)
	}
	if widths[2] > colMaxW {
		t.Errorf("body width = %d, want <= %d", widths[2], colMaxW)
	}
	total := (len(widths) - 1) * colSepW
	for _, w := range widths {
		total += w
	}
	if total > 40 {
		t.Errorf("total = %d, want <= 40", total)
	}
	// the narrow columns keep their natural width; the long one absorbs the rest
	if widths[0] != 2 {
		t.Errorf("id width = %d, want 2 (kept natural)", widths[0])
	}
	if widths[2] <= widths[1] {
		t.Errorf("body width %d should be the largest", widths[2])
	}
}

func TestColWidths_CapsSingleHugeColumn(t *testing.T) {
	cols := []string{"id", "huge"}
	rows := [][]string{{"1", strings.Repeat("x", 500)}}
	widths := colWidths(cols, rows, 200)
	if widths[1] != colMaxW {
		t.Errorf("huge width = %d, want %d", widths[1], colMaxW)
	}
}

func TestRenderRowDetail_ShowsFullValue(t *testing.T) {
	val := strings.Repeat("z", 100)
	out := RenderRowDetail("users", []string{"id", "text"}, []string{"1", val}, 0, 40, 20)
	// the value is wrapped, so every character must still be present
	if got := strings.Count(out, "z"); got != 100 {
		t.Errorf("found %d 'z', want 100 (full value)", got)
	}
	if !strings.Contains(out, "id") || !strings.Contains(out, "text") {
		t.Error("column names must appear")
	}
}

func TestRenderRowDetail_Scrolls(t *testing.T) {
	val := strings.Repeat("y", 100)
	out0 := RenderRowDetail("t", []string{"id", "body"}, []string{"1", val}, 0, 40, 3)
	out1 := RenderRowDetail("t", []string{"id", "body"}, []string{"1", val}, 5, 40, 3)
	if out0 == out1 {
		t.Error("scrolling should change the visible content")
	}
}

func ansiStripLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		out = append(out, ansi.Strip(l))
	}
	return out
}
