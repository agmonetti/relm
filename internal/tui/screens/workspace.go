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
	// FocusSidebar is the items list on the left.
	FocusSidebar WorkspaceFocus = iota
	// FocusMain is the central panel (browser or structure).
	FocusMain
	// FocusEditor is the bottom query editor.
	FocusEditor
)

// Pane sizing bounds.
const (
	SidebarMinW = 10
	EditorMinH  = 5
	RightMinW   = 20
	MainMinH    = 2
)

// WorkspaceLayout is the resolved geometry of the workspace panes.
type WorkspaceLayout struct {
	ShowSidebar bool
	SidebarW    int
	EditorH     int
	MainH       int
	RightW      int
}

// ComputeLayout resolves the pane sizes for the given terminal size.
func ComputeLayout(width, height int, showSidebar bool, sidebarW, editorH int) WorkspaceLayout {
	if width < 60 {
		showSidebar = false
	}
	l := WorkspaceLayout{ShowSidebar: showSidebar}

	if editorH > 0 {
		l.EditorH = clampInt(editorH, EditorMinH, height-MainMinH-1)
		if l.EditorH < 2 {
			l.EditorH = 2
		}
	} else {
		l.EditorH = height / 4
		if l.EditorH < 9 {
			l.EditorH = 9
		}
		if l.EditorH > height-5 {
			l.EditorH = height - 5
		}
		if l.EditorH < 2 {
			l.EditorH = 2
		}
	}
	l.MainH = height - l.EditorH - 1
	if l.MainH < MainMinH {
		l.MainH = MainMinH
	}

	if showSidebar && sidebarW > 0 {
		l.SidebarW = clampInt(sidebarW, SidebarMinW, width-RightMinW-1)
	} else {
		l.SidebarW = clampInt(width/5, 14, 20)
	}

	l.RightW = width
	if showSidebar {
		l.RightW = width - l.SidebarW - 1
	}
	if l.RightW < 4 {
		l.RightW = 4
	}
	return l
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// RenderWorkspace renders the single working screen.
func RenderWorkspace(b *browser.Browser, es *EditorScreen, e *editor.Editor,
	focus WorkspaceFocus, structure bool, layout WorkspaceLayout, sidebarCursor int,
	width, height int) string {
	if width < 10 || height < 3 {
		return styles.StyleHeaderDim.Render("terminal too small")
	}

	sidebarW := layout.SidebarW
	editorH := layout.EditorH
	mainH := layout.MainH
	rightW := layout.RightW
	showSidebar := layout.ShowSidebar

	gap := " "
	contentW := func(paneW int) int {
		if w := paneW - 4; w < 1 {
			return 1
		}
		return paneW - 4
	}

	mainTitle := ""
	if b != nil && b.ActiveTable != "" {
		if b.Data != nil {
			mainTitle = fmt.Sprintf("%s · %s", b.ActiveTable, b.Data.Summary())
		} else if b.TotalRows > 0 {
			noun := "rows"
			if b.TotalRows == 1 {
				noun = "row"
			}
			mainTitle = fmt.Sprintf("%s · %d %s", b.ActiveTable, b.TotalRows, noun)
		} else {
			mainTitle = fmt.Sprintf("%s · (empty)", b.ActiveTable)
		}
	}
	mainBodyH := mainH - 2
	var mainContent string
	if b == nil {
		mainContent = styles.StyleHeaderDim.Render("no connection")
	} else if structure {
		mainContent = RenderStructure(b, contentW(rightW), mainBodyH)
	} else {
		mainContent = RenderMainBrowser(b, contentW(rightW), mainBodyH)
	}

	editorContent := es.View(e, contentW(rightW), editorH-2)

	editorTitle := "QUERY EDITOR"
	if es.Title() != "" {
		editorTitle = es.Title()
	}

	mainPane := boxed(mainContent, mainTitle, rightW, mainH, focus == FocusMain)
	editorPane := boxed(editorContent, editorTitle, rightW, editorH, focus == FocusEditor)
	right := lipgloss.JoinVertical(lipgloss.Top, mainPane, " ", editorPane)

	if !showSidebar {
		return right
	}

	sidebarTitle := "TABLES"
	if b != nil && b.CatalogTitle != "" {
		sidebarTitle = b.CatalogTitle
	}

	sidebarContent := RenderSidebar(b, sidebarCursor, contentW(sidebarW), height-2)
	sidebarPane := boxed(sidebarContent, sidebarTitle, sidebarW, height, focus == FocusSidebar)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebarPane, gap, right)
}

func boxed(content, title string, width, height int, focused bool) string {
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
	box := style.Width(innerW).Height(innerH).Padding(0, 1).Render(content)
	if title == "" {
		return box
	}
	lines := strings.Split(box, "\n")
	lines[0] = topBorder(title, width, focused)
	return strings.Join(lines, "\n")
}

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
