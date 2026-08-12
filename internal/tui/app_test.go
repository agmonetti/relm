package tui

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	_ "modernc.org/sqlite"

	_ "github.com/agmonetti/relm/internal/store/mssql"
	_ "github.com/agmonetti/relm/internal/store/mysql" // registers the engines for the tests
	_ "github.com/agmonetti/relm/internal/store/postgres"
	_ "github.com/agmonetti/relm/internal/store/sqlite"
	"github.com/agmonetti/relm/internal/conn"
	"github.com/agmonetti/relm/internal/prefs"
	"github.com/agmonetti/relm/internal/tui/screens"
)

// createTestDB creates a temporary sqlite with users and orders tables.
func createTestDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	stmts := []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)",
		"CREATE TABLE orders (id INTEGER PRIMARY KEY)",
		"INSERT INTO users (name, email) VALUES ('Alice','a@t.com'), ('Bob','b@t.com')",
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("exec %q: %v", s, err)
		}
	}
	return path
}

// press sends a KeyMsg with runes to the model (typed text).
func press(t *testing.T, m *Model, text string) {
	t.Helper()
	step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)})
}

// newModel creates a Model and closes its store at the end of the test, so the
// SQLite file is not left locked (Windows fails to delete a TempDir in use).
func newModel(t *testing.T) *Model {
	t.Helper()
	m := New()
	t.Cleanup(func() {
		if m.store != nil {
			m.store.Close()
		}
	})
	return m
}

// connect creates a model connected to a fresh SQLite test database.
func connect(t *testing.T) *Model {
	t.Helper()
	db := createTestDB(t)
	m := newModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	pressKey(t, m, "tab") // focus the File field
	press(t, m, db)
	pressKey(t, m, "enter") // connect
	if m.screen != ScreenWorkspace {
		t.Fatalf("screen = %d, want ScreenWorkspace", m.screen)
	}
	return m
}

var namedKeys = map[string]tea.KeyType{
	"tab":    tea.KeyTab,
	"enter":  tea.KeyEnter,
	"esc":    tea.KeyEsc,
	"up":     tea.KeyUp,
	"down":   tea.KeyDown,
	"left":   tea.KeyLeft,
	"right":  tea.KeyRight,
	"home":   tea.KeyHome,
	"end":    tea.KeyEnd,
	"pgup":   tea.KeyPgUp,
	"pgdown": tea.KeyPgDown,
	"ctrl+c": tea.KeyCtrlC,
	"ctrl+i": tea.KeyCtrlI,
	"ctrl+l": tea.KeyCtrlL,
	"ctrl+n": tea.KeyCtrlN,
	"ctrl+p": tea.KeyCtrlP,
	"ctrl+r": tea.KeyCtrlR,
	"ctrl+s": tea.KeyCtrlS,
}

// pressKey sends a typed KeyMsg (tab, enter, ctrl+n, etc.).
func pressKey(t *testing.T, m *Model, key string) {
	t.Helper()
	kt, ok := namedKeys[key]
	if !ok {
		t.Fatalf("key %q not mapped in the test", key)
	}
	step(t, m, tea.KeyMsg{Type: kt})
}

// pressAlt sends an alt+digit key (used to jump between workspace panes).
func pressAlt(t *testing.T, m *Model, digit string) {
	t.Helper()
	step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(digit), Alt: true})
}

// step applies a message to the model and runs the returned cmd, feeding back
// the messages it produces (like the bubbletea program does at runtime).
func step(t *testing.T, m *Model, msg tea.Msg) {
	t.Helper()
	updated, cmd := m.Update(msg)
	m2, ok := updated.(*Model)
	if !ok {
		t.Fatalf("Update returned %T, want *Model", updated)
	}
	*m = *m2
	if cmd != nil {
		if out := cmd(); out != nil {
			if batch, ok := out.(tea.BatchMsg); ok {
				for _, sub := range batch {
					if sub == nil {
						continue
					}
					if subMsg := sub(); subMsg != nil {
						step(t, m, subMsg)
					}
				}
			} else {
				step(t, m, out)
			}
		}
	}
}

