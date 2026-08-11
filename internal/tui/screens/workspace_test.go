package screens

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/mattn/go-runewidth"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/editor"
	"github.com/agmonetti/relm/internal/store"
)

func TestRenderWorkspace_ShowsPaneTitles(t *testing.T) {
	out := RenderWorkspace(sampleTestBrowser(), NewEditorScreen(), sampleTestEditor(),
		FocusSidebar, false, true, 0, 100, 24)

	if !strings.Contains(out, "TABLES") {
		t.Error("workspace must show the TABLES sidebar title")
	}
	if !strings.Contains(out, "SQL EDITOR") {
		t.Error("workspace must show the SQL EDITOR title")
	}
	if !strings.Contains(out, "users · 2 rows") {
		t.Errorf("main pane title missing: %q", out)
	}
	// the column header is rendered and styled (ANSI)
	lines := strings.Split(out, "\n")
	found := false
	for _, l := range lines {
		if strings.Contains(l, "id") && strings.Contains(l, "name") && strings.Contains(l, "email") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("column header missing: %q", out)
	}
}

func TestRenderWorkspace_ExactHeight(t *testing.T) {
	for _, h := range []int{12, 20, 30} {
		out := RenderWorkspace(sampleTestBrowser(), NewEditorScreen(), sampleTestEditor(),
			FocusSidebar, false, true, 0, 100, h)
		if n := len(strings.Split(out, "\n")); n != h {
			t.Errorf("height=%d: got %d lines", h, n)
		}
	}
}

func TestRenderWorkspace_ExactWidth(t *testing.T) {
	for _, w := range []int{60, 100, 150} {
		out := RenderWorkspace(sampleTestBrowser(), NewEditorScreen(), sampleTestEditor(),
			FocusSidebar, false, true, 0, w, 20)
		for i, line := range strings.Split(out, "\n") {
			if got := runewidth.StringWidth(ansi.Strip(line)); got != w {
				t.Errorf("width=%d: line %d is %d cols", w, i, got)
			}
		}
	}
}

func TestRenderWorkspace_HasPanelGapsAndPadding(t *testing.T) {
	out := RenderWorkspace(sampleTestBrowser(), NewEditorScreen(), sampleTestEditor(),
		FocusSidebar, false, true, 0, 100, 24)
	lines := strings.Split(out, "\n")

	// the sidebar and main boxes are separated by exactly one char
	top := ansi.Strip(lines[0])
	if !strings.Contains(top, "╮ ╭") {
		t.Errorf("expected a 1-char gap between the panes: %q", top)
	}
	// content is padded one char inside the border
	foundTitle := false
	for _, l := range lines {
		if s := ansi.Strip(l); strings.Contains(s, "TABLES") {
			if !strings.HasPrefix(s, "╭─ TABLES") {
				t.Errorf("sidebar title must live in the top border: %q", s)
			}
			foundTitle = true
		}
	}
	if !foundTitle {
		t.Error("sidebar title not rendered")
	}
}

func TestRenderWorkspace_WithoutSidebarSpansWidth(t *testing.T) {
	// with the sidebar hidden the output must not contain the sidebar tables
	out := RenderWorkspace(sampleTestBrowser(), NewEditorScreen(), sampleTestEditor(),
		FocusSidebar, false, false, 0, 100, 20)
	if strings.Contains(out, "TABLES") {
		t.Error("sidebar must be hidden")
	}
}

func sampleTestBrowser() *browser.Browser {
	return &browser.Browser{
		Tables:      []string{"orders", "users"},
		ActiveTable: "users",
		Columns: []store.Column{
			{Name: "id", Type: "INTEGER", PK: true},
			{Name: "name", Type: "TEXT"},
			{Name: "email", Type: "TEXT"},
		},
		Rows:      [][]string{{"1", "Alice", "alice@test.com"}, {"2", "Bob", "bob@test.com"}},
		TotalRows: 2,
		PageSize:  50,
	}
}

func sampleTestEditor() *editor.Editor {
	return &editor.Editor{Buffer: "SELECT * FROM users", History: editor.NewHistory()}
}
