// Package tui implements the bubbletea loop for relm.
package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/conn"
	"github.com/agmonetti/relm/internal/editor"
	"github.com/agmonetti/relm/internal/prefs"
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
	// ScreenSettings is the preferences screen.
	ScreenSettings
)

// Divider targeted by a right-click drag.
const (
	resizeNone int = iota
	resizeSidebar
	resizeEditor
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
	queryID int                // token to discard results from stale queries
	cancel  context.CancelFunc // cancels the running query

	// browser navigation runs in the background (like the editor) so a slow
	// table cannot freeze the UI. Only one navigation and one connection load
	// may be in flight; both are bounded by the configured query timeout and
	// cancelled with Esc.
	navigating    bool
	navCancel     context.CancelFunc
	navID         int // incremented per navigation; stale results are dropped
	connecting    bool
	connectCancel context.CancelFunc

	// workspace pane sizes (0 = auto), resizable with a right-click drag
	sidebarW  int
	editorH   int
	resizing  bool
	resizeDiv int // resizeNone, resizeSidebar or resizeEditor

	width       int
	height      int
	showSidebar bool
	showHelp    bool
	err         string

	// detail view ("v"): a snapshot of the selected row with full values
	showDetail   bool
	detailScroll int
	detailTitle  string
	detailCols   []string
	detailVals   []string

	// export prompt ("alt+e"): a centered input for the target filename
	exporting   bool
	exportInput textinput.Model
	exportErr   string
	exportRes   *store.Result
	exportNote  string // "N rows" description appended to the success message
	exported    string // last success message, rendered in green
}

// New creates the initial model (connection screen).
func New() *Model {
	saved, _ := conn.LoadSaved()
	p, _ := prefs.Load()
	exportInput := textinput.New()
	exportInput.Cursor.BlinkSpeed = screens.CursorBlink
	exportInput.Prompt = " "
	exportInput.Width = 40
	return &Model{
		connect:      screens.NewConnScreen(saved),
		settings:     screens.NewSettingsScreen(),
		editor:       editor.New(),
		editorScreen: screens.NewEditorScreen(),
		exportInput:  exportInput,
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
