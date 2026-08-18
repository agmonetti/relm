package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/store"
	"github.com/agmonetti/relm/internal/tui/screens"
)

// handleWorkspaceKeys dispatches keys of the single working screen.
func (m *Model) handleWorkspaceKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	typingEditor := m.focus == screens.FocusEditor

	switch {
	case key.Matches(msg, m.keys.ToggleSidebar):
		m.showSidebar = !m.showSidebar
		return m, nil
	case key.Matches(msg, m.keys.FocusSidebar):
		m.setFocus(screens.FocusSidebar)
		return m, nil
	case key.Matches(msg, m.keys.FocusMain):
		m.setFocus(screens.FocusMain)
		return m, nil
	case key.Matches(msg, m.keys.FocusEditor):
		m.setFocus(screens.FocusEditor)
		return m, nil
	case key.Matches(msg, m.keys.Switch):
		m.cycleFocus()
		return m, nil
	case !typingEditor && key.Matches(msg, m.keys.Inspect):
		m.structure = true
		m.setFocus(screens.FocusMain)
		return m, nil
	case !typingEditor && key.Matches(msg, m.keys.Refresh) && m.browser != nil:
		return m, m.runBrowserOp(func(b *browser.Browser, st store.Store, ctx context.Context) error {
			return b.Reload(ctx, st)
		})
	case !m.loading && !m.navigating && key.Matches(msg, m.keys.Export):
		m.openExport()
		return m, nil
	}

	switch m.focus {
	case screens.FocusSidebar:
		return m.handleSidebarKeys(msg)
	case screens.FocusMain:
		return m.handleMainKeys(msg)
	case screens.FocusEditor:
		return m.handleEditorKeys(msg)
	}
	return m, nil
}

// cycleFocus moves the focus to the next workspace pane.
func (m *Model) cycleFocus() {
	var next screens.WorkspaceFocus
	switch m.focus {
	case screens.FocusSidebar:
		next = screens.FocusMain
	case screens.FocusMain:
		next = screens.FocusEditor
	case screens.FocusEditor:
		next = screens.FocusSidebar
	}
	m.setFocus(next)
}

// setFocus moves the focus to a pane, keeping the editor's textarea state in
// sync.
func (m *Model) setFocus(f screens.WorkspaceFocus) {
	if f == screens.FocusEditor {
		m.editorScreen.Focus()
	} else {
		m.editorScreen.Blur()
	}
	m.focus = f
}

// handleSidebarKeys handles keys when the sidebar has the focus.
func (m *Model) handleSidebarKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.browser == nil {
		return m, nil
	}
	b := m.browser
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.sidebarCursor > 0 {
			m.sidebarCursor--
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.sidebarCursor < len(b.Tables)-1 {
			m.sidebarCursor++
		}
		return m, nil
	case key.Matches(msg, m.keys.PageUp):
		m.sidebarCursor -= 10
		m.clampSidebar()
		return m, nil
	case key.Matches(msg, m.keys.PageDown):
		m.sidebarCursor += 10
		m.clampSidebar()
		return m, nil
	case key.Matches(msg, m.keys.First):
		m.sidebarCursor = 0
		return m, nil
	case key.Matches(msg, m.keys.Last):
		if len(b.Tables) > 0 {
			m.sidebarCursor = len(b.Tables) - 1
		}
		return m, nil
	case msg.Type == tea.KeyEnter:
		return m, m.selectTable(m.sidebarCursor)
	case msg.Type == tea.KeyRunes && !msg.Alt && len(msg.Runes) == 1 &&
		msg.Runes[0] >= '1' && msg.Runes[0] <= '9':
		idx := int(msg.Runes[0] - '1')
		return m, m.selectTable(idx)
	}
	return m, nil
}

// selectTable opens the table at index idx in the sidebar, loading its data in
// the background.
func (m *Model) selectTable(idx int) tea.Cmd {
	if m.browser == nil || idx < 0 || idx >= len(m.browser.Tables) {
		return nil
	}
	m.sidebarCursor = idx
	m.structure = false
	name := m.browser.Tables[idx]
	return m.runBrowserOp(func(b *browser.Browser, st store.Store, ctx context.Context) error {
		return b.SelectTable(ctx, name, st)
	})
}

// clampSidebar keeps the sidebar cursor inside the table list.
func (m *Model) clampSidebar() {
	if m.browser == nil {
		return
	}
	if n := len(m.browser.Tables); n == 0 {
		m.sidebarCursor = 0
	} else if m.sidebarCursor >= n {
		m.sidebarCursor = n - 1
	} else if m.sidebarCursor < 0 {
		m.sidebarCursor = 0
	}
}

// handleMainKeys handles keys when the main pane has the focus.
func (m *Model) handleMainKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.browser == nil {
		return m, nil
	}
	b := m.browser
	if m.navigating {
		// a navigation is in flight: ignore row/page keys until it lands
		return m, nil
	}
	switch {
	case key.Matches(msg, m.keys.Back):
		if m.structure {
			m.structure = false
		}
		return m, nil
	case key.Matches(msg, m.keys.Detail):
		if !m.structure && len(b.Rows) > 0 && b.Cursor >= 0 && b.Cursor < len(b.Rows) {
			cols := make([]string, len(b.Columns))
			for i, c := range b.Columns {
				cols[i] = c.Name
			}
			m.openDetail(b.ActiveTable, cols, b.Rows[b.Cursor])
		}
		return m, nil
	case key.Matches(msg, m.keys.Up):
		b.MoveCursor(-1)
		return m, nil
	case key.Matches(msg, m.keys.Down):
		b.MoveCursor(1)
		return m, nil
	case key.Matches(msg, m.keys.PageUp):
		return m, m.runBrowserOp(func(nb *browser.Browser, st store.Store, ctx context.Context) error {
			return nb.PrevPage(ctx, st)
		})
	case key.Matches(msg, m.keys.PageDown):
		return m, m.runBrowserOp(func(nb *browser.Browser, st store.Store, ctx context.Context) error {
			return nb.NextPage(ctx, st)
		})
	case key.Matches(msg, m.keys.First):
		b.MoveCursor(-len(b.Rows))
		return m, nil
	case key.Matches(msg, m.keys.Last):
		b.MoveCursor(len(b.Rows))
		return m, nil
	}
	return m, nil
}
