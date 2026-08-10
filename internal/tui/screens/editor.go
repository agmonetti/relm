package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/agmonetti/relm/internal/editor"
	"github.com/agmonetti/relm/internal/tui/styles"
)

// EditorScreen mantiene el input multilínea del editor SQL y su renderizado.
type EditorScreen struct {
	ta textarea.Model
}

// NewEditorScreen crea el textarea del editor.
func NewEditorScreen() *EditorScreen {
	ta := textarea.New()
	ta.Placeholder = "SELECT * FROM tabla LIMIT 10"
	ta.ShowLineNumbers = true
	ta.Prompt = ""
	ta.CharLimit = 0
	return &EditorScreen{ta: ta}
}

// SetValue reemplaza el contenido del textarea.
func (s *EditorScreen) SetValue(v string) { s.ta.SetValue(v) }

// Value devuelve el contenido actual del textarea.
func (s *EditorScreen) Value() string { return s.ta.Value() }

// Focus enfoca el textarea y lleva el cursor al final.
func (s *EditorScreen) Focus() tea.Cmd {
	cmd := s.ta.Focus()
	s.ta.CursorEnd()
	return cmd
}

// FocusStart enfoca el textarea y lleva el cursor al inicio.
func (s *EditorScreen) FocusStart() tea.Cmd {
	cmd := s.ta.Focus()
	s.ta.CursorStart()
	return cmd
}

// Blur desenfoca el textarea.
func (s *EditorScreen) Blur() { s.ta.Blur() }

// Update pasa el mensaje al textarea.
func (s *EditorScreen) Update(msg tea.Msg) (*EditorScreen, tea.Cmd) {
	ta, cmd := s.ta.Update(msg)
	s.ta = ta
	return s, cmd
}

// AtBoundary indica si el cursor está en el borde superior (up=true) o
// inferior (up=false) del input.
func (s *EditorScreen) AtBoundary(up bool) bool {
	if up {
		return s.ta.Line() == 0
	}
	return s.ta.Line() >= s.ta.LineCount()-1
}

// Line devuelve la línea actual del cursor.
func (s *EditorScreen) Line() int { return s.ta.Line() }

// LineCount devuelve la cantidad de líneas del input.
func (s *EditorScreen) LineCount() int { return s.ta.LineCount() }

// View renderiza input + resultados del editor.
func (s *EditorScreen) View(e *editor.Editor, width, height int) string {
	if width < 1 {
		width = 80
	}
	s.ta.SetWidth(width - 2)
	inputHeight := height / 2
	if inputHeight < 3 {
		inputHeight = 3
	}
	s.ta.SetHeight(inputHeight - 1)

	var b strings.Builder
	b.WriteString(s.ta.View())

	// separador
	b.WriteString("\n" + styles.StyleBorder.Render(strings.Repeat("─", width-2)) + "\n")

	if e.Error != "" {
		b.WriteString(styles.StyleError.Render(e.Error))
		return b.String()
	}

	if e.Result == nil {
		b.WriteString(styles.StyleHeaderDim.Render("  ctrl+r para ejecutar"))
		return b.String()
	}

	if len(e.Result.Columns) > 0 {
		b.WriteString(renderDataTable(e.Result.Columns, e.Result.Rows, -1, width-2, height-inputHeight))
	} else if e.Result.Affected >= 0 {
		b.WriteString(fmt.Sprintf("  %d filas afectadas", e.Result.Affected))
	}
	return b.String()
}
