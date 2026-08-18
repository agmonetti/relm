package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/agmonetti/relm/internal/conn"
	"github.com/agmonetti/relm/internal/editor"
	"github.com/agmonetti/relm/internal/store"
	"github.com/agmonetti/relm/internal/tui/screens"
)

// newEditorWithPersistedHistory creates an editor whose history is preloaded
// from the previous sessions, so the last queries survive a restart.
func newEditorWithPersistedHistory() *editor.Editor {
	e := editor.New()
	if items := editor.LoadHistory(); items != nil {
		for _, q := range items {
			e.History.Push(q)
		}
	}
	return e
}

// doConnect opens the store and starts loading the browser. The store opening
// (with the engine's connection timeout) is synchronous; the schema and first
// page are loaded in the background so connecting to a big or slow database
// shows a spinner instead of freezing the UI.
func (m *Model) doConnect(cfg conn.ConnectionConfig) tea.Cmd {
	// a global --read-only forces every connection, including saved ones
	cfg.ReadOnly = cfg.ReadOnly || m.opts.GlobalReadOnly
	m.cancelNav()
	m.cancelConnect()
	m.cancelQuery()
	m.closeStore()
	m.queryID++
	m.navID = 0
	m.loading = false
	m.navigating = false
	m.connecting = false
	m.resizing = false
	m.showDetail = false
	m.exporting = false
	m.exported = ""
	m.warn = ""
	m.editorScreen.ResetResult()
	m.browser = nil
	m.editor = newEditorWithPersistedHistory()
	m.editorScreen.SetValue("")
	m.cfgLabel = cfg.Label()
	m.err = ""
	m.structure = false
	m.sidebarCursor = 0
	m.setFocus(screens.FocusSidebar)

	st, err := store.New(cfg)
	if err != nil {
		m.connect.SetError(err.Error())
		m.screen = ScreenConnect
		return nil
	}
	m.store = st
	// read-only is not enforce-able per session on SQL Server: say so instead
	// of pretending the flag protects the database.
	if cfg.ReadOnly && cfg.Driver == conn.DriverMSSQL {
		m.warn = "read-only is not enforced on mssql — connect with a read-only user"
	}
	return m.loadBrowserCmd(st)
}

// saveConnection persists the current connection.
func (m *Model) saveConnection(cfg conn.ConnectionConfig) {
	if cfg.Name == "" {
		cfg.Name = defaultName(cfg)
	}
	saved, err := conn.LoadSaved()
	if err == nil {
		saved = conn.SaveNamed(saved, cfg)
		err = conn.SaveSaved(saved)
	}
	if err != nil {
		m.connect.SetError(err.Error())
		return
	}
	m.connect.SetSaved(saved)
}

func defaultName(cfg conn.ConnectionConfig) string {
	if cfg.Driver == conn.DriverSQLite {
		return cfg.Path
	}
	return fmt.Sprintf("%s:%d/%s", cfg.Host, cfg.Port, cfg.Database)
}

// deleteConnection removes a saved connection by name.
func (m *Model) deleteConnection(name string) {
	if name == "" {
		return
	}
	saved, err := conn.LoadSaved()
	if err != nil {
		m.connect.SetError(err.Error())
		return
	}
	filtered := saved[:0]
	for _, s := range saved {
		if s.Name != name {
			filtered = append(filtered, s)
		}
	}
	if err := conn.SaveSaved(filtered); err != nil {
		m.connect.SetError(err.Error())
		return
	}
	m.connect.SetSaved(filtered)
}

// newSession closes the session and returns to the connection screen.
func (m *Model) newSession() {
	m.cancelNav()
	m.cancelConnect()
	m.cancelQuery()
	m.closeStore()
	m.queryID++
	m.navID = 0
	m.loading = false
	m.navigating = false
	m.connecting = false
	m.resizing = false
	m.showDetail = false
	m.exporting = false
	m.exported = ""
	m.warn = ""
	m.editorScreen.ResetResult()
	m.browser = nil
	m.editor = newEditorWithPersistedHistory()
	m.editorScreen.SetValue("")
	m.cfgLabel = ""
	m.err = ""
	m.structure = false
	m.sidebarCursor = 0
	m.setFocus(screens.FocusSidebar)
	m.screen = ScreenConnect
	m.connect.ResetForm()
}

func (m *Model) closeStore() {
	if m.store != nil {
		m.store.Close()
		m.store = nil
	}
}

// openSettings opens the preferences screen, remembering where to return.
func (m *Model) openSettings() {
	m.prevScreen = m.screen
	m.screen = ScreenSettings
	m.settings.SetValue(fmt.Sprintf("%d", m.prefs.QueryTimeout()))
	m.settings.SetError("")
	m.settings.Focus()
}

// leaveSettings returns to the screen the user was on before settings.
func (m *Model) leaveSettings() {
	m.screen = m.prevScreen
	m.settings.Blur()
}
