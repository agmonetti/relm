// Package editor maneja el estado del editor SQL: buffer, historial y resultados.
package editor

import (
	"strings"

	"github.com/agmonetti/relm/internal/store"
)

// EditorMode es el estado operativo del editor.
type EditorMode int

const (
	// EditorModeNormal: listo para editar o ejecutar.
	EditorModeNormal EditorMode = iota
	// EditorModeExecuting: una query está corriendo.
	EditorModeExecuting
)

// Editor mantiene el estado del editor SQL.
type Editor struct {
	Buffer  string
	History *History
	Result  *store.Result
	Error   string
	Mode    EditorMode
}

// New crea un editor con historial vacío.
func New() *Editor {
	return &Editor{History: NewHistory(), Mode: EditorModeNormal}
}

// Execute ejecuta el primer statement del buffer (equivalente a ExecuteAt en la
// línea 0). Se mantiene para tests y compatibilidad.
func (e *Editor) Execute(st store.Store) error {
	return e.ExecuteAt(st, 0)
}

// ExecuteAt ejecuta el statement que contiene la línea line (0-based) del buffer
// contra el store y guarda el resultado o el error. Con un solo statement se
// ejecuta siempre; con varios, el que está bajo el cursor. Los drivers de red no
// tienen multiStatements, así que nunca se mandan dos a la vez. Query con
// resultado (SELECT, WITH, PRAGMA...) usa Query(); el resto, Exec().
func (e *Editor) ExecuteAt(st store.Store, line int) error {
	stmts := splitStatements(e.Buffer)
	if len(stmts) == 0 {
		return nil // la UI muestra "escribe un query primero"
	}

	q := stmts[0].Text
	if len(stmts) > 1 {
		q = stmts[statementAt(stmts, line)].Text
	}

	e.Mode = EditorModeExecuting
	defer func() { e.Mode = EditorModeNormal }()

	if e.returnsRows(q) {
		res, err := st.Query(q)
		if err != nil {
			e.Result = nil
			e.Error = err.Error()
			return err
		}
		e.Result = res
		e.Error = ""
	} else {
		n, err := st.Exec(q)
		if err != nil {
			e.Result = nil
			e.Error = err.Error()
			return err
		}
		e.Result = &store.Result{Affected: n}
		e.Error = ""
	}

	e.History.Push(q)
	return nil
}

// Statement es un statement del buffer con su línea de inicio.
type Statement struct {
	Text string
	Line int // 0-based, línea del primer carácter no espaciado
}

// splitStatements parte el SQL en statements respetando strings (comillas
// simples, escapes "\" y duplicación "''") y los `;` fuera de ellos.
func splitStatements(sql string) []Statement {
	var stmts []Statement
	var b strings.Builder
	inQuote := false
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

	for i := 0; i < len(sql); i++ {
		c := sql[i]
		if c == '\n' {
			line++
		}
		switch {
		case inQuote:
			b.WriteByte(c)
			if c == '\\' && i+1 < len(sql) {
				b.WriteByte(sql[i+1])
				if sql[i+1] == '\n' {
					line++
				}
				i++
			} else if c == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' { // '' escapado
					b.WriteByte(sql[i+1])
					i++
				} else {
					inQuote = false
				}
			}
		case c == '\'':
			inQuote = true
			if !started {
				startLine = line
				started = true
			}
			b.WriteByte(c)
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
	flush()
	return stmts
}

// firstStatement devuelve el primer statement del buffer e indica si había más.
func firstStatement(sql string) (stmt string, multiple bool) {
	stmts := splitStatements(sql)
	if len(stmts) == 0 {
		return "", false
	}
	return stmts[0].Text, len(stmts) > 1
}

// statementAt devuelve el índice del statement que contiene la línea line:
// si varios statements empiezan en la misma línea se elige el primero (más
// seguro, p.ej. CREATE antes que INSERT); si el cursor está en espacios previos
// a todo statement, cae en el primero.
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

// Clear limpia el buffer y el resultado/error.
func (e *Editor) Clear() {
	e.Buffer = ""
	e.Result = nil
	e.Error = ""
	e.History.Reset()
}

// returnsRows indica si el query probablemente devuelve filas. No es 100%
// fiable entre motores; la UI decide cómo mostrar según el Result del store.
func (e *Editor) returnsRows(q string) bool {
	first := strings.ToUpper(strings.TrimSpace(q))
	for _, kw := range []string{"SELECT", "WITH", "PRAGMA", "SHOW", "EXPLAIN", "DESCRIBE", "VALUES", "TABLE"} {
		if strings.HasPrefix(first, kw+" ") || first == kw {
			return true
		}
	}
	return false
}
