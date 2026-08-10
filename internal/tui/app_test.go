package tui

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/mattn/go-sqlite3"

	_ "relm/internal/store/mssql"
	_ "relm/internal/store/mysql" // registra los motores para los tests
	_ "relm/internal/store/postgres"
	_ "relm/internal/store/sqlite"
)

// createTestDB crea un sqlite temporal con tablas users y orders.
func createTestDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := sql.Open("sqlite3", path)
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

// press envia un KeyMsg con runas al modelo (texto escrito).
func press(t *testing.T, m *Model, text string) {
	t.Helper()
	step(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(text)})
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

// pressKey envia un KeyMsg tipado (tab, enter, ctrl+n, etc.).
func pressKey(t *testing.T, m *Model, key string) {
	t.Helper()
	kt, ok := namedKeys[key]
	if !ok {
		t.Fatalf("tecla %q no mapeada en el test", key)
	}
	step(t, m, tea.KeyMsg{Type: kt})
}

// step aplica un mensaje al modelo y ejecuta el cmd devuelto, realimentando
// los mensajes que produzca (como hace el programa de bubbletea en runtime).
func step(t *testing.T, m *Model, msg tea.Msg) {
	t.Helper()
	updated, cmd := m.Update(msg)
	m2, ok := updated.(*Model)
	if !ok {
		t.Fatalf("Update devolvió %T, quiero *Model", updated)
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
	m := New()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.screen != ScreenConnect {
		t.Fatalf("screen = %d, want ScreenConnect", m.screen)
	}
	v := m.View()
	if !strings.Contains(v, "relm") || !strings.Contains(v, "Conectar") {
		t.Errorf("View no muestra la pantalla de conexión: %q", v)
	}
}

func TestModel_ConnectToSQLiteShowsBrowser(t *testing.T) {
	db := createTestDB(t)

	m := New()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Tab: foco al campo Archivo
	pressKey(t, m, "tab")
	// escribir el path
	press(t, m, db)
	// Enter: conectar
	pressKey(t, m, "enter")

	if m.screen != ScreenBrowser {
		t.Fatalf("screen = %d, want ScreenBrowser", m.screen)
	}
	if m.browser == nil {
		t.Fatal("browser nil tras conectar")
	}
	if m.browser.ActiveTable != "orders" {
		t.Errorf("ActiveTable = %q, want orders", m.browser.ActiveTable)
	}
	v := m.View()
	if !strings.Contains(v, "users") || !strings.Contains(v, "orders") {
		t.Errorf("View no muestra el sidebar con tablas: %q", v)
	}
}

func TestModel_ConnectErrorStaysOnConnect(t *testing.T) {
	m := New()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	pressKey(t, m, "tab")
	press(t, m, "/tmp/opencode/definitely-missing.db")
	pressKey(t, m, "enter")

	if m.screen != ScreenConnect {
		t.Fatalf("screen = %d, want ScreenConnect tras error", m.screen)
	}
	if m.connect == nil || m.connect.Error() == "" {
		t.Error("esperaba error de conexión visible")
	}
}

func TestModel_TabSwitchesToEditor(t *testing.T) {
	db := createTestDB(t)

	m := New()
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

	m := New()
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

	m := New()
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
		t.Fatal("query no terminó: loading quedó true")
	}
	if m.editor == nil || m.editor.Result == nil {
		t.Fatalf("editor sin resultado tras ejecutar")
	}
	if len(m.editor.Result.Rows) != 2 {
		t.Errorf("Rows = %d, want 2", len(m.editor.Result.Rows))
	}
	v := m.View()
	if !strings.Contains(v, "Alice") {
		t.Errorf("View no muestra la fila Alice: %q", v)
	}
}

func TestModel_EditorShowsError(t *testing.T) {
	db := createTestDB(t)

	m := New()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	pressKey(t, m, "tab")
	press(t, m, db)
	pressKey(t, m, "enter")
	pressKey(t, m, "tab")

	press(t, m, "SELEC broken")
	pressKey(t, m, "ctrl+r")

	if m.editor == nil || m.editor.Error == "" {
		t.Fatalf("esperaba error SQL en el editor")
	}
	if m.editor.Result != nil {
		t.Error("no debería haber resultado con error")
	}
}

func TestModel_EditorHistoryNavigation(t *testing.T) {
	db := createTestDB(t)

	m := New()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	pressKey(t, m, "tab")
	press(t, m, db)
	pressKey(t, m, "enter")
	pressKey(t, m, "tab")

	press(t, m, "SELECT 1")
	pressKey(t, m, "ctrl+r")
	pressKey(t, m, "ctrl+l") // limpiar antes del siguiente query
	press(t, m, "SELECT 2")
	pressKey(t, m, "ctrl+r")
	pressKey(t, m, "ctrl+l") // limpiar para navegar el historial

	pressKey(t, m, "up")
	if m.editor.Buffer != "SELECT 2" {
		t.Errorf("Buffer = %q, want SELECT 2 (query más reciente)", m.editor.Buffer)
	}
	pressKey(t, m, "up")
	if m.editor.Buffer != "SELECT 1" {
		t.Errorf("Buffer = %q, want SELECT 1", m.editor.Buffer)
	}
}

// TestModel_ConnectToPostgres valida el stack completo (store → browser → TUI)
// contra PostgreSQL real. Se salta sin env var.
func TestModel_ConnectToPostgres(t *testing.T) {
	host := os.Getenv("SQLISH_TEST_POSTGRES_HOST")
	if host == "" {
		t.Skip("SQLISH_TEST_POSTGRES_HOST no seteada")
	}

	db := t.TempDir() + "/none.db" // placeholder no usado
	_ = db

	m := New()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	// Conectar por el formulario completo: motor postgres, host, user, pass, base
	pressKey(t, m, "right") // sqlite -> postgres
	pressKey(t, m, "tab")   // focus: Archivo (oculto) -> Host
	press(t, m, host)
	pressKey(t, m, "tab") // Puerto
	pressKey(t, m, "tab") // Usuario
	press(t, m, os.Getenv("SQLISH_TEST_POSTGRES_USER"))
	pressKey(t, m, "tab") // Password
	press(t, m, os.Getenv("SQLISH_TEST_POSTGRES_PASSWORD"))
	pressKey(t, m, "tab") // Base
	press(t, m, os.Getenv("SQLISH_TEST_POSTGRES_DATABASE"))
	pressKey(t, m, "enter")

	if m.screen != ScreenBrowser {
		t.Fatalf("screen = %d, want ScreenBrowser. connectErr=%q", m.screen, m.connect.Error())
	}
	if m.browser == nil || len(m.browser.Tables) == 0 {
		t.Fatalf("browser sin tablas")
	}
	_ = m.View()
}

