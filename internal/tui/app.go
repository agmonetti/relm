// Package tui implements the bubbletea loop for relm.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/conn"
	"github.com/agmonetti/relm/internal/editor"
	"github.com/agmonetti/relm/internal/store"
	"github.com/agmonetti/relm/internal/tui/screens"
	"github.com/agmonetti/relm/internal/tui/styles"
)

// Screen identifies the active screen.
type Screen int

const (
	// ScreenConnect is the connection screen (first on open).
	ScreenConnect Screen = iota
	// ScreenWorkspace is the single working screen: sidebar + main + editor.
	ScreenWorkspace
)

// editorDoneMsg carries the result of a query run in the background.
type editorDoneMsg struct {
	ed    *editor.Editor
	err   error
	token int
}

// Model is the main bubbletea model.
type Model struct {
	store        store.Store
	connect      *screens.ConnScreen
	browser      *browser.Browser
	editor       *editor.Editor
	editorScreen *screens.EditorScreen
	screen       Screen
	focus        screens.WorkspaceFocus
	keys         KeyMap
	cfgLabel     string

	// workspace state
	structure     bool // the main pane shows the structure of the active table
	sidebarCursor int  // table selected in the sidebar

	spinner spinner.Model
	loading bool
	queryID int // token to discard results from stale queries

	width       int
	height      int
	showSidebar bool
	showHelp    bool
	err         string
}

// New creates the initial model (connection screen).
func New() *Model {
	saved, _ := conn.LoadSaved()
	return &Model{
		connect:      screens.NewConnScreen(saved),
		editor:       editor.New(),
		editorScreen: screens.NewEditorScreen(),
		screen:       ScreenConnect,
		focus:        screens.FocusSidebar,
		keys:         DefaultKeyMap(),
		spinner:      spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(styles.StyleHeaderDim)),
		showSidebar:  true,
	}
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

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
			m.newSession()
			return m, nil
		}

		switch m.screen {
		case ScreenConnect:
			return m.handleConnectKeys(msg)
		case ScreenWorkspace:
			return m.handleWorkspaceKeys(msg)
		}

	case screens.ConnectMsg:
		m.doConnect(msg.Cfg)

	case screens.SaveConnectionMsg:
		m.saveConnection(msg.Cfg)

	case editorDoneMsg:
		if msg.token != m.queryID {
			return m, nil // stale result: the session changed
		}
		m.loading = false
		m.editor = msg.ed
		m.setErr(msg.err)
		m.editorScreen.Focus()

		// A write query (INSERT/UPDATE/DELETE/CREATE/DROP/...) may have changed
		// the schema or the data: refresh the table list and the open table.
		if msg.err == nil && m.browser != nil && m.store != nil &&
			msg.ed.Result != nil && msg.ed.Result.Affected >= 0 {
			m.setErr(m.browser.Reload(m.store))
			m.clampSidebar()
		}
		return m, nil

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m *Model) View() string {
	if m.showHelp {
		return m.renderHelp()
	}
	return m.render()
}

func (m *Model) render() string {
	header := m.renderHeader()
	footer := m.renderFooter()
	contentHeight := m.height - 2
	if contentHeight < 1 {
		contentHeight = 1
	}

	var content string
	switch m.screen {
	case ScreenConnect:
		content = m.connect.View(m.width, contentHeight)
	case ScreenWorkspace:
		content = screens.RenderWorkspace(m.browser, m.editorScreen, m.editor,
			m.focus, m.structure, m.showSidebar, m.sidebarCursor, m.width, contentHeight)
	}

	body := content
	if m.err != "" {
		body = content + "\n" + styles.StyleError.Render(m.err)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m *Model) renderHeader() string {
	label := "no connection"
	if m.store != nil {
		label = m.cfgLabel
	}
	mode := ""
	switch m.screen {
	case ScreenWorkspace:
		switch {
		case m.focus == screens.FocusEditor:
			mode = "editor"
		case m.focus == screens.FocusSidebar:
			mode = "tables"
		case m.structure:
			mode = "structure"
		default:
			mode = "browser"
		}
	}
	table := "—"
	if m.browser != nil && m.browser.ActiveTable != "" {
		table = m.browser.ActiveTable
	}
	return styles.StyleHeader.Render("relm") +
		styles.StyleHeaderDim.Render(" · "+label) +
		styles.StyleHeaderDim.Render(" · "+table) +
		styles.StyleHeaderDim.Render(" · "+mode)
}

func (m *Model) renderFooter() string {
	left := ""
	switch m.screen {
	case ScreenConnect:
		left = "↑↓ saved · tab engine/fields · ←→ engine · enter connect · ctrl+s save"
	case ScreenWorkspace:
		switch m.focus {
		case screens.FocusSidebar:
			left = "↑↓ tables · enter open · tab next · ctrl+1/2/3 focus · ? help"
		case screens.FocusMain:
			if m.structure {
				left = "esc back · tab next"
			} else {
				left = "↑↓ rows · i structure · r refresh · pgup/pgdn page · tab next"
			}
		case screens.FocusEditor:
			left = "ctrl+r run · ctrl+l clear · esc back"
		}
	}

	right := ""
	if m.loading {
		right = m.spinner.View() + " running query…"
	} else if m.screen == ScreenWorkspace && m.focus != screens.FocusEditor &&
		m.browser != nil && m.browser.ActiveTable != "" && m.browser.TotalRows > 0 {
		page := m.browser.Page + 1
		arrow := ""
		if m.browser.HasNextPage() {
			arrow = " ▼"
		}
		right = fmt.Sprintf("%d/%d%s", page, m.browser.TotalRows, arrow)
	}

	pad := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}
	return styles.StyleFooter.Render(left + padSpaces(pad) + right)
}

