package screens

import (
	"fmt"
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

	sidebarW := width / 5
	if sidebarW < 14 {
		sidebarW = 14
	}
	if sidebarW > 20 {
		sidebarW = 20
	}
	editorH := height / 4
	if editorH < 9 {
		editorH = 9
	}
	if editorH > height-5 {
		editorH = height - 5
	}
	if editorH < 2 {
		editorH = 2
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

	// main pane: a title line with the table name + count, then the content
	mainTitle := ""
	mainHasTitle := b != nil && b.ActiveTable != ""
	if mainHasTitle {
		mainTitle = fmt.Sprintf("%s · %d rows", b.ActiveTable, b.TotalRows)
	}
	mainBodyH := mainH - 2
	if mainHasTitle {
		mainBodyH--
	}
	var mainContent string
	if b == nil {
		mainContent = styles.StyleHeaderDim.Render("no connection")
	} else if structure {
		mainContent = RenderStructure(b, rightW-2, mainBodyH)
	} else if len(b.Tables) == 0 {
		mainContent = styles.StyleHeaderDim.Render("no tables — use the editor to create one")
	} else if b.ActiveTable != "" && len(b.Rows) == 0 {
		mainContent = styles.StyleHeaderDim.Render("empty table")
	} else {
		cols := make([]string, len(b.Columns))
		for i, c := range b.Columns {
			cols[i] = c.Name
		}
		mainContent = RenderDataTable(cols, b.Rows, b.Cursor, rightW-2, mainBodyH)
	}
	mainContent = titled(mainTitle, mainContent)

	// editor pane: title + editor
	editorContent := titled("SQL EDITOR", es.View(e, rightW-2, editorH-2-1))

	mainPane := boxed(mainContent, rightW, mainH, focus == FocusMain)
	editorPane := boxed(editorContent, rightW, editorH, focus == FocusEditor)
	right := lipgloss.JoinVertical(lipgloss.Top, mainPane, editorPane)

	if !showSidebar {
		return right
	}

	sidebarContent := titled("TABLES", RenderSidebar(b, sidebarCursor, sidebarW-2, height-2-1))
	sidebarPane := boxed(sidebarContent, sidebarW, height, focus == FocusSidebar)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebarPane, gap, right)
}

// titled prepends a pane title line to the body, if any.
func titled(title, body string) string {
	if title == "" {
		return body
	}
	return styles.StylePaneTitle.Render(title) + "\n" + body
}

// boxed wraps content in a bordered pane of exactly the given size, with an
// accent border when focused. lipgloss Width/Height apply to the content, and
// the border adds one line/column per side, so the content is sized to
// width-2 × height-2 and overflow lines are trimmed.
func boxed(content string, width, height int, focused bool) string {
	innerW := width - 2
	innerH := height - 2
	if innerW < 0 {
		innerW = 0
	}
	if innerH > 0 {
		if lines := strings.Split(content, "\n"); len(lines) > innerH {
			content = strings.Join(lines[:innerH], "\n")
		}
	}
	style := styles.StylePane
	if focused {
		style = styles.StylePaneFocus
	}
	return style.Width(innerW).Height(innerH).Render(content)
}
