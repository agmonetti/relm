package screens

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/agmonetti/relm/internal/editor"
	"github.com/agmonetti/relm/internal/tui/styles"
)

// EditorScreen keeps the multiline SQL editor input and its rendering.
type EditorScreen struct {
	ta textarea.Model
}

// NewEditorScreen creates the editor textarea.
func NewEditorScreen() *EditorScreen {
	ta := textarea.New()
	ta.Cursor.BlinkSpeed = CursorBlink
	ta.Placeholder = "SELECT * FROM table LIMIT 10"
	ta.ShowLineNumbers = true
	ta.Prompt = ""
	ta.CharLimit = 0
	return &EditorScreen{ta: ta}
}

// SetValue replaces the textarea content.
func (s *EditorScreen) SetValue(v string) { s.ta.SetValue(v) }

// Value returns the current textarea content.
func (s *EditorScreen) Value() string { return s.ta.Value() }

// Focus focuses the textarea and moves the cursor to the end.
func (s *EditorScreen) Focus() tea.Cmd {
	cmd := s.ta.Focus()
	s.ta.CursorEnd()
	return cmd
}

// FocusStart focuses the textarea and moves the cursor to the start.
func (s *EditorScreen) FocusStart() tea.Cmd {
	cmd := s.ta.Focus()
	s.ta.CursorStart()
	return cmd
}

// Blur blurs the textarea.
func (s *EditorScreen) Blur() { s.ta.Blur() }

// Update forwards the message to the textarea.
func (s *EditorScreen) Update(msg tea.Msg) (*EditorScreen, tea.Cmd) {
	ta, cmd := s.ta.Update(msg)
	s.ta = ta
	return s, cmd
}

// AtBoundary reports whether the cursor is at the top (up=true) or bottom
// (up=false) edge of the input.
func (s *EditorScreen) AtBoundary(up bool) bool {
	if up {
		return s.ta.Line() == 0
	}
	return s.ta.Line() >= s.ta.LineCount()-1
}

// Line returns the current cursor line.
func (s *EditorScreen) Line() int { return s.ta.Line() }

// LineCount returns the number of lines in the input.
func (s *EditorScreen) LineCount() int { return s.ta.LineCount() }

// View renders the editor input + results.
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

	// separator
	b.WriteString("\n" + styles.StyleBorder.Render(strings.Repeat("─", width-2)) + "\n")

	if e.Error != "" {
		b.WriteString(styles.StyleError.Render(e.Error))
		return b.String()
	}

	if e.Result == nil {
		b.WriteString(styles.StyleHeaderDim.Render("  ctrl+r to run"))
		return b.String()
	}

	if len(e.Result.Columns) > 0 {
		b.WriteString(RenderDataTable(e.Result.Columns, e.Result.Rows, -1, width-2, height-inputHeight))
	} else if e.Result.Affected >= 0 {
		b.WriteString(fmt.Sprintf("  %d rows affected", e.Result.Affected))
	}
	return b.String()
}
