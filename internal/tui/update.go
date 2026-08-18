package tui

import (
	"context"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	term "github.com/charmbracelet/x/term"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/store"
	"github.com/agmonetti/relm/internal/tui/screens"
)

// termGetSize is a seam for the size guard tests.
var termGetSize = term.GetSize

// correctSize returns a usable terminal size. Windows conhost reports the
// screen buffer (e.g. 9001 rows of scrollback) instead of the visible window
// in its resize events, so the raw value must not be trusted: when it looks
// bogus it is re-queried with the OS viewport and clamped as a safety net.
func correctSize(w, h int) (int, int) {
	if w < 10 || h < 3 || w > 500 || h > 1000 {
		if rw, rh, err := termGetSize(os.Stdout.Fd()); err == nil && rw >= 10 && rh >= 3 {
			w, h = rw, rh
		}
	}
	if w < 10 {
		w = 10
	}
	if h < 3 {
		h = 3
	}
	if w > 500 {
		w = 500
	}
	if h > 1000 {
		h = 1000
	}
	return w, h
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = correctSize(msg.Width, msg.Height)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q":
			if !m.typing() {
				return m, tea.Quit
			}
		case "?":
			if !m.typing() {
				m.showHelp = !m.showHelp
			}
			return m, nil
		case "ctrl+n":
			if !m.exporting && m.screen != ScreenSettings {
				m.newSession()
			}
			return m, nil
		case "ctrl+p":
			if !m.exporting && m.screen != ScreenSettings {
				m.openSettings()
			}
			return m, nil
		}

		// the export prompt owns every key while it is open
		if m.exporting {
			return m.handleExportKeys(msg)
		}

		// the detail view only accepts scrolling and Esc
		if m.showDetail {
			return m.handleDetailKeys(msg)
		}

		// while a query runs, Esc cancels it instead of the pane default
		if m.loading && msg.Type == tea.KeyEsc {
			m.cancelQuery()
			m.loading = false
			return m, nil
		}
		if m.navigating && msg.Type == tea.KeyEsc {
			m.cancelNav()
			m.navigating = false
			return m, nil
		}
		if m.connecting && msg.Type == tea.KeyEsc {
			m.cancelConnect()
			m.connecting = false
			return m, nil
		}
		// Esc also closes the help cheatsheet
		if m.showHelp && msg.Type == tea.KeyEsc {
			m.showHelp = false
			return m, nil
		}

		switch m.screen {
		case ScreenConnect:
			return m.handleConnectKeys(msg)
		case ScreenSettings:
			return m.handleSettingsKeys(msg)
		case ScreenWorkspace:
			return m.handleWorkspaceKeys(msg)
		}

	case screens.ConnectMsg:
		cmd := m.doConnect(msg.Cfg)
		return m, cmd

	case screens.SaveConnectionMsg:
		m.saveConnection(msg.Cfg)

	case screens.DeleteConnectionMsg:
		m.deleteConnection(msg.Name)

	case screens.SettingsMsg:
		m.prefs.QueryTimeoutSeconds = msg.QueryTimeoutSeconds
		if err := m.prefs.Save(); err != nil {
			m.settings.SetError(err.Error())
			return m, nil
		}
		m.leaveSettings()

	case screens.SettingsBackMsg:
		m.leaveSettings()

	case editorDoneMsg:
		m.cancelQuery()
		if msg.token != m.queryID {
			return m, nil // stale result: the session changed
		}
		m.loading = false
		m.editorScreen.ResetResult()
		// the goroutine ran against a throwaway history; keep the persistent
		// ring buffer and push the executed statement here, on the UI goroutine
		hist := m.editor.History
		m.editor = msg.ed
		m.editor.History = hist
		m.setErr(friendlyErr(msg.err))
		m.editorScreen.Focus()

		if msg.err == nil && msg.ed.LastQuery != "" {
			hist.Push(msg.ed.LastQuery)
		}

		// A write query (INSERT/UPDATE/DELETE/CREATE/DROP/...) may have changed
		// the schema or the data: refresh the table list and the open table in
		// the background, so a slow reload cannot freeze the UI.
		var cmds []tea.Cmd
		if msg.err == nil && m.browser != nil && m.store != nil && msg.ed.Wrote {
			if cmd := m.runBrowserOp(func(b *browser.Browser, st store.Store, ctx context.Context) error {
				return b.Reload(ctx, st)
			}); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		if len(cmds) == 0 {
			return m, nil
		}
		return m, tea.Batch(cmds...)

	case browserDoneMsg:
		if msg.load {
			if msg.token != m.queryID {
				return m, nil // stale result: the session changed
			}
			m.cancelConnect()
			m.connecting = false
		} else {
			if msg.navID != m.navID || msg.token != m.queryID {
				return m, nil // stale: superseded by a newer navigation or the session changed
			}
			m.cancelNav()
			m.navigating = false
		}
		if msg.err != nil {
			err := friendlyErr(msg.err)
			if msg.load {
				// the initial load failed: close the store and go back to the
				// connection screen with the engine error
				m.closeStore()
				m.connect.SetError(err.Error())
				m.screen = ScreenConnect
				return m, nil
			}
			m.setErr(err)
			return m, nil
		}
		m.browser = msg.b
		if msg.load {
			m.screen = ScreenWorkspace
		}
		m.clampSidebar()
		return m, nil

	case spinner.TickMsg:
		if m.busy() {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.MouseMsg:
		return m, m.handleMouse(msg)
	}

	return m, nil
}

// typing reports whether the user is typing text (q/? must not quit).
func (m *Model) typing() bool {
	if m.exporting {
		return true
	}
	switch m.screen {
	case ScreenConnect:
		return m.connect.FocusOnField()
	case ScreenSettings:
		return m.settings.FocusOnField()
	case ScreenWorkspace:
		return m.focus == screens.FocusEditor
	}
	return false
}

// handleConnectKeys handles keys of the connection screen.
func (m *Model) handleConnectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	updated, cmd := m.connect.Update(msg)
	m.connect = updated
	return m, cmd
}

// handleSettingsKeys handles keys of the settings screen.
func (m *Model) handleSettingsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	updated, cmd := m.settings.Update(msg)
	m.settings = updated
	return m, cmd
}

// setErr stores or clears the current error message.
func (m *Model) setErr(err error) {
	if err != nil {
		m.err = err.Error()
	} else {
		m.err = ""
	}
}
