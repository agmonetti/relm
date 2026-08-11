package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

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
	// one line of air between the main pane and the editor
	mainH := height - editorH - 1
	if mainH < 2 {
		mainH = 2
	}

	// the right column spans the full width when the sidebar is hidden; a
	// one-char gap separates it from the sidebar
	gap := " "
	rightW := width
	if showSidebar {
		rightW = width - sidebarW - 1
	}
	if rightW < 20 {
		rightW = 20
	}

	// content width inside a pane: border (1 col/side) + padding (1 col/side)
	contentW := func(paneW int) int {
		if w := paneW - 4; w < 1 {
			return 1
		}
		return paneW - 4
	}

	// main pane: the title (table name + count) lives inside the top border
	mainTitle := ""
	if b != nil && b.ActiveTable != "" {
		mainTitle = fmt.Sprintf("%s · %d rows", b.ActiveTable, b.TotalRows)
	}
	mainBodyH := mainH - 2
	var mainContent string
	if b == nil {
		mainContent = styles.StyleHeaderDim.Render("no connection")
	} else if structure {
		mainContent = RenderStructure(b, contentW(rightW), mainBodyH)
	} else if len(b.Tables) == 0 {
		mainContent = styles.StyleHeaderDim.Render("no tables — use the editor to create one")
	} else if b.ActiveTable != "" && len(b.Rows) == 0 {
		mainContent = styles.StyleHeaderDim.Render("empty table")
	} else {
		cols := make([]string, len(b.Columns))
		for i, c := range b.Columns {
			cols[i] = c.Name
		}
		mainContent = RenderDataTable(cols, b.Rows, b.Cursor, contentW(rightW), mainBodyH)
	}

	// editor pane: title + editor
	editorContent := es.View(e, contentW(rightW), editorH-2)

	mainPane := boxed(mainContent, mainTitle, rightW, mainH, focus == FocusMain)
	editorPane := boxed(editorContent, "SQL EDITOR", rightW, editorH, focus == FocusEditor)
	right := lipgloss.JoinVertical(lipgloss.Top, mainPane, " ", editorPane)

	if !showSidebar {
		return right
	}

	sidebarContent := RenderSidebar(b, sidebarCursor, contentW(sidebarW), height-2)
	sidebarPane := boxed(sidebarContent, "TABLES", sidebarW, height, focus == FocusSidebar)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebarPane, gap, right)
}

// boxed wraps content in a bordered pane of exactly the given size, with the
// title drawn inside the top border and an accent border when focused.
// lipgloss Width/Height apply to the content (Width already includes the
// horizontal padding) and the border adds one line/column per side, so the
// content area is width-4 × height-2; overflow lines are trimmed.
func boxed(content, title string, width, height int, focused bool) string {
	innerW := width - 2 // width of content + horizontal padding
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
	box := style.Width(innerW).Height(innerH).Padding(0, 1).Render(content)
	if title == "" {
		return box
	}
	lines := strings.Split(box, "\n")
	lines[0] = topBorder(title, width, focused)
	return strings.Join(lines, "\n")
}

// topBorder rebuilds the top border line of a pane with the title embedded:
// ╭─ TABLES ──────────╮. The border runes keep the pane's border color and
// the title keeps the pane title style.
func topBorder(title string, width int, focused bool) string {
	border := styles.StyleBorderLine
	if focused {
		border = styles.StyleBorderLineFocus
	}
	left := "╭─ "
	maxTitle := width - 6
	if maxTitle < 0 {
		maxTitle = 0
	}
	title = truncate(title, maxTitle)
	fill := width - runewidth.StringWidth(left) - runewidth.StringWidth(title) - 2
	if fill < 1 {
		fill = 1
	}
	return border.Render(left) + styles.StylePaneTitle.Render(title) +
		border.Render(" "+strings.Repeat("─", fill)+"╮")
}