func TestModel_StartsOnConnect(t *testing.T) {
	m := newModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.screen != ScreenConnect {
		t.Fatalf("screen = %d, want ScreenConnect", m.screen)
	}
	v := m.View()
	if !strings.Contains(v, "relm") || !strings.Contains(v, "Connect") {
		t.Errorf("View does not show the connection screen: %q", v)
	}
}

func TestModel_ConnectShowsWorkspace(t *testing.T) {
	m := connect(t)
	if m.screen != ScreenWorkspace {
		t.Fatalf("screen = %d, want ScreenWorkspace", m.screen)
	}
	if m.browser == nil {
		t.Fatal("browser nil after connecting")
	}
	if m.focus != screens.FocusSidebar {
		t.Errorf("focus = %v, want sidebar", m.focus)
	}
	if m.browser.ActiveTable != "orders" {
		t.Errorf("ActiveTable = %q, want orders", m.browser.ActiveTable)
	}
	v := m.View()
	if !strings.Contains(v, "users") || !strings.Contains(v, "orders") {
		t.Errorf("View does not show the sidebar with tables: %q", v)
	}
}

func TestModel_ConnectErrorStaysOnConnect(t *testing.T) {
	m := newModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	pressKey(t, m, "tab")
	press(t, m, filepath.Join(t.TempDir(), "definitely-missing.db"))
	pressKey(t, m, "enter")

	if m.screen != ScreenConnect {
		t.Fatalf("screen = %d, want ScreenConnect after error", m.screen)
	}
	if m.connect == nil || m.connect.Error() == "" {
		t.Error("expected a visible connection error")
	}
}

func TestModel_TabCyclesFocus(t *testing.T) {
	m := connect(t)
	if m.focus != screens.FocusSidebar {
		t.Fatalf("setup: focus = %v, want sidebar", m.focus)
	}
	pressKey(t, m, "tab")
	if m.focus != screens.FocusMain {
		t.Errorf("focus = %v, want main", m.focus)
	}
	pressKey(t, m, "tab")
	if m.focus != screens.FocusEditor {
		t.Errorf("focus = %v, want editor", m.focus)
	}
	pressKey(t, m, "tab")
	if m.focus != screens.FocusSidebar {
		t.Errorf("focus = %v, want sidebar (cycle)", m.focus)
	}
}

func TestModel_AltDigitsJumpFocus(t *testing.T) {
	m := connect(t)
	pressAlt(t, m, "3")
	if m.focus != screens.FocusEditor {
		t.Errorf("alt+3: focus = %v, want editor", m.focus)
	}
	pressAlt(t, m, "1")
	if m.focus != screens.FocusSidebar {
		t.Errorf("alt+1: focus = %v, want sidebar", m.focus)
	}
	pressAlt(t, m, "2")
	if m.focus != screens.FocusMain {
		t.Errorf("alt+2: focus = %v, want main", m.focus)
	}
}

func TestModel_NewSessionReturnsToConnect(t *testing.T) {
	m := connect(t)
	pressKey(t, m, "ctrl+n")
	if m.screen != ScreenConnect {
		t.Fatalf("screen = %d, want ScreenConnect", m.screen)
	}
}

func TestModel_SidebarEnterOpensTable(t *testing.T) {
	m := connect(t)
	if m.browser.ActiveTable != "orders" {
		t.Fatalf("setup: ActiveTable = %q, want orders", m.browser.ActiveTable)
	}
	pressKey(t, m, "down") // sidebar cursor -> users
	if m.sidebarCursor != 1 {
		t.Fatalf("sidebarCursor = %d, want 1", m.sidebarCursor)
	}
	pressKey(t, m, "enter")
	if m.browser.ActiveTable != "users" {
		t.Errorf("ActiveTable = %q, want users", m.browser.ActiveTable)
	}
}

