package screens

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
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

	out := renderDataTable(cols, rows, 1, 40, 10)
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

	out := renderDataTable(cols, rows, -1, 40, 10)
	for i, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "\x1b[") {
			t.Errorf("line %d must not be highlighted with cursor=-1: %q", i, line)
		}
	}
}
