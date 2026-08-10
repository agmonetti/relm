package editor

import (
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
	if _, err := st.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)`); err != nil {
		t.Fatalf("create: %v", err)
	}
	return st
}

func TestEditor_ExecuteSelect(t *testing.T) {
	st := newTestStore(t)
	e := New()
	e.Buffer = "SELECT * FROM users"
	if err := e.Execute(st); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if e.Error != "" {
		t.Errorf("Error = %q, want vacío", e.Error)
	}
	if e.Result == nil || len(e.Result.Columns) != 2 {
		t.Errorf("Result = %+v, want 2 columnas", e.Result)
	}
	if e.Result.Affected != -1 {
		t.Errorf("Affected = %d, want -1 (lectura)", e.Result.Affected)
	}
}

func TestEditor_ExecuteInsert(t *testing.T) {
	st := newTestStore(t)
	e := New()
	e.Buffer = "INSERT INTO users (name) VALUES ('X')"
	if err := e.Execute(st); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if e.Result == nil || e.Result.Affected != 1 {
		t.Errorf("Result = %+v, want Affected=1", e.Result)
	}
}

func TestEditor_ExecuteInvalidSQL(t *testing.T) {
	st := newTestStore(t)
	e := New()
	e.Buffer = "SELLECT * FROM users"
	err := e.Execute(st)
	if err == nil {
		t.Fatal("esperaba error de SQL")
	}
	if e.Error == "" || e.Result != nil {
		t.Errorf("Error=%q Result=%+v, want error y nil result", e.Error, e.Result)
	}
}

func TestEditor_ExecuteEmptyBuffer(t *testing.T) {
	st := newTestStore(t)
	e := New()
	e.Buffer = "   "
	if err := e.Execute(st); err != nil {
		t.Fatalf("Execute vacío: %v", err)
	}
	if e.Result != nil {
		t.Errorf("Result = %+v, want nil", e.Result)
	}
}

func TestEditor_HistoryPushedOnExecute(t *testing.T) {
	st := newTestStore(t)
	e := New()
	e.Buffer = "SELECT 1"
	_ = e.Execute(st)
	if e.History.Len() != 1 {
		t.Errorf("History.Len = %d, want 1", e.History.Len())
	}
	// Query inválida no entra al historial
	e.Buffer = "BROKEN SQL"
	_ = e.Execute(st)
	if e.History.Len() != 1 {
		t.Errorf("History.Len = %d, want 1 (solo queries exitosas)", e.History.Len())
	}
}

func TestEditor_Clear(t *testing.T) {
	st := newTestStore(t)
	e := New()
	e.Buffer = "SELECT 1"
	_ = e.Execute(st)
	e.Clear()
	if e.Buffer != "" || e.Result != nil || e.Error != "" {
		t.Errorf("Clear no limpió: %+v", e)
	}
}