// typing reports whether the user is typing text (q/? must not quit).
func (m *Model) typing() bool {
	switch m.screen {
	case ScreenConnect:
		return m.connect.FocusOnField()
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

// doConnect opens the store and loads the browser.
func (m *Model) doConnect(cfg conn.ConnectionConfig) {
	m.closeStore()
	m.queryID++
	m.loading = false
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
	return fmt.Sprintf("%s/%s", cfg.Host, cfg.Database)
}

// newSession closes the session and returns to the connection screen.
func (m *Model) newSession() {
	m.closeStore()
	m.queryID++
	m.loading = false
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
		m.setErr(m.browser.Reload(m.store))
		m.clampSidebar()
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
	case msg.Type == tea.KeyEnter:
		m.selectTable(m.sidebarCursor)
		return m, nil
	case msg.Type == tea.KeyRunes && !msg.Alt && len(msg.Runes) == 1 &&
		msg.Runes[0] >= '1' && msg.Runes[0] <= '9':
		idx := int(msg.Runes[0] - '1')
		m.selectTable(idx)
		return m, nil
	}
	return m, nil
}

// selectTable opens the table at index idx in the sidebar.
func (m *Model) selectTable(idx int) {
	if m.browser == nil || idx < 0 || idx >= len(m.browser.Tables) {
		return
	}
	m.sidebarCursor = idx
	m.structure = false
	m.setErr(m.browser.SelectTable(m.browser.Tables[idx], m.store))
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
	switch {
	case key.Matches(msg, m.keys.Back):
		if m.structure {
			m.structure = false
		}
		return m, nil
	case key.Matches(msg, m.keys.Up):
		b.MoveCursor(-1)
		return m, nil
	case key.Matches(msg, m.keys.Down):
		b.MoveCursor(1)
		return m, nil
	case key.Matches(msg, m.keys.PageUp):
		m.setErr(b.PrevPage(m.store))
		return m, nil
	case key.Matches(msg, m.keys.PageDown):
		m.setErr(b.NextPage(m.store))
		return m, nil
	case key.Matches(msg, m.keys.First):
		b.MoveCursor(-len(b.Rows))
		return m, nil
	case key.Matches(msg, m.keys.Last):
		b.MoveCursor(len(b.Rows))
		return m, nil
	}
	return m, nil
}

// handleEditorKeys handles keys when the editor has the focus.
func (m *Model) handleEditorKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Back):
		m.setFocus(screens.FocusMain)
		return m, nil
	case key.Matches(msg, m.keys.Execute):
		if m.loading {
			return m, nil
		}
		if strings.TrimSpace(m.editorScreen.Value()) == "" {
			m.editor.Result = nil
			m.editor.Error = "write a query first"
			return m, nil
		}
		return m, m.executeEditor()
	case key.Matches(msg, m.keys.ClearInput):
		m.editor.Clear()
		m.editorScreen.SetValue("")
		return m, nil
	case key.Matches(msg, m.keys.Up) && !m.loading && m.editorScreen.AtBoundary(true) &&
		(m.editorScreen.Value() == "" || m.editor.History.InNavigation()):
		m.editor.Buffer = m.editor.History.Prev()
		m.editorScreen.SetValue(m.editor.Buffer)
		m.editorScreen.FocusStart()
		return m, nil
	case key.Matches(msg, m.keys.Down) && !m.loading && m.editorScreen.AtBoundary(false) &&
		m.editor.History.InNavigation():
		m.editor.Buffer = m.editor.History.Next()
		m.editorScreen.SetValue(m.editor.Buffer)
		m.editorScreen.Focus()
		return m, nil
	}

	// any other key stops navigating the history
	m.editor.History.Reset()
	updated, cmd := m.editorScreen.Update(msg)
	m.editorScreen = updated
	m.editor.Buffer = m.editorScreen.Value()
	return m, cmd
}

// executeEditor runs the current query in the background and shows a spinner.
func (m *Model) executeEditor() tea.Cmd {
	if m.store == nil {
		return nil
	}
	buf := m.editorScreen.Value()
	line := m.editorScreen.Line() // statement under the cursor
	st := m.store
	token := m.queryID
	m.loading = true
	return tea.Batch(
		func() tea.Msg {
			// fresh editor for the query, but shares the accumulated
			// history with the model (the ring buffer persists across runs).
			ed := editor.New()
			ed.History = m.editor.History
			ed.Buffer = buf
			err := ed.ExecuteAt(st, line)
			return editorDoneMsg{ed: ed, err: err, token: token}
		},
		m.spinner.Tick,
	)
}

// setErr stores or clears the current error message.
func (m *Model) setErr(err error) {
	if err != nil {
		m.err = err.Error()
	} else {
		m.err = ""
	}
}

func (m *Model) renderHelp() string {
	var out string
	for _, group := range m.keys.FullHelp() {
		for _, b := range group {
			out += fmt.Sprintf("  %-18s %s\n", b.Help().Key, b.Help().Desc)
		}
		out += "\n"
	}
	return lipgloss.JoinVertical(lipgloss.Left,
		styles.StyleHeader.Render("Help"),
		out,
		styles.StyleHeaderDim.Render("? to close"),
	)
}

func padSpaces(n int) string {
	if n < 0 {
		n = 0
	}
	return fmt.Sprintf("%*s", n, "")
}
