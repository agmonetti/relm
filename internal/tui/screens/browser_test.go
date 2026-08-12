package screens

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"github.com/agmonetti/relm/internal/browser"
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

	// lines[0] = header, lines[1..3] = the three rows
	if strings.Contains(lines[1], "\x1b[") {
		t.Errorf("row 0 must not be highlighted: %q", lines[1])
	}
	if !strings.Contains(lines[2], "\x1b[") {
		t.Errorf("row 1 (cursor) must be highlighted: %q", lines[2])
	}
	if strings.Contains(lines[3], "\x1b[") {
		t.Errorf("row 2 must not be highlighted: %q", lines[3])
	}
}

func TestRenderDataTable_NoSelectionHighlightsNothing(t *testing.T) {
	cols := []string{"a", "b", "c"}
	rows := [][]string{{"1", "2", "3"}, {"4", "5", "6"}}

	out := RenderDataTable(cols, rows, -1, 40, 10)
	// lines[0] is the column header (styled by design); the data rows must not
	// be highlighted
	for i, line := range strings.Split(out, "\n") {
		if i == 0 {
			continue
		}
		if strings.Contains(line, "\x1b[") {
			t.Errorf("line %d must not be highlighted with cursor=-1: %q", i, line)
		}
	}
}

func TestRenderSidebar_ScrollsToCursor(t *testing.T) {
	b := &browser.Browser{ActiveTable: "table_005"}
	for i := 0; i < 200; i++ {
		b.Tables = append(b.Tables, fmt.Sprintf("table_%03d", i))
	}

	// cursor at the top: no scrolling
	out := RenderSidebar(b, 5, 20, 5)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("lines = %d, want 5", len(lines))
	}
	if !strings.Contains(lines[0], ">") || !strings.Contains(lines[0], "table_005") {
		t.Errorf("cursor line = %q, want >table_005", lines[0])
	}

	// cursor far away: scroll so it stays visible
	out = RenderSidebar(b, 150, 20, 5)
	lines = strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("lines = %d, want 5 (scrolled)", len(lines))
	}
	if !strings.Contains(lines[0], ">") || !strings.Contains(lines[0], "table_150") {
		t.Errorf("first visible line = %q, want >table_150", lines[0])
	}
	if !strings.Contains(lines[4], "table_154") {
		t.Errorf("last visible line = %q, want table_154", lines[4])
	}

	// cursor at the very end: the last table alone is visible
	out = RenderSidebar(b, 199, 20, 5)
	lines = strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], "table_199") {
		t.Errorf("last table line = %q, want a single >table_199", strings.Join(lines, "|"))
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

	// height=6 -> 5 visible rows. Cursor 0: rows 0..4.
	out := RenderDataTable(cols, rows, 0, 20, 6)
	data := dataLines(out, 6)
	if len(data) != 5 {
		t.Fatalf("data rows = %d, want 5", len(data))
	}
	if data[0] != "a" || data[4] != "e" {
		t.Errorf("cursor 0 should show rows a-e only: %v", data)
	}

	// Cursor 7: window must scroll so row 7 (h) is visible (rows d-h).
	out = RenderDataTable(cols, rows, 7, 20, 6)
	data = dataLines(out, 6)
	if len(data) != 5 {
		t.Fatalf("data rows = %d, want 5", len(data))
	}
	if data[0] != "d" || data[4] != "h" {
		t.Errorf("cursor 7 should show rows d-h: %v", data)
	}
}

func TestColWidths_ShrinksToFitManyColumns(t *testing.T) {
	cols := make([]string, 0, 12)
	for i := 0; i < 12; i++ {
		cols = append(cols, "col012345") // 9 wide
	}
	rows := [][]string{cols} // each cell as wide as its header

	widths := colWidths(cols, rows, 70)
	total := len(widths) - 1
	for _, w := range widths {
		total += w
	}
	if total > 70 {
		t.Errorf("widths %v sum to %d, want <= 70", widths, total)
	}
	for _, w := range widths {
		if w < 4 {
			t.Errorf("width %d below the 4-char floor", w)
		}
	}

	// with a tiny width every column reaches the floor
	widths = colWidths(cols, rows, 40)
	for i, w := range widths {
		if w != 4 {
			t.Errorf("width[%d] = %d, want 4 (floor)", i, w)
		}
	}
}

func dataLines(out string, height int) []string {
	lines := ansiStripLines(strings.TrimSuffix(out, "\n"))
	if len(lines) > height {
		lines = lines[:height]
	}
	data := make([]string, 0, len(lines))
	for i, l := range lines {
		if i == 0 {
			continue // header
		}
		data = append(data, strings.TrimSpace(l))
	}
	return data
}

func ansiStripLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		out = append(out, ansi.Strip(l))
	}
	return out
}