func TestModel_SidebarFirstLast(t *testing.T) {
	m := connect(t)
	// tables: orders, users (alphabetical); cursor starts at 0
	pressKey(t, m, "end") // last table
	if m.sidebarCursor != 1 {
		t.Fatalf("end: sidebarCursor = %d, want 1", m.sidebarCursor)
	}
	press(t, m, "g") // first table
	if m.sidebarCursor != 0 {
		t.Errorf("g: sidebarCursor = %d, want 0", m.sidebarCursor)
	}
}

func TestModel_OpenSettingsAndSave(t *testing.T) {
	t.Setenv("RELM_CONFIG_DIR", t.TempDir())
	m := connect(t)

	pressKey(t, m, "ctrl+p")
	if m.screen != ScreenSettings {
		t.Fatalf("screen = %d, want ScreenSettings", m.screen)
	}
	if v := m.View(); !strings.Contains(v, "Settings") {
		t.Errorf("settings screen does not render: %q", v)
	}
	// prefill the value as if the user typed it, then save
	m.settings.SetValue("90")
	pressKey(t, m, "enter")

	if m.screen != ScreenWorkspace {
		t.Fatalf("after save screen = %d, want back to workspace", m.screen)
	}
	if m.prefs.QueryTimeoutSeconds != 90 {
		t.Errorf("prefs.QueryTimeoutSeconds = %d, want 90", m.prefs.QueryTimeoutSeconds)
	}
	p, err := prefs.Load()
	if err != nil {
		t.Fatalf("prefs.Load: %v", err)
	}
	if p.QueryTimeoutSeconds != 90 {
		t.Errorf("persisted QueryTimeoutSeconds = %d, want 90", p.QueryTimeoutSeconds)
	}
}

func TestModel_SettingsEscBack(t *testing.T) {
	m := connect(t)
	pressKey(t, m, "ctrl+p")
	if m.screen != ScreenSettings {
		t.Fatalf("screen = %d, want ScreenSettings", m.screen)
	}
	pressKey(t, m, "esc")
	if m.screen != ScreenWorkspace {
		t.Errorf("after esc screen = %d, want workspace", m.screen)
	}
}

func TestModel_SettingsKeepsContext(t *testing.T) {
	m := newModel(t) // starts on the connect screen
	pressKey(t, m, "ctrl+p")
	if m.screen != ScreenSettings {
		t.Fatalf("screen = %d, want ScreenSettings", m.screen)
	}
	pressKey(t, m, "esc")
	if m.screen != ScreenConnect {
		t.Errorf("after esc screen = %d, want connect", m.screen)
	}
}

func TestFriendlyErr(t *testing.T) {
	if got := friendlyErr(context.DeadlineExceeded).Error(); got != "query timed out" {
		t.Errorf("DeadlineExceeded -> %q", got)
	}
	if got := friendlyErr(context.Canceled).Error(); got != "query cancelled" {
		t.Errorf("Canceled -> %q", got)
	}
	if got := friendlyErr(errors.New("boom")); got.Error() != "boom" {
		t.Errorf("other error -> %q", got)
	}
}

func TestModel_StructureMode(t *testing.T) {
	m := connect(t)
	pressAlt(t, m, "2") // main
	press(t, m, "i")
	if !m.structure {
		t.Fatal("structure should be on after i")
	}
	v := m.View()
	if !strings.Contains(v, "Columns") {
		t.Errorf("structure not shown in the main pane: %q", v)
	}
	pressKey(t, m, "esc")
	if m.structure {
		t.Fatal("structure should be off after esc")
	}
}

