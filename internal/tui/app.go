// Package tui implements the bubbletea loop for relm.
package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	term "github.com/charmbracelet/x/term"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/conn"
	"github.com/agmonetti/relm/internal/editor"
	"github.com/agmonetti/relm/internal/prefs"
	"github.com/agmonetti/relm/internal/store"
	"github.com/agmonetti/relm/internal/tui/screens"
	"github.com/agmonetti/relm/internal/tui/styles"
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

// Screen identifies the active screen.
type Screen int

const (
	// ScreenConnect is the connection screen (first on open).
	ScreenConnect Screen = iota
	// ScreenWorkspace is the single working screen: sidebar + main + editor.
	ScreenWorkspace
	// ScreenSettings is the preferences screen.
	ScreenSettings
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
	settings     *screens.SettingsScreen
	browser      *browser.Browser
	editor       *editor.Editor
	editorScreen *screens.EditorScreen
	screen       Screen
	prevScreen   Screen // screen to return to from ScreenSettings
	focus        screens.WorkspaceFocus
	keys         KeyMap
	cfgLabel     string
	prefs        prefs.Prefs

	// workspace state
	structure     bool // the main pane shows the structure of the active table
	sidebarCursor int  // table selected in the sidebar

	spinner spinner.Model
	loading bool
	queryID int    // token to discard results from stale queries
	cancel  context.CancelFunc // cancels the running query

	// workspace pane sizes (0 = auto), resizable with a right-click drag
	sidebarW int
	editorH  int
	resizing bool
	resizeDiv int // resizeNone, resizeSidebar or resizeEditor

	width       int
	height      int
	showSidebar bool
	showHelp    bool
	err         string
}

// Divider targeted by a right-click drag.
const (
	resizeNone int = iota
	resizeSidebar
	resizeEditor
)

// New creates the initial model (connection screen).
func New() *Model {
	saved, _ := conn.LoadSaved()
	p, _ := prefs.Load()
	return &Model{
		connect:      screens.NewConnScreen(saved),
		settings:     screens.NewSettingsScreen(),
		editor:       editor.New(),
		editorScreen: screens.NewEditorScreen(),
		screen:       ScreenConnect,
		prevScreen:   ScreenConnect,
		focus:        screens.FocusSidebar,
		keys:         DefaultKeyMap(),
		spinner:      spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(styles.StyleHeaderDim)),
		showSidebar:  true,
		prefs:        p,
		sidebarW:     p.SidebarWidth,
		editorH:      p.EditorHeight,
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
			if m.screen != ScreenSettings {
				m.newSession()
			}
			return m, nil
		case "ctrl+p":
			if m.screen != ScreenSettings {
				m.openSettings()
			}
			return m, nil
		}

		// while a query runs, Esc cancels it instead of the pane default
		if m.loading && msg.Type == tea.KeyEsc {
			m.cancelQuery()
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
		m.doConnect(msg.Cfg)

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

	case tea.MouseMsg:
		m.handleMouse(msg)
		return m, nil
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
	innerW := m.width - 2 // 1 char outer margin on each side
	if innerW < 1 {
		innerW = 1
	}

	header := m.renderHeader()
	footer := m.renderFooter()
	// the layout is wrapped in a 1-char margin on all four sides: a blank
	// line above the header and below the footer, plus the lateral padding
	contentHeight := m.height - 4
	if contentHeight < 1 {
		contentHeight = 1
	}

	var content string
	switch m.screen {
	case ScreenConnect:
		content = m.connect.View(innerW, contentHeight)
	case ScreenSettings:
		content = m.settings.View(innerW, contentHeight)
	case ScreenWorkspace:
		layout := screens.ComputeLayout(innerW, contentHeight, m.showSidebar, m.sidebarW, m.editorH)
		content = screens.RenderWorkspace(m.browser, m.editorScreen, m.editor,
			m.focus, m.structure, layout, m.sidebarCursor, innerW, contentHeight)
	}

	body := content
	if m.err != "" {
		body = content + "\n" + styles.StyleError.Render(m.err)
	}
	body = styles.StyleOuterMargin.Render(body)

	return lipgloss.JoinVertical(lipgloss.Left,
		" ",
		" "+header+" ",
		body,
		" "+footer+" ",
		" ")
}

