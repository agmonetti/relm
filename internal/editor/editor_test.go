package editor

import (
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
		t.Errorf("Error = %q, want empty", e.Error)
	}
	if e.Result == nil || len(e.Result.Columns) != 2 {
		t.Errorf("Result = %+v, want 2 columns", e.Result)
	}
	if e.Result.Affected != -1 {
		t.Errorf("Affected = %d, want -1 (read)", e.Result.Affected)
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
		t.Fatal("expected an SQL error")
	}
	if e.Error == "" || e.Result != nil {
		t.Errorf("Error=%q Result=%+v, want error and nil result", e.Error, e.Result)
	}
}

func TestEditor_ExecuteEmptyBuffer(t *testing.T) {
	st := newTestStore(t)
	e := New()
	e.Buffer = "   "
	if err := e.Execute(st); err != nil {
		t.Fatalf("Execute empty: %v", err)
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
	// an invalid query is not added to the history
	e.Buffer = "BROKEN SQL"
	_ = e.Execute(st)
	if e.History.Len() != 1 {
		t.Errorf("History.Len = %d, want 1 (only successful queries)", e.History.Len())
	}
}

func TestEditor_Clear(t *testing.T) {
	st := newTestStore(t)
	e := New()
	e.Buffer = "SELECT 1"
	_ = e.Execute(st)
	e.Clear()
	if e.Buffer != "" || e.Result != nil || e.Error != "" {
		t.Errorf("Clear did not reset: %+v", e)
	}
}

func TestEditor_ExecuteOnlyFirstStatement(t *testing.T) {
	st := newTestStore(t)
	e := New()
	e.Buffer = "CREATE TABLE t (x INT); INSERT INTO t VALUES (1)"
	if err := e.Execute(st); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// the second statement must not have run
	n, err := st.CountTable("t")
	if err != nil {
		t.Fatalf("CountTable: %v", err)
	}
	if n != 0 {
		t.Errorf("CountTable = %d, want 0 (INSERT must not run)", n)
	}
}

func TestEditor_ExecuteAtSelectsStatementAtCursor(t *testing.T) {
	st := newTestStore(t)
	e := New()
	e.Buffer = "SELECT 1;\nSELECT 2;\nSELECT * FROM users"

	// cursor on line 2 (0-based): the third statement
	if err := e.ExecuteAt(st, 2); err != nil {
		t.Fatalf("ExecuteAt: %v", err)
	}
	if e.Result == nil || len(e.Result.Columns) != 2 {
		t.Errorf("Result = %+v, want 2 columns from users", e.Result)
	}
}

func TestEditor_ExecuteAtSingleStatementIgnoresLine(t *testing.T) {
	st := newTestStore(t)
	e := New()
	e.Buffer = "INSERT INTO users (name) VALUES ('X')"
	if err := e.ExecuteAt(st, 5); err != nil {
		t.Fatalf("ExecuteAt: %v", err)
	}
	if e.Result == nil || e.Result.Affected != 1 {
		t.Errorf("Result = %+v, want Affected=1", e.Result)
	}
}

func TestEditor_ExecuteAtLineInPreamble(t *testing.T) {
	st := newTestStore(t)
	e := New()
	e.Buffer = "\n\nCREATE TABLE t (x INT);\nINSERT INTO t VALUES (1)"
	// cursor on line 0 (leading whitespace): falls into the first statement
	if err := e.ExecuteAt(st, 0); err != nil {
		t.Fatalf("ExecuteAt: %v", err)
	}
	if _, err := st.CountTable("t"); err != nil {
		t.Fatalf("CountTable: %v", err)
	}
}

func TestEditor_ExecuteAtSeparateLines(t *testing.T) {
	st := newTestStore(t)
	e := New()
	e.Buffer = "CREATE TABLE t (x INT);\nINSERT INTO t VALUES (1)"

	// cursor on line 0 → CREATE
	if err := e.ExecuteAt(st, 0); err != nil {
		t.Fatalf("ExecuteAt CREATE: %v", err)
	}

	// cursor on line 1 → INSERT
	if err := e.ExecuteAt(st, 1); err != nil {
		t.Fatalf("ExecuteAt INSERT: %v", err)
	}
	if e.Result == nil || e.Result.Affected != 1 {
		t.Errorf("Result = %+v, want Affected=1", e.Result)
	}
	n, _ := st.CountTable("t")
	if n != 1 {
		t.Errorf("CountTable = %d, want 1", n)
	}
}

func TestEditor_FirstStatementRespectsStrings(t *testing.T) {
	// ";" inside a string literal does not split the statement
	stmt, multiple := firstStatement("INSERT INTO users (name) VALUES ('a;b')")
	if multiple {
		t.Error("should not split on ';' inside a string")
	}
	if stmt != "INSERT INTO users (name) VALUES ('a;b')" {
		t.Errorf("stmt = %q", stmt)
	}

	// doubled quotes '' inside the string
	stmt, multiple = firstStatement("INSERT INTO t VALUES ('it''s; ok'); DROP TABLE x")
	if !multiple {
		t.Error("should detect the ';' after the string")
	}
	if stmt != "INSERT INTO t VALUES ('it''s; ok')" {
		t.Errorf("stmt = %q", stmt)
	}

	// escaped backslash inside the string
	stmt, multiple = firstStatement(`INSERT INTO t VALUES ('a\;b')`)
	if multiple {
		t.Error("should not split on an escaped ';'")
	}
	if stmt != `INSERT INTO t VALUES ('a\;b')` {
		t.Errorf("stmt = %q", stmt)
	}
}
