package browser

import (
	"fmt"
	"testing"

	"relm/internal/conn"
	"relm/internal/store"
	_ "relm/internal/store/sqlite"
)

func newTestStore(t *testing.T) store.Store {
	t.Helper()
	cfg := conn.New(conn.DriverSQLite)
	cfg.Path = ":memory:"
	st, err := store.New(cfg)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	if _, err := st.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY,
		name TEXT,
		email TEXT
	)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := st.Exec("CREATE TABLE orders (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create: %v", err)
	}
	return st
}

func seedRows(t *testing.T, st store.Store, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		sql := fmt.Sprintf(`INSERT INTO users (name, email) VALUES ('u%d', 'u%d@t.com')`, i, i)
		if _, err := st.Exec(sql); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
}

func TestNewLoadsTablesAndSelectsFirst(t *testing.T) {
	st := newTestStore(t)
	b, err := New(st)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(b.Tables) != 2 {
		t.Fatalf("Tables = %v, want 2", b.Tables)
	}
	// Orden alfabético: orders < users
	if b.ActiveTable != "orders" {
		t.Errorf("ActiveTable = %q, want orders", b.ActiveTable)
	}
	if len(b.Columns) != 1 {
		t.Errorf("len(Columns) = %d, want 1", len(b.Columns))
	}
}

func TestBrowser_SelectTable_LoadsColumns(t *testing.T) {
	st := newTestStore(t)
	b := &Browser{PageSize: 50}
	if err := b.Load(st); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := b.SelectTable("users", st); err != nil {
		t.Fatalf("SelectTable: %v", err)
	}
	if len(b.Columns) != 3 {
		t.Errorf("len(Columns) = %d, want 3", len(b.Columns))
	}
	if b.Columns[0].PK != true {
		t.Errorf("col 0 debería ser PK: %+v", b.Columns[0])
	}
}

func TestBrowser_Pagination(t *testing.T) {
	st := newTestStore(t)
	seedRows(t, st, 120)

	b := &Browser{PageSize: 50}
	if err := b.SelectTable("users", st); err != nil {
		t.Fatalf("SelectTable: %v", err)
	}

	if b.TotalRows != 120 {
		t.Errorf("TotalRows = %d, want 120", b.TotalRows)
	}
	if len(b.Rows) != 50 {
		t.Errorf("len(Rows) página 0 = %d, want 50", len(b.Rows))
	}
	if !b.HasNextPage() || b.HasPrevPage() {
		t.Error("página 0: HasNextPage=true y HasPrevPage=false esperados")
	}

	if err := b.NextPage(st); err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(b.Rows) != 50 || b.Page != 1 {
		t.Errorf("página 1: rows=%d page=%d", len(b.Rows), b.Page)
	}

	if err := b.NextPage(st); err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(b.Rows) != 20 {
		t.Errorf("página 2: rows=%d, want 20", len(b.Rows))
	}
	if b.HasNextPage() {
		t.Error("página 2 (última) no debería tener siguiente")
	}

	if err := b.NextPage(st); err != nil {
		t.Fatalf("NextPage fuera de rango: %v", err)
	}
	if b.Page != 2 {
		t.Errorf("Page = %d, no debería avanzar más allá de 2", b.Page)
	}

	if err := b.PrevPage(st); err != nil {
		t.Fatalf("PrevPage: %v", err)
	}
	if b.Page != 1 {
		t.Errorf("Page = %d, want 1", b.Page)
	}
}

func TestBrowser_MoveCursor_Clamps(t *testing.T) {
	st := newTestStore(t)
	seedRows(t, st, 5)

	b := &Browser{PageSize: 50}
	if err := b.SelectTable("users", st); err != nil {
		t.Fatalf("SelectTable: %v", err)
	}
	b.MoveCursor(10)
	if b.Cursor != 4 {
		t.Errorf("Cursor = %d, want 4 (clamp al máximo)", b.Cursor)
	}
	b.MoveCursor(-100)
	if b.Cursor != 0 {
		t.Errorf("Cursor = %d, want 0 (clamp al mínimo)", b.Cursor)
	}
}

func TestBrowser_EmptyDatabase(t *testing.T) {
	cfg := conn.New(conn.DriverSQLite)
	cfg.Path = ":memory:"
	st, err := store.New(cfg)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer st.Close()

	b, err := New(st)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(b.Tables) != 0 {
		t.Errorf("Tables = %v, want 0", b.Tables)
	}
	if b.ActiveTable != "" {
		t.Errorf("ActiveTable = %q, want vacío", b.ActiveTable)
	}
}
