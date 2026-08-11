package sqlite

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agmonetti/relm/internal/conn"
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
		t.Errorf("bad constraints: %+v", cols)
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
		t.Fatal("expected an error for a missing file")
	}
}

func TestReadOnlyBlocksWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ro.db")
	// relm does not create files: we create the .db before opening it.
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("create file: %v", err)
	}

	cfg := conn.New(conn.DriverSQLite)
	cfg.Path = path
	s, err := New(cfg)
	if err != nil {
		t.Fatalf("New (crear): %v", err)
	}
	if _, err := s.Exec("CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)"); err != nil {
		t.Fatalf("Exec create table: %v", err)
	}
	s.Close()

	cfg.ReadOnly = true
	ro, err := New(cfg)
	if err != nil {
		t.Fatalf("New (read-only): %v", err)
	}
	defer ro.Close()

	if _, err := ro.Exec("INSERT INTO users (name) VALUES ('x')"); err == nil {
		t.Error("expected a write error in read-only mode")
	}
	tables, err := ro.Tables()
	if err != nil {
		t.Fatalf("Tables read-only: %v", err)
	}
	if len(tables) != 1 || tables[0] != "users" {
		t.Errorf("Tables = %v, want [users]", tables)
	}
}
