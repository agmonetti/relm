package screens

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/editor"
	"github.com/agmonetti/relm/internal/tui/styles"
)

// WorkspaceFocus identifies which pane of the workspace has the keyboard focus.
type WorkspaceFocus int

const (
	// FocusSidebar is the tables list on the left.
	FocusSidebar WorkspaceFocus = iota
	// FocusMain is the central panel (browser or structure).
	FocusMain
	// FocusEditor is the bottom SQL editor.
	FocusEditor
)

// RenderWorkspace renders the single working screen: sidebar, main pane and
// editor, each inside a visible border. The focused pane has an accent border.
func RenderWorkspace(b *browser.Browser, es *EditorScreen, e *editor.Editor,
	focus WorkspaceFocus, structure, showSidebar bool, sidebarCursor int,
	width, height int) string {
	if width < 10 || height < 3 {
		return styles.StyleHeaderDim.Render("terminal too small")
	}
	if width < 60 {
		showSidebar = false
	}

	sidebarW := width / 4
	if sidebarW < 14 {
		sidebarW = 14
	}
	if sidebarW > 36 {
		sidebarW = 36
	}
	editorH := height / 3
	if editorH < 7 {
		editorH = 7
	}
	if editorH > height-3 {
		editorH = height - 3
	}
	mainH := height - editorH
	if mainH < 2 {
		mainH = 2
	}

	// the right column spans the full width when the sidebar is hidden
	gap := "  "
	rightW := width
	if showSidebar {
		rightW = width - sidebarW - lipgloss.Width(gap)
	}
	if rightW < 20 {
		rightW = 20
	}

	var mainContent string
	if b == nil {
		mainContent = styles.StyleHeaderDim.Render("no connection")
	} else if structure {
		mainContent = RenderStructure(b, rightW-2, mainH-2)
	} else if len(b.Tables) == 0 {
		mainContent = styles.StyleHeaderDim.Render("no tables — use the editor to create one")
	} else if b.ActiveTable != "" && len(b.Rows) == 0 {
		mainContent = styles.StyleHeaderDim.Render("empty table")
	} else {
		cols := make([]string, len(b.Columns))
		for i, c := range b.Columns {
			cols[i] = c.Name
		}
		mainContent = RenderDataTable(cols, b.Rows, b.Cursor, rightW-2, mainH-2)
	}

	editorContent := es.View(e, rightW-2, editorH-2)

	mainPane := boxed(mainContent, rightW, mainH, focus == FocusMain)
	editorPane := boxed(editorContent, rightW, editorH, focus == FocusEditor)
	right := lipgloss.JoinVertical(lipgloss.Top, mainPane, editorPane)

	if !showSidebar {
		return right
	}

	sidebarContent := RenderSidebar(b, sidebarCursor, sidebarW-2, height-2)
	sidebarPane := boxed(sidebarContent, sidebarW, height, focus == FocusSidebar)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebarPane, gap, right)
}

// boxed wraps content in a bordered pane of the given size, with an accent
// border when focused. Overflow lines are trimmed so the pane never exceeds
// the requested height.
func boxed(content string, width, height int, focused bool) string {
	if inner := height - 2; inner > 0 {
		if lines := strings.Split(content, "\n"); len(lines) > inner {
			content = strings.Join(lines[:inner], "\n")
		}
	}
	style := styles.StylePane
	if focused {
		style = styles.StylePaneFocus
	}
	return style.Width(width).Height(height).Render(content)
}
