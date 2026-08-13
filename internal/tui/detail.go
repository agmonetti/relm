package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// handleDetailKeys handles the keys of the row detail view.
func (m *Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		m.showDetail = false
		m.detailScroll = 0
	case key.Matches(msg, m.keys.Up):
		m.detailScroll--
	case key.Matches(msg, m.keys.Down):
		m.detailScroll++
	case key.Matches(msg, m.keys.PageUp):
		m.detailScroll -= 10
	case key.Matches(msg, m.keys.PageDown):
		m.detailScroll += 10
	case key.Matches(msg, m.keys.First):
		m.detailScroll = 0
	case key.Matches(msg, m.keys.Last):
		m.detailScroll = 1 << 30 // clamped in the renderer
	}
	return m, nil
}

// openDetail shows the full values of a row.
func (m *Model) openDetail(title string, cols, vals []string) {
	m.detailTitle = title
	m.detailCols = cols
	m.detailVals = vals
	m.detailScroll = 0
	m.showDetail = true
}
