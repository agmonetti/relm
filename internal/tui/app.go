// Package tui implementa el loop de bubbletea de relm.
package tui

import (
	"fmt"

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

// Screen identifica la pantalla activa.
type Screen int

const (
	// ScreenConnect es la pantalla de conexión (primera al abrir).
	ScreenConnect Screen = iota
	// ScreenBrowser es la pantalla de navegación de tablas y filas.
	ScreenBrowser
	// ScreenEditor es la pantalla del editor SQL.
	ScreenEditor
	// ScreenStructure es la pantalla de estructura de tabla.
	ScreenStructure
)

// editorDoneMsg transporta el resultado de una query ejecutada en background.
type editorDoneMsg struct {
	ed    *editor.Editor
	err   error
	token int
}

// Model es el modelo principal de bubbletea.
type Model struct {
	store        store.Store
	connect      *screens.ConnScreen
	browser      *browser.Browser
	editor       *editor.Editor
	editorScreen *screens.EditorScreen
	screen       Screen
	keys         KeyMap
	cfgLabel     string

	spinner spinner.Model
	loading bool
	queryID int // token para descartar resultados de queries obsoletas

	width       int
	height      int
	showSidebar bool
	showHelp    bool
	err         string
}

// New crea el modelo inicial (pantalla de conexión).
func New() *Model {
	saved, _ := conn.LoadSaved()
	return &Model{
		connect:      screens.NewConnScreen(saved),
		editor:       editor.New(),
		editorScreen: screens.NewEditorScreen(),
		screen:       ScreenConnect,
		keys:         DefaultKeyMap(),
		spinner:      spinner.New(spinner.WithSpinner(spinner.Dot), spinner.WithStyle(styles.StyleHeaderDim)),
		showSidebar:  true,
	}
}

// Init implementa tea.Model.
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update implementa tea.Model.
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
			m.showHelp = !m.showHelp
			return m, nil
		case "ctrl+n":
			m.newSession()
			return m, nil
		}

		switch m.screen {
		case ScreenConnect:
			return m.handleConnectKeys(msg)
		case ScreenBrowser:
			return m.handleBrowserKeys(msg)
		case ScreenEditor:
			return m.handleEditorKeys(msg)
		case ScreenStructure:
			return m.handleStructureKeys(msg)
		}

	case screens.ConnectMsg:
		m.doConnect(msg.Cfg)

	case screens.SaveConnectionMsg:
		m.saveConnection(msg.Cfg)

	case editorDoneMsg:
		if msg.token != m.queryID {
			return m, nil // resultado obsoleto: cambió la sesión
		}
		m.loading = false
		m.editor = msg.ed
		m.setErr(msg.err)
		m.editorScreen.Focus()

	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

