package editor

import (
	"context"
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
	if err := e.ExecuteAt(context.Background(), st, 2); err != nil {
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
	if err := e.ExecuteAt(context.Background(), st, 5); err != nil {
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
	if err := e.ExecuteAt(context.Background(), st, 0); err != nil {
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
	if err := e.ExecuteAt(context.Background(), st, 0); err != nil {
		t.Fatalf("ExecuteAt CREATE: %v", err)
	}

	// cursor on line 1 → INSERT
	if err := e.ExecuteAt(context.Background(), st, 1); err != nil {
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

func TestEditor_FirstStatementIgnoresSemicolonsInComments(t *testing.T) {
	cases := []struct{ sql, want string }{
		{"SELECT 1; -- note with ; inside\nSELECT 2", "SELECT 1"},
		{"SELECT 1 /* block ; with ; */; SELECT 2", "SELECT 1"},
		{"-- c ; d\nSELECT 1; SELECT 2", "SELECT 1"},
		{"/* c ; d */\nSELECT 1; SELECT 2", "SELECT 1"},
		{"SELECT * FROM t -- ; done\n", "SELECT * FROM t"},
		{"SELECT 1 # mysql ; comment\n; SELECT 2", "SELECT 1"},
		// SQL Server temp tables: #temp is not a comment
		{"CREATE TABLE #temp (x INT); SELECT * FROM #temp", "CREATE TABLE #temp (x INT)"},
	}
	for _, tc := range cases {
		stmt, _ := firstStatement(tc.sql)
		if stmt != tc.want {
			t.Errorf("firstStatement(%q) = %q, want %q", tc.sql, stmt, tc.want)
		}
	}
}

func TestEditor_CommentsDoNotGlueTokens(t *testing.T) {
	// a block comment acts as whitespace: tokens must not merge
	stmts := splitStatements("SELECT a/*x*/b")
	if len(stmts) != 1 || stmts[0].Text != "SELECT a b" {
		t.Errorf("splitStatements = %+v, want one statement 'SELECT a b'", stmts)
	}
}

func TestEditor_ExecuteSelectWithLeadingComment(t *testing.T) {
	st := newTestStore(t)
	e := New()
	e.Buffer = "-- load users\nSELECT * FROM users"
	if err := e.Execute(st); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if e.Result == nil || len(e.Result.Rows) != 0 {
		t.Errorf("Result = %+v, want a select result", e.Result)
	}
}

func TestEditor_ExecuteInsertReturning(t *testing.T) {
	st := newTestStore(t)
	e := New()
	// INSERT ... RETURNING must go through Query, not Exec
	e.Buffer = "INSERT INTO users (name) VALUES ('X') RETURNING id"
	if err := e.Execute(st); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// SQLite supports RETURNING; the result must carry columns
	if e.Result == nil || len(e.Result.Columns) == 0 {
		t.Errorf("Result = %+v, want columns from RETURNING", e.Result)
	}
}

func TestEditor_WroteFlag(t *testing.T) {
	st := newTestStore(t)
	cases := []struct {
		sql  string
		want bool
	}{
		{"SELECT * FROM users", false},
		{"WITH x AS (SELECT 1) SELECT * FROM x", false},
		{"INSERT INTO users (name) VALUES ('x')", true},
		{"INSERT INTO users (name) VALUES ('x') RETURNING id", true}, // rows but still a write
		{"UPDATE users SET name = 'x'", true},
		{"CREATE TABLE t (x INT)", true},
		{"ALTER TABLE users ADD COLUMN y INT", true},
		{"DELETE FROM users", true},
		{"DROP TABLE users", true},
	}
	for _, tc := range cases {
		e := New()
		e.Buffer = tc.sql
		if err := e.Execute(st); err != nil {
			t.Fatalf("Execute(%q): %v", tc.sql, err)
		}
		if got := e.Wrote; got != tc.want {
			t.Errorf("Wrote for %q = %v, want %v", tc.sql, got, tc.want)
		}
	}
}

func TestEditor_ResultTruncatedOverLimit(t *testing.T) {
	st := newTestStore(t)
	for i := 0; i < MaxResultRows+5; i++ {
		if _, err := st.Exec("INSERT INTO users (name) VALUES ('u')"); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	e := New()
	e.Buffer = "SELECT * FROM users"
	if err := e.Execute(st); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if e.Result == nil || !e.Result.Truncated {
		t.Errorf("Result.Truncated = %v, want true (result over the row cap)", e.Result)
	}
	if len(e.Result.Rows) != MaxResultRows {
		t.Errorf("Rows = %d, want %d", len(e.Result.Rows), MaxResultRows)
	}
}

func TestEditor_ReturnsRows(t *testing.T) {
	e := New()
	cases := []struct {
		sql  string
		want bool
	}{
		{"SELECT 1", true},
		{"SELECT;", true},
		{"-- c\nSELECT 1", true},
		{"/* c */ SELECT 1", true},
		{"WITH x AS (SELECT 1) SELECT * FROM x", true},
		{"INSERT INTO t VALUES (1) RETURNING id", true},
		{"INSERT INTO t VALUES (1)", false},
		{"UPDATE t SET x = 1", false},
		{"DELETE FROM t", false},
		{"SELECT * FROM #temp", true},
		{"(SELECT 1)", true},
		{"((SELECT 1) UNION (SELECT 2))", true},
	}
	for _, tc := range cases {
		if got := e.returnsRows(tc.sql); got != tc.want {
			t.Errorf("returnsRows(%q) = %v, want %v", tc.sql, got, tc.want)
		}
	}
}

func TestEditor_SplitStatementsSpecialQuotes(t *testing.T) {
	// Double-quoted identifier with semicolon
	stmt, multiple := firstStatement(`SELECT * FROM "users;backup"; SELECT 2`)
	if !multiple {
		t.Error("expected multiple statements")
	}
	if stmt != `SELECT * FROM "users;backup"` {
		t.Errorf("stmt = %q", stmt)
	}

	// Backtick identifier with semicolon
	stmt, multiple = firstStatement("SELECT * FROM `users;backup`; SELECT 2")
	if !multiple {
		t.Error("expected multiple statements")
	}
	if stmt != "SELECT * FROM `users;backup`" {
		t.Errorf("stmt = %q", stmt)
	}

	// Bracket identifier with semicolon
	stmt, multiple = firstStatement("SELECT * FROM [users;backup]; SELECT 2")
	if !multiple {
		t.Error("expected multiple statements")
	}
	if stmt != "SELECT * FROM [users;backup]" {
		t.Errorf("stmt = %q", stmt)
	}

	// PostgreSQL dollar quotes with semicolon inside
	sql := "CREATE FUNCTION foo() RETURNS void AS $$ BEGIN SELECT 1; END; $$ LANGUAGE plpgsql; SELECT 2"
	stmt, multiple = firstStatement(sql)
	if !multiple {
		t.Error("expected multiple statements")
	}
	if stmt != "CREATE FUNCTION foo() RETURNS void AS $$ BEGIN SELECT 1; END; $$ LANGUAGE plpgsql" {
		t.Errorf("stmt = %q", stmt)
	}
}

func TestEditor_ExecuteValuesQuery(t *testing.T) {
	st := newTestStore(t)
	e := New()
	e.Buffer = "VALUES (1), (2)"
	if err := e.Execute(st); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if e.Result == nil || len(e.Result.Rows) != 2 {
		t.Fatalf("Result = %+v, want 2 rows", e.Result)
	}
}


