package browser

import (
	"fmt"
	"testing"

	"github.com/agmonetti/relm/internal/conn"
	"github.com/agmonetti/relm/internal/store"
	_ "github.com/agmonetti/relm/internal/store/sqlite"
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
	// Alphabetical order: orders < users
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
		t.Errorf("col 0 should be PK: %+v", b.Columns[0])
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
		t.Errorf("len(Rows) page 0 = %d, want 50", len(b.Rows))
	}
	if !b.HasNextPage() || b.HasPrevPage() {
		t.Error("page 0: HasNextPage=true and HasPrevPage=false expected")
	}

	if err := b.NextPage(st); err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(b.Rows) != 50 || b.Page != 1 {
		t.Errorf("page 1: rows=%d page=%d", len(b.Rows), b.Page)
	}

	if err := b.NextPage(st); err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if len(b.Rows) != 20 {
		t.Errorf("page 2: rows=%d, want 20", len(b.Rows))
	}
	if b.HasNextPage() {
		t.Error("page 2 (last) should not have a next page")
	}

	if err := b.NextPage(st); err != nil {
		t.Fatalf("NextPage out of range: %v", err)
	}
	if b.Page != 2 {
		t.Errorf("Page = %d, should not advance beyond 2", b.Page)
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
		t.Errorf("Cursor = %d, want 4 (clamp to max)", b.Cursor)
	}
	b.MoveCursor(-100)
	if b.Cursor != 0 {
		t.Errorf("Cursor = %d, want 0 (clamp to min)", b.Cursor)
	}
}

func TestBrowser_KeysetRefreshIsStable(t *testing.T) {
	st := newTestStore(t)
	seedRows(t, st, 60) // users id 1..60

	b := &Browser{PageSize: 50}
	if err := b.SelectTable("users", st); err != nil {
		t.Fatalf("SelectTable: %v", err)
	}
	if b.orderBy != "id" || b.keyIdx != 0 {
		t.Fatalf("orderBy = %q keyIdx=%d, want keyset on id", b.orderBy, b.keyIdx)
	}
	if len(b.Rows) != 50 || b.Rows[0][1] != "u0" {
		t.Fatalf("page 0 = %d rows, first %v", len(b.Rows), b.Rows[0])
	}

	// a row inserted before the last row of the page must not shift the page
	if _, err := st.Exec("INSERT INTO users (name, email) VALUES ('u0b','u0b@t.com')"); err != nil {
		t.Fatalf("insert: %v", err)
	}
	if err := b.Refresh(st); err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if b.Rows[0][1] != "u0" {
		t.Errorf("first row after refresh = %v, want u0 (stable)", b.Rows[0])
	}

	// next page and back
	if !b.HasNextPage() {
		t.Fatal("page 0 should have a next page")
	}
	page0Last := b.Rows[len(b.Rows)-1][0]
	if err := b.NextPage(st); err != nil {
		t.Fatalf("NextPage: %v", err)
	}
	if b.Page != 1 || len(b.Rows) == 0 {
		t.Fatalf("page 1 = %d rows, want > 0", len(b.Rows))
	}
	if err := b.PrevPage(st); err != nil {
		t.Fatalf("PrevPage: %v", err)
	}
	if b.Page != 0 || len(b.Rows) != 50 {
		t.Errorf("back to page 0 = %d rows, want 50", len(b.Rows))
	}
	if b.Rows[len(b.Rows)-1][0] != page0Last {
		t.Errorf("last row of page 0 = %s, want %s (same page back)", b.Rows[len(b.Rows)-1][0], page0Last)
	}
}

func TestBrowser_KeysetFallbackWithoutPK(t *testing.T) {
	st := newTestStore(t)
	if _, err := st.Exec("CREATE TABLE nopk (name TEXT, email TEXT)"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := st.Exec("INSERT INTO nopk (name, email) VALUES ('a','a@t.com'), ('b','b@t.com')"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	b := &Browser{PageSize: 50}
	if err := b.SelectTable("nopk", st); err != nil {
		t.Fatalf("SelectTable: %v", err)
	}
	if b.orderBy != "" {
		t.Errorf("orderBy = %q, want empty (fallback)", b.orderBy)
	}
	if len(b.Rows) != 2 {
		t.Errorf("Rows = %d, want 2", len(b.Rows))
	}
}

func TestBrowser_ReloadAfterDropRecreateWithoutPK(t *testing.T) {
	st := newTestStore(t)
	b, err := New(st) // selects "orders" (has id PK)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if b.ActiveTable != "orders" || b.orderBy == "" {
		t.Fatalf("setup: active=%q orderBy=%q, want orders on keyset", b.ActiveTable, b.orderBy)
	}

	// simulate the editor dropping and re-creating the table without a PK
	if _, err := st.Exec("DROP TABLE orders"); err != nil {
		t.Fatalf("drop: %v", err)
	}
	if _, err := st.Exec("CREATE TABLE orders (ref TEXT)"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := st.Exec("INSERT INTO orders VALUES ('x')"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := b.Reload(st); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if b.orderBy != "" {
		t.Errorf("orderBy = %q, want empty (fallback after recreate)", b.orderBy)
	}
	if len(b.Rows) != 1 {
		t.Errorf("Rows = %v, want the re-created row", b.Rows)
	}
}

func TestBrowser_KeysetFallbackWithCompositePK(t *testing.T) {
	st := newTestStore(t)
	if _, err := st.Exec("CREATE TABLE comp (a INT, b INT, PRIMARY KEY (a, b))"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := st.Exec("INSERT INTO comp VALUES (1, 1), (1, 2)"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	b := &Browser{PageSize: 50}
	if err := b.SelectTable("comp", st); err != nil {
		t.Fatalf("SelectTable: %v", err)
	}
	if b.orderBy != "" {
		t.Errorf("orderBy = %q, want empty (fallback for composite PK)", b.orderBy)
	}
	if len(b.Rows) != 2 {
		t.Errorf("Rows = %d, want 2", len(b.Rows))
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
		t.Errorf("ActiveTable = %q, want empty", b.ActiveTable)
	}
}

func TestBrowser_ReloadPicksUpNewTableAndRows(t *testing.T) {
	st := newTestStore(t)
	b, err := New(st)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if b.ActiveTable != "orders" {
		t.Fatalf("setup: ActiveTable = %q", b.ActiveTable)
	}

	// create a new table and add a row to the active table (orders),
	// as if it came from the editor
	if _, err := st.Exec("CREATE TABLE products (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := st.Exec("INSERT INTO orders (id) VALUES (1)"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	if err := b.Reload(st); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if len(b.Tables) != 3 {
		t.Errorf("Tables = %v, want 3 (orders, products, users)", b.Tables)
	}
	if b.ActiveTable != "orders" {
		t.Errorf("ActiveTable = %q, want orders (sigue activa)", b.ActiveTable)
	}
	if b.TotalRows != 1 {
		t.Errorf("TotalRows = %d, want 1 (new row in orders)", b.TotalRows)
	}

	// Reload with an empty active table: selects the first one
	cfg := conn.New(conn.DriverSQLite)
	cfg.Path = ":memory:"
	st2, err := store.New(cfg)
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	defer st2.Close()
	b2, err := New(st2) // empty database → ActiveTable ""
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := st2.Exec("CREATE TABLE recien (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := b2.Reload(st2); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if b2.ActiveTable != "recien" {
		t.Errorf("ActiveTable = %q, want recien (selecciona la primera)", b2.ActiveTable)
	}
}