func TestModel_EditorExecutesQuery(t *testing.T) {
	m := connect(t)
	pressAlt(t, m, "3") // editor
	if m.focus != screens.FocusEditor {
		t.Fatalf("setup: focus = %v, want editor", m.focus)
	}

	press(t, m, "SELECT * FROM users")
	pressKey(t, m, "ctrl+r")

	if m.loading {
		t.Fatal("query did not finish: loading stayed true")
	}
	if m.editor == nil || m.editor.Result == nil {
		t.Fatalf("editor without result after running")
	}
	if len(m.editor.Result.Rows) != 2 {
		t.Errorf("Rows = %d, want 2", len(m.editor.Result.Rows))
	}
	v := m.View()
	if !strings.Contains(v, "Alice") {
		t.Errorf("View does not show the Alice row: %q", v)
	}
}

func TestModel_EditorEmptyBufferShowsNotice(t *testing.T) {
	m := connect(t)
	pressAlt(t, m, "3")
	pressKey(t, m, "ctrl+r") // empty buffer

	if m.loading {
		t.Fatal("an empty buffer must not run a query")
	}
	if m.editor == nil || m.editor.Error != "write a query first" {
		t.Fatalf("expected 'write a query first', got %+v", m.editor)
	}
}

func TestModel_EditorShowsError(t *testing.T) {
	m := connect(t)
	pressAlt(t, m, "3")
	press(t, m, "SELEC broken")
	pressKey(t, m, "ctrl+r")

	if m.editor == nil || m.editor.Error == "" {
		t.Fatalf("expected an SQL error in the editor")
	}
	if m.editor.Result != nil {
		t.Error("there should be no result with an error")
	}
}

func TestModel_EditorHistoryNavigation(t *testing.T) {
	m := connect(t)
	pressAlt(t, m, "3")

	press(t, m, "SELECT 1")
	pressKey(t, m, "ctrl+r")
	pressKey(t, m, "ctrl+l") // clear before the next query
	press(t, m, "SELECT 2")
	pressKey(t, m, "ctrl+r")
	pressKey(t, m, "ctrl+l") // clear to navigate the history

	pressKey(t, m, "up")
	if m.editor.Buffer != "SELECT 2" {
		t.Errorf("Buffer = %q, want SELECT 2 (most recent query)", m.editor.Buffer)
	}
	pressKey(t, m, "up")
	if m.editor.Buffer != "SELECT 1" {
		t.Errorf("Buffer = %q, want SELECT 1", m.editor.Buffer)
	}
}

func TestModel_AutoRefreshAfterInsert(t *testing.T) {
	m := connect(t)

	// sidebar focus default: quick-select users (second alphabetically)
	press(t, m, "2")
	if m.browser.ActiveTable != "users" {
		t.Fatalf("setup: ActiveTable = %q, want users", m.browser.ActiveTable)
	}

	pressAlt(t, m, "3") // editor
	press(t, m, "INSERT INTO users (name, email) VALUES ('Carol','c@t.com')")
	pressKey(t, m, "ctrl+r")
	if m.editor == nil || m.editor.Result == nil || m.editor.Result.Affected != 1 {
		t.Fatalf("insert did not run: %+v", m.editor)
	}

	// a write query auto-refreshes the open table without pressing r
	if m.browser.TotalRows != 3 {
		t.Errorf("TotalRows = %d, want 3 after auto-refresh", m.browser.TotalRows)
	}
	v := m.View()
	if !strings.Contains(v, "Carol") {
		t.Errorf("View does not show the Carol row: %q", v)
	}
}

func TestModel_ToggleSidebar(t *testing.T) {
	m := connect(t)
	press(t, m, "2") // open users (has rows)
	if m.browser.ActiveTable != "users" {
		t.Fatalf("setup: ActiveTable = %q, want users", m.browser.ActiveTable)
	}
	if !m.showSidebar {
		t.Fatal("setup: showSidebar should be true")
	}

	pressAlt(t, m, "b")
	if m.showSidebar {
		t.Error("showSidebar should be false after alt+b")
	}
	v := m.View()
	if !strings.Contains(v, "Alice") {
		t.Errorf("main pane should still render with the sidebar hidden: %q", v)
	}

	pressAlt(t, m, "b")
	if !m.showSidebar {
		t.Error("showSidebar should be true after the second alt+b")
	}
}