func (m *Model) renderHeader() string {
	label := "no connection"
	if m.store != nil {
		label = m.cfgLabel
	}
	mode := ""
	switch m.screen {
	case ScreenSettings:
		mode = "settings"
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
		left = "↑↓ saved · tab engine/fields · ←→ engine · enter connect · ctrl+s save · d delete · ctrl+p settings"
	case ScreenSettings:
		left = "enter save · esc back"
	case ScreenWorkspace:
		switch m.focus {
		case screens.FocusSidebar:
			left = "↑↓ tables · enter open · tab next · ? help · ctrl+p settings"
		case screens.FocusMain:
			if m.structure {
				left = "esc back · tab next"
			} else {
				left = "↑↓ rows · i structure · r refresh · pgup/pgdn page · tab next · right-click resize"
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
		first := m.browser.Page*m.browser.PageSize + 1
		last := (m.browser.Page + 1) * m.browser.PageSize
		if last > m.browser.TotalRows {
			last = m.browser.TotalRows
		}
		arrow := ""
		if m.browser.HasNextPage() {
			arrow = " ▼"
		}
		right = fmt.Sprintf("%d-%d/%d%s", first, last, m.browser.TotalRows, arrow)
	}

	pad := (m.width - 2) - lipgloss.Width(left) - lipgloss.Width(right)
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

// cancelQuery cancels a running query, if any.
func (m *Model) cancelQuery() {
	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
	}
}

// handleMouse handles mouse events in the workspace: a left click focuses the
// clicked pane and a right-click drag resizes the nearest pane divider.
func (m *Model) handleMouse(msg tea.MouseMsg) {
	if m.screen != ScreenWorkspace || m.showHelp {
		return
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

// friendlyErr maps context cancellation to the message shown to the user.
func friendlyErr(err error) error {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return errors.New("query timed out")
	case errors.Is(err, context.Canceled):
		return errors.New("query cancelled")
	}
	return err
}

// doConnect opens the store and loads the browser.
func (m *Model) doConnect(cfg conn.ConnectionConfig) {
	m.cancelQuery()
	m.closeStore()
	m.queryID++
	m.loading = false
	m.resizing = false
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
	case key.Matches(msg, m.keys.First):
		m.sidebarCursor = 0
		return m, nil
	case key.Matches(msg, m.keys.Last):
		if len(b.Tables) > 0 {
			m.sidebarCursor = len(b.Tables) - 1
		}
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
// The query is bounded by the configured timeout and can be cancelled with Esc.
func (m *Model) executeEditor() tea.Cmd {
	if m.store == nil {
		return nil
	}
	buf := m.editorScreen.Value()
	line := m.editorScreen.Line() // statement under the cursor
	st := m.store
	token := m.queryID
	timeout := time.Duration(m.prefs.QueryTimeout()) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	m.cancelQuery() // release a previous cancel, if any
	m.cancel = cancel
	m.loading = true
	return tea.Batch(
		func() tea.Msg {
			// The query runs on a fresh editor with a throwaway history: the
			// persistent ring buffer is only touched from the UI goroutine (in
			// editorDoneMsg), so the two never race.
			ed := editor.New()
			ed.Buffer = buf
			err := ed.ExecuteAt(ctx, st, line)
			if err != nil {
				ed.Error = friendlyErr(err).Error()
			}
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
	out += fmt.Sprintf("  %-18s %s\n", "right-click drag", "resize panes")
	out += fmt.Sprintf("  %-18s %s\n", "click", "focus pane")
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
