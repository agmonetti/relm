package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/store"
	"github.com/agmonetti/relm/internal/tui/screens"
	"github.com/agmonetti/relm/internal/tui/styles"
)

// handleDetailKeys handles the keys of the detail view.
func (m *Model) handleDetailKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc || key.Matches(msg, m.keys.Back):
		m.showDetail = false
		m.detailScroll = 0
		m.detailDoc = ""
		m.detailGraph = nil
		m.detailCols = nil
		m.detailVals = nil
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
	m.detailDoc = ""
	m.detailGraph = nil
	m.detailScroll = 0
	m.showDetail = true
}

// openDetailFromBrowser opens the appropriate detail view for the current DataView item.
func (m *Model) openDetailFromBrowser(b *browser.Browser) {
	if b.Data == nil {
		return
	}
	switch v := b.Data.(type) {
	case *store.TabularData:
		if len(v.Rows) > 0 && b.Cursor >= 0 && b.Cursor < len(v.Rows) {
			cols := make([]string, len(b.Columns))
			for i, c := range b.Columns {
				cols[i] = c.Name
			}
			if len(cols) == 0 {
				cols = v.Columns
			}
			m.openDetail(b.ActiveTable, cols, v.Rows[b.Cursor])
		}
	case *store.DocumentData:
		if len(v.Documents) > 0 && b.Cursor >= 0 && b.Cursor < len(v.Documents) {
			doc := v.Documents[b.Cursor]
			m.detailTitle = fmt.Sprintf("Document: %s", doc.ID)
			m.detailDoc = doc.RawJSON
			m.detailCols = nil
			m.detailVals = nil
			m.detailGraph = nil
			m.detailScroll = 0
			m.showDetail = true
		}
	case *store.KeyValueData:
		m.detailTitle = fmt.Sprintf("Key: %s (%s)", v.Key, v.Type)
		var bld strings.Builder
		bld.WriteString(fmt.Sprintf("TTL: %s\n", v.TTL))
		for k, val := range v.Metadata {
			bld.WriteString(fmt.Sprintf("%s: %s\n", k, val))
		}
		bld.WriteString("\nEntries:\n")
		for _, e := range v.Entries {
			extra := ""
			if e.Extra != "" {
				extra = fmt.Sprintf(" [%s]", e.Extra)
			}
			bld.WriteString(fmt.Sprintf("  %s -> %s%s\n", e.Index, e.Value, extra))
		}
		m.detailDoc = bld.String()
		m.detailCols = nil
		m.detailVals = nil
		m.detailGraph = nil
		m.detailScroll = 0
		m.showDetail = true
	case *store.GraphData:
		if len(v.Nodes) > 0 && b.Cursor >= 0 && b.Cursor < len(v.Nodes) {
			node := v.Nodes[b.Cursor]
			m.detailTitle = fmt.Sprintf("Node: %s (:%s)", node.ID, strings.Join(node.Labels, ":"))
			m.detailGraph = &node
			m.detailDoc = ""
			m.detailCols = nil
			m.detailVals = nil
			m.detailScroll = 0
			m.showDetail = true
		}
	}
}

// renderActiveDetail renders the active detail view.
func (m *Model) renderActiveDetail(width, height int) string {
	if m.detailDoc != "" {
		return renderTextDetail(m.detailTitle, m.detailDoc, m.detailScroll, width, height)
	}
	if m.detailGraph != nil {
		return renderGraphNodeDetail(m.detailTitle, m.detailGraph, m.detailScroll, width, height)
	}
	return screens.RenderRowDetail(m.detailTitle, m.detailCols, m.detailVals, m.detailScroll, width, height)
}

func renderTextDetail(title, text string, scroll, width, height int) string {
	var b strings.Builder
	b.WriteString(styles.StyleHeader.Render(title) + "\n\n")
	lines := strings.Split(text, "\n")
	if scroll < 0 {
		scroll = 0
	}
	if scroll > len(lines) {
		scroll = len(lines)
	}
	end := scroll + height - 2
	if end > len(lines) {
		end = len(lines)
	}
	b.WriteString(strings.Join(lines[scroll:end], "\n"))
	return b.String()
}

func renderGraphNodeDetail(title string, node *store.GraphNode, scroll, width, height int) string {
	var lines []string
	lines = append(lines, styles.StyleHeader.Render(title), "")
	lines = append(lines, styles.StyleColHeader.Render("Properties:"))
	if len(node.Properties) == 0 {
		lines = append(lines, styles.StyleHeaderDim.Render("  (no properties)"))
	}
	for k, v := range node.Properties {
		lines = append(lines, fmt.Sprintf("  %-20s %s", k+":", v))
	}
	lines = append(lines, "", styles.StyleColHeader.Render("Relationships:"))
	if len(node.Incident) == 0 {
		lines = append(lines, styles.StyleHeaderDim.Render("  (no incident edges)"))
	}
	for _, inc := range node.Incident {
		lines = append(lines, fmt.Sprintf("  %s %s %s (%s)", inc.Direction, inc.Type, inc.TargetID, inc.TargetSummary))
	}

	if scroll < 0 {
		scroll = 0
	}
	if scroll > len(lines) {
		scroll = len(lines)
	}
	end := scroll + height
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[scroll:end], "\n")
}
