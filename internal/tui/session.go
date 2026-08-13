package tui

import (
	"fmt"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/conn"
	"github.com/agmonetti/relm/internal/editor"
	"github.com/agmonetti/relm/internal/store"
	"github.com/agmonetti/relm/internal/tui/screens"
)

// doConnect opens the store and loads the browser.
func (m *Model) doConnect(cfg conn.ConnectionConfig) {
	m.cancelQuery()
	m.closeStore()
	m.queryID++
	m.loading = false
	m.resizing = false
	m.showDetail = false
	m.editorScreen.ResetResult()
	m.browser = nil
	m.editor = editor.New()
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
		return
	}
	m.store = st

	b, err := browser.New(st)
	if err != nil {
		st.Close()
		m.store = nil
		m.connect.SetError(err.Error())
		m.screen = ScreenConnect
		return
	}
	m.browser = b
	m.screen = ScreenWorkspace
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
	m.cancelQuery()
	m.closeStore()
	m.queryID++
	m.loading = false
	m.resizing = false
	m.showDetail = false
	m.editorScreen.ResetResult()
	m.browser = nil
	m.editor = editor.New()
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
