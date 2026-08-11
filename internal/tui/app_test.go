package tui

import (
	"database/sql"
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

func TestModel_ConnectToSQLiteShowsBrowser(t *testing.T) {
	db := createTestDB(t)

	m := newModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Tab: focus the File field
	pressKey(t, m, "tab")
	// escribir el path
	press(t, m, db)
	// Enter: connect
	pressKey(t, m, "enter")

	if m.screen != ScreenBrowser {
		t.Fatalf("screen = %d, want ScreenBrowser", m.screen)
	}
	if m.browser == nil {
		t.Fatal("browser nil after connecting")
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
	press(t, m, "/tmp/opencode/definitely-missing.db")
	pressKey(t, m, "enter")

	if m.screen != ScreenConnect {
		t.Fatalf("screen = %d, want ScreenConnect after error", m.screen)
	}
	if m.connect == nil || m.connect.Error() == "" {
		t.Error("expected a visible connection error")
	}
}

func TestModel_TabSwitchesToEditor(t *testing.T) {
	db := createTestDB(t)

	m := newModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	pressKey(t, m, "tab")
	press(t, m, db)
	pressKey(t, m, "enter")

	pressKey(t, m, "tab")
	if m.screen != ScreenEditor {
		t.Fatalf("screen = %d, want ScreenEditor", m.screen)
	}
}

func TestModel_NewSessionReturnsToConnect(t *testing.T) {
	db := createTestDB(t)

	m := newModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	pressKey(t, m, "tab")
	press(t, m, db)
	pressKey(t, m, "enter")
	if m.screen != ScreenBrowser {
		t.Fatalf("setup: screen = %d", m.screen)
	}

	pressKey(t, m, "ctrl+n")
	if m.screen != ScreenConnect {
		t.Fatalf("screen = %d, want ScreenConnect", m.screen)
	}
}

func TestModel_EditorExecutesQuery(t *testing.T) {
	db := createTestDB(t)

	m := newModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	pressKey(t, m, "tab")
	press(t, m, db)
	pressKey(t, m, "enter")

	pressKey(t, m, "tab") // al editor
	if m.screen != ScreenEditor {
		t.Fatalf("setup: screen = %d, want ScreenEditor", m.screen)
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
	db := createTestDB(t)

	m := newModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	pressKey(t, m, "tab")
	press(t, m, db)
	pressKey(t, m, "enter")
	pressKey(t, m, "tab") // to the editor

	pressKey(t, m, "ctrl+r") // empty buffer

	if m.loading {
		t.Fatal("an empty buffer must not run a query")
	}
	if m.editor == nil || m.editor.Error != "write a query first" {
		t.Fatalf("expected 'write a query first', got %+v", m.editor)
	}
}

func TestModel_EditorShowsError(t *testing.T) {
	db := createTestDB(t)

	m := newModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	pressKey(t, m, "tab")
	press(t, m, db)
	pressKey(t, m, "enter")
	pressKey(t, m, "tab")

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
	db := createTestDB(t)

	m := newModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	pressKey(t, m, "tab")
	press(t, m, db)
	pressKey(t, m, "enter")
	pressKey(t, m, "tab")

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

// TestModel_ConnectToPostgres valida el stack completo (store → browser → TUI)
// contra PostgreSQL real. Se salta sin env var.
func TestModel_RefreshShowsInsertedRow(t *testing.T) {
	db := createTestDB(t)

	m := newModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	pressKey(t, m, "tab")
	press(t, m, db)
	pressKey(t, m, "enter")
	if m.screen != ScreenBrowser {
		t.Fatalf("setup: screen = %d", m.screen)
	}

	// select users (second alphabetically, key "2")
	press(t, m, "2")
	if m.browser.ActiveTable != "users" {
		t.Fatalf("setup: ActiveTable = %q, want users", m.browser.ActiveTable)
	}

	// insert a row from the editor
	pressKey(t, m, "tab")
	press(t, m, "INSERT INTO users (name, email) VALUES ('Carol','c@t.com')")
	pressKey(t, m, "ctrl+r")
	if m.editor == nil || m.editor.Result == nil || m.editor.Result.Affected != 1 {
		t.Fatalf("insert did not run: %+v", m.editor)
	}

	// go back to the browser and refresh with "r"
	pressKey(t, m, "tab")
	press(t, m, "r")

	if m.browser.TotalRows != 3 {
		t.Errorf("TotalRows = %d, want 3 tras refresh", m.browser.TotalRows)
	}
	v := m.View()
	if !strings.Contains(v, "Carol") {
		t.Errorf("View does not show the Carol row after refresh: %q", v)
	}
}

func TestModel_ConnectToPostgres(t *testing.T) {
	host := os.Getenv("SQLISH_TEST_POSTGRES_HOST")
	if host == "" {
		t.Skip("SQLISH_TEST_POSTGRES_HOST not set")
	}

	db := t.TempDir() + "/none.db" // placeholder no usado
	_ = db

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

	if m.screen != ScreenBrowser {
		t.Fatalf("screen = %d, want ScreenBrowser. connectErr=%q", m.screen, m.connect.Error())
	}
	if m.browser == nil || len(m.browser.Tables) == 0 {
		t.Fatalf("browser without tables")
	}
	_ = m.View()
}

