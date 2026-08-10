// Package editor maneja el estado del editor SQL: buffer, historial y resultados.
package editor

import (
	"strings"

	"relm/internal/store"
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

// Execute ejecuta el buffer actual contra el store y guarda el resultado o el
// error. Query con resultado (SELECT, WITH, PRAGMA...) usa Query(); el resto, Exec().
func (e *Editor) Execute(st store.Store) error {
	q := strings.TrimSpace(e.Buffer)
	if q == "" {
		return nil // la UI muestra "escribe un query primero"
	}

	e.Mode = EditorModeExecuting
	defer func() { e.Mode = EditorModeNormal }()

	if e.returnsRows() {
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

// Clear limpia el buffer y el resultado/error.
func (e *Editor) Clear() {
	e.Buffer = ""
	e.Result = nil
	e.Error = ""
	e.History.Reset()
}

// returnsRows indica si el query probablemente devuelve filas. No es 100%
// fiable entre motores; la UI decide cómo mostrar según el Result del store.
func (e *Editor) returnsRows() bool {
	first := strings.ToUpper(strings.TrimSpace(e.Buffer))
	for _, kw := range []string{"SELECT", "WITH", "PRAGMA", "SHOW", "EXPLAIN", "DESCRIBE", "VALUES", "TABLE"} {
		if strings.HasPrefix(first, kw+" ") || first == kw {
			return true
		}
	}
	return false
}
