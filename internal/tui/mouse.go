package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/store"
	"github.com/agmonetti/relm/internal/tui/screens"
)

// handleMouse handles mouse events. On the connection screen a left click
// focuses the clicked field / engine, toggles a checkbox, connects or opens a
// saved connection. In the workspace a left click focuses and selects the
// clicked row, a right-click drag resizes the nearest pane divider and the
// wheel scrolls the pane under the pointer. Page changes from the wheel run in
// the background; the returned cmd carries the navigation.
func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	if m.showHelp {
		return nil
	}
	switch m.screen {
	case ScreenConnect:
		return m.handleConnectMouse(msg)
	case ScreenSettings:
		return nil
	case ScreenWorkspace:
		return m.handleWorkspaceMouse(msg)
	}
	return nil
}

// handleConnectMouse handles left clicks on the connection screen. The screen
// content starts at terminal cell (1,2): the frame adds a blank line, the
// header and a one-column left margin.
func (m *Model) handleConnectMouse(msg tea.MouseMsg) tea.Cmd {
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil
	}
	return m.connect.Activate(m.connect.HitTest(msg.X-1, msg.Y-2))
}

// handleWorkspaceMouse handles mouse events inside the workspace.
func (m *Model) handleWorkspaceMouse(msg tea.MouseMsg) tea.Cmd {
	if m.showDetail {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.detailScroll--
		case tea.MouseButtonWheelDown:
			m.detailScroll++
		}
		return nil
	}
	if m.navigating {
		// a navigation is in flight: ignore wheel/click row ops until it lands
		return nil
	}
	innerW := m.width - 2
	if innerW < 1 {
		innerW = 1
	}
	contentHeight := m.height - 4
	if contentHeight < 1 {
		contentHeight = 1
	}
	layout := screens.ComputeLayout(innerW, contentHeight, m.showSidebar, m.sidebarW, m.editorH)

	// the workspace content starts at terminal cell (1, 2): the frame adds a
	// blank line, the header and a one-column left margin
	wx := msg.X - 1
	wy := msg.Y - 2

	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		return m.scrollAt(wx, wy, layout, msg.Button)
	}

	switch msg.Action {
	case tea.MouseActionPress:
		switch msg.Button {
		case tea.MouseButtonRight:
			if div := m.pickResizeDivider(wx, wy, layout); div != resizeNone {
				m.resizing = true
				m.resizeDiv = div
			}
		case tea.MouseButtonLeft:
			m.focusPaneAt(wx, wy, layout)
			m.selectAt(wx, wy, layout)
		}
	case tea.MouseActionMotion:
		if m.resizing {
			m.applyResize(wx, wy, layout)
		}
	case tea.MouseActionRelease:
		if m.resizing {
			m.resizing = false
			m.resizeDiv = resizeNone
			m.persistLayout()
		}
	}
	return nil
}

// scrollAt scrolls the pane under the pointer with the mouse wheel.
func (m *Model) scrollAt(wx, wy int, layout screens.WorkspaceLayout, btn tea.MouseButton) tea.Cmd {
	delta := 0
	switch btn {
	case tea.MouseButtonWheelUp:
		delta = -1
	case tea.MouseButtonWheelDown:
		delta = 1
	default:
		return nil
	}
	const wheelStep = 3

	switch {
	case layout.ShowSidebar && wx < layout.SidebarW:
		m.scrollSidebar(delta * wheelStep)
	case wy > layout.MainH:
		m.scrollResults(delta*wheelStep, layout)
	default:
		return m.scrollMain(delta * wheelStep)
	}
	return nil
}

func (m *Model) scrollSidebar(delta int) {
	if m.browser == nil {
		return
	}
	m.sidebarCursor += delta
	if m.sidebarCursor < 0 {
		m.sidebarCursor = 0
	}
	if n := len(m.browser.Tables); m.sidebarCursor >= n {
		m.sidebarCursor = n - 1
	}
}

func (m *Model) scrollMain(delta int) tea.Cmd {
	b := m.browser
	if b == nil || len(b.Rows) == 0 {
		return nil
	}
	if delta < 0 && b.Cursor == 0 && b.HasPrevPage() {
		// land on the last row of the previous page after the async swap
		return m.runBrowserOp(func(nb *browser.Browser, st store.DataSource, ctx context.Context) error {
			if err := nb.PrevPage(ctx, st); err != nil {
				return err
			}
			nb.MoveCursor(len(nb.Rows))
			return nil
		})
	}
	if delta > 0 && b.Cursor >= len(b.Rows)-1 && b.HasNextPage() {
		return m.runBrowserOp(func(nb *browser.Browser, st store.DataSource, ctx context.Context) error {
			return nb.NextPage(ctx, st)
		})
	}
	b.MoveCursor(delta)
	return nil
}