// View implementa tea.Model.
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
	case ScreenBrowser:
		content = screens.RenderBrowser(m.browser, m.width, contentHeight, m.showSidebar)
	case ScreenEditor:
		content = m.editorScreen.View(m.editor, m.width, contentHeight)
	case ScreenStructure:
		content = screens.RenderStructure(m.browser, m.width, contentHeight)
	}

	body := content
	if m.err != "" {
		body = content + "\n" + styles.StyleError.Render(m.err)
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m *Model) renderHeader() string {
	label := "sin conexión"
	if m.store != nil {
		label = m.cfgLabel
	}
	mode := ""
	switch m.screen {
	case ScreenBrowser:
		mode = "browser"
	case ScreenEditor:
		mode = "editor"
	case ScreenStructure:
		mode = "estructura"
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
		left = "↑↓ guardadas · tab motor/campos · ←→ motor · enter conectar · ctrl+s guardar"
	case ScreenBrowser:
		left = "↑↓ navegar · tab editor · i estructura · r refrescar · ? ayuda"
	case ScreenEditor:
		left = "ctrl+r correr · tab browser · ctrl+l limpiar"
	case ScreenStructure:
		left = "esc volver · tab editor"
	}

	right := ""
	if m.loading {
		right = m.spinner.View() + " ejecutando query…"
	} else if m.browser != nil && m.screen == ScreenBrowser && m.browser.ActiveTable != "" && m.browser.TotalRows > 0 {
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

// typing indica si el usuario está escribiendo texto (q no debe salir).
func (m *Model) typing() bool {
	switch m.screen {
	case ScreenConnect:
		return m.connect.FocusOnField()
	case ScreenEditor:
		return true
	}
	return false
}

// handleConnectKeys procesa teclas de la pantalla de conexión.
func (m *Model) handleConnectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	updated, cmd := m.connect.Update(msg)
	m.connect = updated
	return m, cmd
}

// doConnect abre el store y carga el browser.
func (m *Model) doConnect(cfg conn.ConnectionConfig) {
	m.closeStore()
	m.queryID++
	m.loading = false
	m.browser = nil
	m.editor = editor.New()
	m.editorScreen.SetValue("")
	m.cfgLabel = cfg.Label()
	m.err = ""

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
	m.screen = ScreenBrowser
}

// saveConnection persiste la conexión actual.
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

// newSession cierra la sesión y vuelve a la pantalla de conexión.
func (m *Model) newSession() {
	m.closeStore()
	m.queryID++
	m.loading = false
	m.browser = nil
	m.editor = editor.New()
	m.editorScreen.SetValue("")
	m.cfgLabel = ""
	m.err = ""
	m.screen = ScreenConnect
	m.connect.ResetForm()
}

func (m *Model) closeStore() {
	if m.store != nil {
		m.store.Close()
		m.store = nil
	}
}

// handleBrowserKeys procesa teclas del browser.
func (m *Model) handleBrowserKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.browser == nil {
		return m, nil
	}
	b := m.browser

	switch {
	case key.Matches(msg, m.keys.Switch):
		m.editorScreen.Focus()
		m.screen = ScreenEditor
		return m, nil
	case key.Matches(msg, m.keys.Inspect):
		m.screen = ScreenStructure
		return m, nil
	case key.Matches(msg, m.keys.Refresh):
		m.setErr(b.Reload(m.store))
		return m, nil
	case key.Matches(msg, m.keys.ToggleSidebar):
		m.showSidebar = !m.showSidebar
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
	case msg.Type == tea.KeyRunes && len(msg.Runes) == 1 && msg.Runes[0] >= '1' && msg.Runes[0] <= '9':
		idx := int(msg.Runes[0] - '1')
		if idx < len(b.Tables) {
			m.setErr(b.SelectTable(b.Tables[idx], m.store))
		}
		return m, nil
	}
	return m, nil
}

// setErr guarda o limpia el mensaje de error actual.
func (m *Model) setErr(err error) {
	if err != nil {
		m.err = err.Error()
	} else {
		m.err = ""
	}
}

// handleStructureKeys procesa teclas de la pantalla de estructura.
func (m *Model) handleStructureKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Switch):
		m.editorScreen.Focus()
		m.screen = ScreenEditor
		return m, nil
	case key.Matches(msg, m.keys.Back):
		m.screen = ScreenBrowser
		return m, nil
	}
	return m, nil
}

// handleEditorKeys procesa teclas del editor.
func (m *Model) handleEditorKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Switch):
		m.editorScreen.Blur()
		m.screen = ScreenBrowser
		return m, nil
	case key.Matches(msg, m.keys.Back):
		m.editorScreen.Blur()
		m.screen = ScreenBrowser
		return m, nil
	case key.Matches(msg, m.keys.Execute):
		if m.loading {
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

	// cualquier otra tecla deja de navegar el historial
	m.editor.History.Reset()
	updated, cmd := m.editorScreen.Update(msg)
	m.editorScreen = updated
	m.editor.Buffer = m.editorScreen.Value()
	return m, cmd
}

// executeEditor corre el query actual en background y muestra spinner.
func (m *Model) executeEditor() tea.Cmd {
	if m.store == nil {
		return nil
	}
	buf := m.editorScreen.Value()
	line := m.editorScreen.Line() // statement bajo el cursor
	st := m.store
	token := m.queryID
	m.loading = true
	return tea.Batch(
		func() tea.Msg {
			// editor fresco para la query, pero comparte el historial
			// acumulado con el modelo (el ring buffer persiste entre ejecuciones).
			ed := editor.New()
			ed.History = m.editor.History
			ed.Buffer = buf
			err := ed.ExecuteAt(st, line)
			return editorDoneMsg{ed: ed, err: err, token: token}
		},
		m.spinner.Tick,
	)
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
		styles.StyleHeader.Render("Ayuda"),
		out,
		styles.StyleHeaderDim.Render("? para cerrar"),
	)
}

func padSpaces(n int) string {
	if n < 0 {
		n = 0
	}
	return fmt.Sprintf("%*s", n, "")
}
