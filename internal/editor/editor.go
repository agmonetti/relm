// Package editor manages the SQL editor state: buffer, history and results.
package editor

import (
	"context"
	"strings"

	"github.com/agmonetti/relm/internal/store"
)

// EditorMode is the operational state of the editor.
type EditorMode int

const (
	// EditorModeNormal: ready to edit or run.
	EditorModeNormal EditorMode = iota
	// EditorModeExecuting: a query is running.
	EditorModeExecuting
)

// Editor keeps the state of the SQL editor.
type Editor struct {
	Buffer  string
	History *History
	Result  *store.Result
	Error   string
	Mode    EditorMode
	// LastQuery is the exact statement that was executed (used by the UI to
	// push it into the persistent history without crossing goroutines).
	LastQuery string
}

// New creates an editor with an empty history.
func New() *Editor {
	return &Editor{History: NewHistory(), Mode: EditorModeNormal}
}

// Execute runs the first statement of the buffer (equivalent to ExecuteAt on
// line 0). Kept for tests and compatibility.
func (e *Editor) Execute(st store.Store) error {
	return e.ExecuteAt(context.Background(), st, 0)
}

// ExecuteAt runs the statement that contains line line (0-based) of the buffer
// against the store and stores the result or the error. With a single statement
// it always runs it; with several, the one under the cursor. Network drivers do
// not have multiStatements, so two are never sent at once. Queries that return
// rows (SELECT, WITH, PRAGMA...) use Query(); everything else uses Exec(). The
// context can cancel the query or bound it with a timeout.
func (e *Editor) ExecuteAt(ctx context.Context, st store.Store, line int) error {
	stmts := splitStatements(e.Buffer)
	if len(stmts) == 0 {
		return nil // the UI shows "write a query first"
	}

	q := stmts[0].Text
	if len(stmts) > 1 {
		q = stmts[statementAt(stmts, line)].Text
	}

	e.Mode = EditorModeExecuting
	defer func() { e.Mode = EditorModeNormal }()

	if e.returnsRows(q) {
		res, err := st.QueryContext(ctx, q)
		if err != nil {
			e.Result = nil
			e.Error = err.Error()
			return err
		}
		e.Result = res
		e.Error = ""
	} else {
		n, err := st.ExecContext(ctx, q)
		if err != nil {
			e.Result = nil
			e.Error = err.Error()
			return err
		}
		e.Result = &store.Result{Affected: n}
		e.Error = ""
	}

	e.History.Push(q)
	e.LastQuery = q
	return nil
}

// Statement is a statement of the buffer with its start line.
type Statement struct {
	Text string
	Line int // 0-based, line of the first non-space character
}

// splitStatements splits the SQL into statements respecting strings (single
// quotes, "\" escapes and "''" duplication), comments (`--`, `/* */` and
// MySQL's `#`) and the `;` outside them. Comments are replaced by a space so
// adjacent tokens are not glued together.
func splitStatements(sql string) []Statement {
	var stmts []Statement
	var b strings.Builder
	line := 0
	startLine := 0
	started := false

	flush := func() {
		if t := strings.TrimSpace(b.String()); t != "" {
			stmts = append(stmts, Statement{Text: t, Line: startLine})
		}
		b.Reset()
		started = false
	}

	// 0 = code, 1 = single-quoted string, 2 = line comment, 3 = block comment
	mode := 0
	for i := 0; i < len(sql); i++ {
		c := sql[i]
		if c == '\n' {
			line++
		}
		switch mode {
		case 1: // inside a string literal
			b.WriteByte(c)
			if c == '\\' && i+1 < len(sql) {
				b.WriteByte(sql[i+1])
				if sql[i+1] == '\n' {
					line++
				}
				i++
			} else if c == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' { // '' escaped
					b.WriteByte(sql[i+1])
					i++
				} else {
					mode = 0
				}
			}
		case 2: // line comment: skipped until the end of the line
			if c == '\n' {
				// the newline separates tokens, keep it
				b.WriteByte(c)
				mode = 0
			}
		case 3: // block comment: skipped until "*/"
			if c == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				i++
				mode = 0
			}
		default: // code
			switch {
			case c == '\'':
				mode = 1
				if !started {
					startLine = line
					started = true
				}
				b.WriteByte(c)
			case c == '-' && i+1 < len(sql) && sql[i+1] == '-':
				mode = 2
				i++
			case c == '#' && (i+1 >= len(sql) || !isIdentChar(sql[i+1])):
				// MySQL comment; not a comment when it prefixes a temp table
				// identifier such as SQL Server's "#temp".
				mode = 2
			case c == '/' && i+1 < len(sql) && sql[i+1] == '*':
				mode = 3
				i++
				b.WriteByte(' ')
			case c == ';':
				flush()
			default:
				if !started && !isSpace(c) {
					startLine = line
					started = true
				}
				b.WriteByte(c)
			}
		}
	}
	flush()
	return stmts
}

// isIdentChar reports whether c can be part of an identifier (letters, digits,
// underscore, dollar) or the SQL Server temp-table "#" prefix.
func isIdentChar(c byte) bool {
	switch {
	case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		return true
	case c == '_', c == '$':
		return true
	}
	return false
}

// firstStatement returns the first statement of the buffer and whether there were more.
func firstStatement(sql string) (stmt string, multiple bool) {
	stmts := splitStatements(sql)
	if len(stmts) == 0 {
		return "", false
	}
	return stmts[0].Text, len(stmts) > 1
}

// statementAt returns the index of the statement that contains line line:
// if several statements start on the same line, the first one is chosen (safer,
// e.g. CREATE before INSERT); if the cursor is in whitespace before any
// statement, it falls into the first one.
func statementAt(stmts []Statement, line int) int {
	best := 0
	for i, s := range stmts {
		if s.Line > line {
			break
		}
		if s.Line == line {
			best = i
			break
		}
		best = i
	}
	return best
}

func isSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\r', '\n':
		return true
	}
	return false
}

// Clear resets the buffer and the result/error.
func (e *Editor) Clear() {
	e.Buffer = ""
	e.Result = nil
	e.Error = ""
	e.History.Reset()
}

// returnsRows reports whether the query probably returns rows. It is not 100%
// reliable across engines; the UI decides how to render based on the store Result.
func (e *Editor) returnsRows(q string) bool {
	switch firstKeyword(q) {
	case "SELECT", "WITH", "PRAGMA", "SHOW", "EXPLAIN", "DESCRIBE", "VALUES", "TABLE":
		return true
	case "INSERT", "UPDATE", "DELETE", "MERGE":
		// e.g. PostgreSQL's INSERT ... RETURNING produces rows
		return strings.Contains(strings.ToUpper(q), "RETURNING")
	}
	return false
}

// firstKeyword returns the first keyword of the SQL, skipping leading
// whitespace and comments, so a leading "-- note" or a bare "SELECT;" are
// recognized.
func firstKeyword(q string) string {
	s := strings.ToUpper(q)
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n':
			i++
		case c == '-' && i+1 < len(s) && s[i+1] == '-':
			i += 2
			for i < len(s) && s[i] != '\n' {
				i++
			}
		case c == '#' && (i+1 >= len(s) || !isIdentChar(s[i+1])):
			i++
			for i < len(s) && s[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(s) && s[i+1] == '*':
			i += 2
			for i+1 < len(s) && !(s[i] == '*' && s[i+1] == '/') {
				i++
			}
			i += 2
		default:
			j := i
			for j < len(s) && isIdentChar(s[j]) {
				j++
			}
			return s[i:j]
		}
	}
	return ""
}
