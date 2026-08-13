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

	// result viewport: the query results table can be scrolled and a row can
	// be selected with the mouse
	resultScroll int
	resultCursor int
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

// SetResultScroll sets the scroll offset of the query results table.
func (s *EditorScreen) SetResultScroll(n int) { s.resultScroll = n }

// ResultScroll returns the scroll offset of the query results table.
func (s *EditorScreen) ResultScroll() int { return s.resultScroll }

// SetResultCursor selects a row of the query results (-1 = none).
func (s *EditorScreen) SetResultCursor(n int) { s.resultCursor = n }

// ResultCursor returns the selected row of the query results.
func (s *EditorScreen) ResultCursor() int { return s.resultCursor }

// ResetResult resets the results viewport after a new query.
func (s *EditorScreen) ResetResult() {
	s.resultScroll = 0
	s.resultCursor = -1
}

// EditorInputHeight returns the textarea height for an editor pane content
// height, matching View.
func EditorInputHeight(contentH int) int {
	h := contentH / 2
	if h < 3 {
		h = 3
	}
	return h
}

// EditorResultsLayout returns the content line where the query results table
// starts and how many data rows it can show, for the given editor content
// height. Used both by View and the mouse hit-test so they agree.
func EditorResultsLayout(contentH int) (startLine, dataRows int) {
	ih := EditorInputHeight(contentH)
	regionH := contentH - ih
	if regionH < 2 {
		regionH = 2
	}
	return ih, regionH - 1
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
		rows := e.Result.Rows
		resH := height - inputHeight
		if resH < 2 {
			resH = 2
		}
		dataRows := resH - 1
		maxScroll := len(rows) - dataRows
		if maxScroll < 0 {
			maxScroll = 0
		}
		scroll := s.resultScroll
		if scroll < 0 {
			scroll = 0
		}
		if scroll > maxScroll {
			scroll = maxScroll
		}
		view := rows[scroll:]
		cursor := -1
		if s.resultCursor >= scroll {
			cursor = s.resultCursor - scroll
			if cursor >= len(view) {
				cursor = len(view) - 1
			}
		}
		b.WriteString(RenderDataTable(e.Result.Columns, view, cursor, width-2, resH))
	} else if e.Result.Affected >= 0 {
		b.WriteString(fmt.Sprintf("  %d rows affected", e.Result.Affected))
	}
	return b.String()
}