func (m *Model) scrollResults(delta int, layout screens.WorkspaceLayout) {
	tab, ok := m.editor.Data.(*store.TabularData)
	if !ok || len(tab.Rows) == 0 {
		return
	}
	_, dataRows := screens.EditorResultsLayout(layout.EditorH - 2)
	maxScroll := len(tab.Rows) - dataRows
	if maxScroll < 0 {
		maxScroll = 0
	}
	m.editorScreen.SetResultScroll(m.editorScreen.ResultScroll() + delta)
	if s := m.editorScreen.ResultScroll(); s < 0 {
		m.editorScreen.SetResultScroll(0)
	} else if s > maxScroll {
		m.editorScreen.SetResultScroll(maxScroll)
	}
}

// selectAt selects the row or table under the pointer on a left click.
func (m *Model) selectAt(wx, wy int, layout screens.WorkspaceLayout) {
	switch {
	case layout.ShowSidebar && wx < layout.SidebarW:
		if m.browser == nil {
			return
		}
		offset, _ := screens.SidebarWindow(m.sidebarCursor, layout.MainH+layout.EditorH-1)
		idx := offset + (wy - 1)
		if idx >= 0 && idx < len(m.browser.Tables) {
			m.sidebarCursor = idx
		}
	case wy > layout.MainH:
		m.selectResultRow(wy, layout)
	default:
		if m.browser == nil {
			return
		}
		start, visible := screens.TableWindow(len(m.browser.Rows), m.browser.Cursor, layout.MainH-2)
		rel := wy - 2 // header line + top border
		if rel >= 0 && rel < visible {
			if row := start + rel; row < len(m.browser.Rows) {
				m.browser.Cursor = row
			}
		}
	}
}

// selectResultRow selects the query result row under the pointer.
func (m *Model) selectResultRow(wy int, layout screens.WorkspaceLayout) {
	tab, ok := m.editor.Data.(*store.TabularData)
	if !ok || len(tab.Rows) == 0 {
		return
	}
	editorTop := layout.MainH + 1
	startLine, dataRows := screens.EditorResultsLayout(layout.EditorH - 2)
	rel := (wy - editorTop - 1) - startLine - 1 // data row within the viewport
	if rel < 0 || rel >= dataRows {
		return
	}
	row := m.editorScreen.ResultScroll() + rel
	if row >= 0 && row < len(tab.Rows) {
		m.editorScreen.SetResultCursor(row)
	}
}

// pickResizeDivider returns the pane divider closest to the given workspace
// coordinates, or resizeNone when there is none to resize.
func (m *Model) pickResizeDivider(wx, wy int, layout screens.WorkspaceLayout) int {
	best := resizeNone
	bestDist := 1 << 30
	if layout.ShowSidebar {
		if d := absInt(wx - layout.SidebarW); d < bestDist {
			bestDist = d
			best = resizeSidebar
		}
	}
	if d := absInt(wy - layout.MainH); d < bestDist {
		best = resizeEditor
	}
	return best
}

// applyResize moves the divider being dragged to the pointer position. The
// stored value is clamped again when the next layout is computed.
func (m *Model) applyResize(wx, wy int, layout screens.WorkspaceLayout) {
	switch m.resizeDiv {
	case resizeSidebar:
		m.sidebarW = wx
	case resizeEditor:
		m.editorH = layout.MainH + layout.EditorH - wy
	}
}

// focusPaneAt focuses the pane under the pointer.
func (m *Model) focusPaneAt(wx, wy int, layout screens.WorkspaceLayout) {
	switch {
	case layout.ShowSidebar && wx < layout.SidebarW:
		m.setFocus(screens.FocusSidebar)
	case wy > layout.MainH:
		m.setFocus(screens.FocusEditor)
	default:
		m.setFocus(screens.FocusMain)
	}
}

// persistLayout stores the current pane sizes in the preferences.
func (m *Model) persistLayout() {
	m.prefs.SidebarWidth = m.sidebarW
	m.prefs.EditorHeight = m.editorH
	_ = m.prefs.Save() // best effort: layout is not critical
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
