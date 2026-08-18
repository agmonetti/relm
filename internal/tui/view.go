package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/agmonetti/relm/internal/tui/screens"
	"github.com/agmonetti/relm/internal/tui/styles"
)

// View implements tea.Model.
func (m *Model) View() string {
	if m.showHelp {
		return m.renderHelp()
	}
	return m.render()
}

func (m *Model) render() string {
	innerW := m.width - 2 // 1 char outer margin on each side
	if innerW < 1 {
		innerW = 1
	}

	header := m.renderHeader()
	footer := m.renderFooter()
	// the layout is wrapped in a 1-char margin on all four sides: a blank
	// line above the header and below the footer, plus the lateral padding
	contentHeight := m.height - 4
	if contentHeight < 1 {
		contentHeight = 1
	}

	var content string
	switch m.screen {
	case ScreenConnect:
		content = m.connect.View(innerW, contentHeight)
	case ScreenSettings:
		content = m.settings.View(innerW, contentHeight)
	case ScreenWorkspace:
		if m.showDetail {
			content = screens.RenderRowDetail(m.detailTitle, m.detailCols, m.detailVals,
				m.detailScroll, innerW, contentHeight)
		} else if m.exporting {
			content = m.renderExportPrompt(innerW, contentHeight)
		} else {
			layout := screens.ComputeLayout(innerW, contentHeight, m.showSidebar, m.sidebarW, m.editorH)
			content = screens.RenderWorkspace(m.browser, m.editorScreen, m.editor,
				m.focus, m.structure, layout, m.sidebarCursor, innerW, contentHeight)
		}
	}

	body := content
	if m.err != "" {
		body = content + "\n" + styles.StyleError.Render(m.err)
	}
	if m.warn != "" {
		body = body + "\n" + styles.StyleWarn.Render(m.warn)
	}
	if m.exported != "" {
		body = body + "\n" + styles.StyleSuccess.Render(m.exported)
	}
	body = styles.StyleOuterMargin.Render(body)

	return lipgloss.JoinVertical(lipgloss.Left,
		" ",
		" "+header+" ",
		body,
		" "+footer+" ",
		" ")
}

func (m *Model) renderHeader() string {
	label := "no connection"
	if m.store != nil {
		label = m.cfgLabel
	}
	mode := ""
	switch m.screen {
	case ScreenSettings:
		mode = "settings"
	case ScreenWorkspace:
		switch {
		case m.focus == screens.FocusEditor:
			mode = "editor"
		case m.focus == screens.FocusSidebar:
			mode = "tables"
		case m.structure:
			mode = "structure"
		default:
			mode = "browser"
		}
	}
	table := "—"
	if m.browser != nil && m.browser.ActiveTable != "" {
		table = m.browser.ActiveTable
	}
	return styles.StyleHeader.Render("relm") +
		styles.StyleHeaderDim.Render(" · "+label) +
		styles.StyleHeaderDim.Render(" · "+table) +
		styles.StyleHeaderDim.Render(" · "+mode)
}

func (m *Model) renderFooter() string {
	left := ""
	switch m.screen {
	case ScreenConnect:
		left = "↑↓ saved · tab engine/fields · ←→ engine · enter connect · ctrl+s save · d delete · ctrl+p settings"
	case ScreenSettings:
		left = "enter save · esc back"
	case ScreenWorkspace:
		switch m.focus {
		case screens.FocusSidebar:
			left = "↑↓ tables · enter open · tab next · ? help · ctrl+p settings"
		case screens.FocusMain:
			if m.structure {
				left = "esc back · tab next"
			} else {
				left = "↑↓ rows · i structure · v detail · r refresh · pgup/pgdn page · tab next · right-click resize"
			}
		case screens.FocusEditor:
			left = "ctrl+r run · ctrl+l clear · esc back"
		}
	}

	right := ""
	switch {
	case m.loading:
		right = m.spinner.View() + " running query…"
	case m.connecting:
		right = m.spinner.View() + " connecting…"
	case m.navigating:
		right = m.spinner.View() + " loading…"
	}
	if right == "" && m.screen == ScreenWorkspace && m.focus != screens.FocusEditor &&
		m.browser != nil && m.browser.ActiveTable != "" && m.browser.TotalRows > 0 {
		first := m.browser.Page*m.browser.PageSize + 1
		last := (m.browser.Page + 1) * m.browser.PageSize
		if last > m.browser.TotalRows {
			last = m.browser.TotalRows
		}
		arrow := ""
		if m.browser.HasNextPage() {
			arrow = " ▼"
		}
		right = fmt.Sprintf("%d-%d/%d%s", first, last, m.browser.TotalRows, arrow)
	}

	pad := (m.width - 2) - lipgloss.Width(left) - lipgloss.Width(right)
	if pad < 1 {
		pad = 1
	}
	return styles.StyleFooter.Render(left + padSpaces(pad) + right)
}

func (m *Model) renderHelp() string {
	var out string
	for _, group := range m.keys.FullHelp() {
		for _, b := range group {
			out += fmt.Sprintf("  %-18s %s\n", b.Help().Key, b.Help().Desc)
		}
		out += "\n"
	}
	out += fmt.Sprintf("  %-18s %s\n", "right-click drag", "resize panes")
	out += fmt.Sprintf("  %-18s %s\n", "click", "focus / select row")
	out += fmt.Sprintf("  %-18s %s\n", "wheel", "scroll pane")
	return lipgloss.JoinVertical(lipgloss.Left,
		styles.StyleHeader.Render("Help"),
		out,
		styles.StyleHeaderDim.Render("? to close"),
	)
}

func padSpaces(n int) string {
	if n < 0 {
		n = 0
	}
	return fmt.Sprintf("%*s", n, "")
}
