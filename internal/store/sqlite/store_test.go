package sqlite

import (
	"path/filepath"
	"testing"

	"relm/internal/conn"
)

func TestOpenMemoryAndIntrospect(t *testing.T) {
	cfg := conn.New(conn.DriverSQLite)
	cfg.Path = ":memory:"

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()

	if _, err := s.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT NOT NULL, email TEXT)"); err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if _, err := s.Exec("INSERT INTO users (name) VALUES ('Alice'), ('Bob')"); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	tables, err := s.Tables()
	if err != nil {
		t.Fatalf("Tables: %v", err)
	}
	if len(tables) != 1 || tables[0] != "users" {
		t.Fatalf("Tables = %v, want [users]", tables)
	}

	cols, err := s.Columns("users")
	if err != nil {
		t.Fatalf("Columns: %v", err)
	}
	if len(cols) != 3 {
		t.Fatalf("len(Columns) = %d, want 3", len(cols))
	}
	if !cols[0].PK || !cols[1].NotNull || cols[2].NotNull {
		t.Errorf("constraints mal: %+v", cols)
	}

	n, err := s.CountTable("users")
	if err != nil {
		t.Fatalf("CountTable: %v", err)
	}
	if n != 2 {
		t.Errorf("CountTable = %d, want 2", n)
	}

	page, err := s.SelectTablePage("users", 10, 0)
	if err != nil {
		t.Fatalf("SelectTablePage: %v", err)
	}
	if len(page.Rows) != 2 || page.Rows[0][1] != "Alice" {
		t.Errorf("page.Rows = %v", page.Rows)
	}
}

func TestOpenMissingFileErrors(t *testing.T) {
	cfg := conn.New(conn.DriverSQLite)
	cfg.Path = filepath.Join(t.TempDir(), "missing.db")

	if _, err := New(cfg); err == nil {
		t.Fatal("esperaba error para archivo inexistente")
	}
}
