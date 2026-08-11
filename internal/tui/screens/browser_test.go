package screens

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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
	for i, line := range strings.Split(out, "\n") {
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