func TestModel_SaveAndDeleteSavedConnection(t *testing.T) {
	t.Setenv("RELM_CONFIG_DIR", t.TempDir())

	m := newModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	pressKey(t, m, "tab") // File field
	press(t, m, "/data/app.db")
	pressKey(t, m, "ctrl+s") // save

	saved, err := conn.LoadSaved()
	if err != nil {
		t.Fatalf("LoadSaved: %v", err)
	}
	if len(saved) != 1 || saved[0].Path != "/data/app.db" {
		t.Fatalf("saved = %+v, want one /data/app.db", saved)
	}

	// sqlite: File(1), Read-only(2) -> saved list(3); "d" deletes it
	pressKey(t, m, "tab")
	pressKey(t, m, "tab")
	press(t, m, "d")

	saved, err = conn.LoadSaved()
	if err != nil {
		t.Fatalf("LoadSaved after delete: %v", err)
	}
	if len(saved) != 0 {
		t.Errorf("saved = %+v, want 0 after delete", saved)
	}
}

func TestCorrectSize_GuardsAgainstHugeReportedSizes(t *testing.T) {
	old := termGetSize
	termGetSize = func(uintptr) (int, int, error) { return 120, 30, nil }
	t.Cleanup(func() { termGetSize = old })

	cases := []struct{ inW, inH, wantW, wantH int }{
		{120, 30, 120, 30},   // normal size passes through
		{120, 9001, 120, 30}, // conhost buffer height -> real viewport
		{9001, 120, 120, 30}, // conhost buffer width -> real viewport
		{0, 0, 120, 30},      // bogus -> re-query
	}
	for _, tc := range cases {
		w, h := correctSize(tc.inW, tc.inH)
		if w != tc.wantW || h != tc.wantH {
			t.Errorf("correctSize(%d,%d) = (%d,%d), want (%d,%d)",
				tc.inW, tc.inH, w, h, tc.wantW, tc.wantH)
		}
	}
}

func TestCorrectSize_ClampsWhenRequeryFails(t *testing.T) {
	old := termGetSize
	termGetSize = func(uintptr) (int, int, error) { return 0, 0, errors.New("no tty") }
	t.Cleanup(func() { termGetSize = old })

	w, h := correctSize(0, 0)
	if w < 10 || h < 3 {
		t.Errorf("correctSize(0,0) = (%d,%d), want sane minimums", w, h)
	}
}

func TestModel_ConnectToPostgres(t *testing.T) {
	host := os.Getenv("SQLISH_TEST_POSTGRES_HOST")
	if host == "" {
		t.Skip("SQLISH_TEST_POSTGRES_HOST not set")
	}

	m := newModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Connect through the full form: postgres engine, host, user, pass, database
	pressKey(t, m, "right") // sqlite -> postgres
	pressKey(t, m, "tab")   // focus: File (hidden) -> Host
	press(t, m, host)
	pressKey(t, m, "tab") // Port
	pressKey(t, m, "tab") // User
	press(t, m, os.Getenv("SQLISH_TEST_POSTGRES_USER"))
	pressKey(t, m, "tab") // Password
	press(t, m, os.Getenv("SQLISH_TEST_POSTGRES_PASSWORD"))
	pressKey(t, m, "tab") // Database
	press(t, m, os.Getenv("SQLISH_TEST_POSTGRES_DATABASE"))
	pressKey(t, m, "enter")

	if m.screen != ScreenWorkspace {
		t.Fatalf("screen = %d, want ScreenWorkspace. connectErr=%q", m.screen, m.connect.Error())
	}
	if m.browser == nil || len(m.browser.Tables) == 0 {
		t.Fatalf("browser without tables")
	}
	_ = m.View()
}
