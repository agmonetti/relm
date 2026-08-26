package store

import (
	"strings"
)

// Statement is a statement of the buffer with its start line.
type Statement struct {
	Text string
	Line int // 0-based, line of the first non-space character
}

// SplitStatements splits the SQL into statements respecting strings (single
// quotes, "\" escapes and "''" duplication), comments (`--`, `/* */` and
// MySQL's `#`) and the `;` outside them. Comments are replaced by a space so
// adjacent tokens are not glued together.
func SplitStatements(sql string) []Statement {
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

	// 0 = code, 1 = single-quoted string, 2 = line comment, 3 = block comment,
	// 4 = double-quoted ident, 5 = backtick ident, 6 = bracket ident, 7 = dollar-quoted string
	mode := 0
	dollarTag := ""
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
		case 4: // double-quoted identifier "..."
			b.WriteByte(c)
			if c == '"' {
				if i+1 < len(sql) && sql[i+1] == '"' { // "" escaped
					b.WriteByte(sql[i+1])
					i++
				} else {
					mode = 0
				}
			}
		case 5: // backtick-quoted identifier `...`
			b.WriteByte(c)
			if c == '`' {
				if i+1 < len(sql) && sql[i+1] == '`' { // `` escaped
					b.WriteByte(sql[i+1])
					i++
				} else {
					mode = 0
				}
			}
		case 6: // bracket-quoted identifier [...]
			b.WriteByte(c)
			if c == ']' {
				if i+1 < len(sql) && sql[i+1] == ']' { // ]] escaped
					b.WriteByte(sql[i+1])
					i++
				} else {
					mode = 0
				}
			}
		case 7: // dollar-quoted string $tag$...$tag$
			if strings.HasPrefix(sql[i:], dollarTag) {
				b.WriteString(dollarTag)
				// add line counts if dollarTag has newlines (unlikely but safe)
				for k := 0; k < len(dollarTag); k++ {
					if dollarTag[k] == '\n' {
						line++
					}
				}
				i += len(dollarTag) - 1
				mode = 0
			} else {
				b.WriteByte(c)
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
			case c == '"':
				mode = 4
				if !started {
					startLine = line
					started = true
				}
				b.WriteByte(c)
			case c == '`':
				mode = 5
				if !started {
					startLine = line
					started = true
				}
				b.WriteByte(c)
			case c == '[':
				mode = 6
				if !started {
					startLine = line
					started = true
				}
				b.WriteByte(c)
			case c == '$':
				if j := i + 1; j < len(sql) {
					for j < len(sql) && isTagChar(sql[j]) {
						j++
					}
					if j < len(sql) && sql[j] == '$' {
						dollarTag = sql[i : j+1]
						mode = 7
						if !started {
							startLine = line
							started = true
						}
						b.WriteString(dollarTag)
						i = j
						break
					}
				}
				if !started && !isSpace(c) {
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

// IsIdentChar reports whether c can be part of an identifier (letters, digits,
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

// IsTagChar reports whether c can be part of a PostgreSQL dollar-quote tag (letters, digits, underscore).
func isTagChar(c byte) bool {
	switch {
	case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9':
		return true
	case c == '_':
		return true
	}
	return false
}

// StatementAt returns the index of the statement that contains line line:
// if several statements start on the same line, the first one is chosen (safer,
// e.g. CREATE before INSERT); if the cursor is in whitespace before any
// statement, it falls into the first one.
func StatementAt(stmts []Statement, line int) int {
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

// IsSQLWrite reports whether the statement may change the schema or the data,
// regardless of whether it also returns rows (e.g. INSERT ... RETURNING). The
// heuristic only inspects the leading keyword; exotic forms (a WITH that wraps
// a DELETE) are missed and simply keep the UI from auto-refreshing.
func IsSQLWrite(q string) bool {
	switch FirstSQLKeyword(q) {
	case "INSERT", "UPDATE", "DELETE", "MERGE", "REPLACE", "CREATE", "DROP",
		"ALTER", "TRUNCATE", "GRANT", "REVOKE", "ANALYZE", "VACUUM",
		"ATTACH", "DETACH", "REINDEX", "COMMENT":
		return true
	}
	return false
}

// ReturnsSQLRows reports whether the query probably returns rows.
func ReturnsSQLRows(q string) bool {
	switch FirstSQLKeyword(q) {
	case "SELECT", "WITH", "PRAGMA", "SHOW", "EXPLAIN", "DESCRIBE", "VALUES", "TABLE":
		return true
	case "INSERT", "UPDATE", "DELETE", "MERGE":
		return strings.Contains(strings.ToUpper(q), "RETURNING")
	}
	return false
}

// FirstSQLKeyword returns the first keyword of the SQL, skipping leading
// whitespace, parentheses and comments.
func FirstSQLKeyword(q string) string {
	s := strings.ToUpper(q)
	i := 0
	for i < len(s) {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '(':
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
