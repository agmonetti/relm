// Package styles centralizes all lipgloss.Style used across the project.
package styles

import "github.com/charmbracelet/lipgloss"

// Colors that adapt to the terminal theme.
var (
	ColorPrimary  = lipgloss.AdaptiveColor{Light: "#1D9E75", Dark: "#5DCAA5"} // teal
	ColorAccent   = lipgloss.AdaptiveColor{Light: "#534AB7", Dark: "#AFA9EC"} // purple
	ColorMuted    = lipgloss.AdaptiveColor{Light: "#888780", Dark: "#B4B2A9"} // gray
	ColorError    = lipgloss.AdaptiveColor{Light: "#A32D2D", Dark: "#F09595"} // red
	ColorNull     = lipgloss.AdaptiveColor{Light: "#B4B2A9", Dark: "#5F5E5A"} // gray tenue
	ColorBorder   = lipgloss.AdaptiveColor{Light: "#D3D1C7", Dark: "#444441"} // gray borde
	ColorHeader   = lipgloss.AdaptiveColor{Light: "#4A4A4A", Dark: "#DDDDDD"} // texto
	ColorWarn     = lipgloss.AdaptiveColor{Light: "#B9932E", Dark: "#E0C568"} // amber
	ColorPillConn = lipgloss.AdaptiveColor{Light: "#B58900", Dark: "#F2C94A"} // yellow
)

// Precomputed package-level styles. Do not create styles in every View().
var (
	// StyleHeader is the header text.
	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	// StyleHeaderDim is the dim part of the header.
	StyleHeaderDim = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// StyleSidebarActive is the > cursor of the active table.
	StyleSidebarActive = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	// StyleSidebarItem is a regular sidebar table.
	StyleSidebarItem = lipgloss.NewStyle().Foreground(ColorHeader)

	// StyleError is the error text.
	StyleError = lipgloss.NewStyle().
			Foreground(ColorError)

	// StyleSuccess is a confirmation message (e.g. a completed export).
	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	// StyleWarn is an advisory notice (e.g. a limitation of the connection).
	StyleWarn = lipgloss.NewStyle().
			Foreground(ColorWarn)

	// StyleFooter is the screen footer.
	StyleFooter = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// StyleFooterKey is a pressing shortcut shown in the footer brackets.
	StyleFooterKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHeader)

	// StylePill is the shared base of the bracketed header pills: black text
	// on a solid background.
	StylePill = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Padding(0, 1)

	// StylePillConn is the connection pill (yellow).
	StylePillConn = StylePill.Background(ColorPillConn)

	// StylePillTable is the active-table pill (purple).
	StylePillTable = StylePill.Background(ColorAccent)

	// StylePillMode is the current-mode pill (teal).
	StylePillMode = StylePill.Background(ColorPrimary)

	// StylePillDefault is a neutral pill used when there is nothing to show
	// (no connection, no table).
	StylePillDefault = lipgloss.NewStyle().
				Foreground(ColorMuted)

	// StyleEditorLineNo is the line-number gutter of the SQL editor.
	StyleEditorLineNo = lipgloss.NewStyle().
				Foreground(ColorHeader).
				Background(ColorBorder).
				Padding(0, 1)

	// StyleEditorLineNoCursor is the gutter of the cursor line.
	StyleEditorLineNoCursor = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#000000")).
				Background(ColorAccent).
				Bold(true).
				Padding(0, 1)

	// StyleNull is the NULL value marker.
	StyleNull = lipgloss.NewStyle().
			Foreground(ColorNull)

	// StyleNullCursor is the NULL value marker on a selected row.
	StyleNullCursor = lipgloss.NewStyle().
			Foreground(ColorNull).
			Background(ColorAccent)

	// StyleCursor is the selected row in the browser.
	StyleCursor = lipgloss.NewStyle().
			Background(ColorAccent).
			Foreground(lipgloss.Color("#FFFFFF"))

	// StyleColHeader is the column header.
	StyleColHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent)

	// StyleBorder is a separator line.
	StyleBorder = lipgloss.NewStyle().Foreground(ColorBorder)

	// StyleLogo is the ASCII logo of the connection screen.
	StyleLogo = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	// StyleFieldLabel is a label of a connection form field.
	StyleFieldLabel = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorHeader)

	// StyleInputBox is the visible box of a form input.
	StyleInputBox = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// StyleInputBoxFocus is the box of the focused input.
	StyleInputBoxFocus = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)

	// StyleBtnPrimary is the primary button (solid teal background).
	StyleBtnPrimary = lipgloss.NewStyle().
			Background(ColorPrimary).
			Foreground(lipgloss.Color("#000000")).
			Bold(true).
			Padding(0, 3)

	// StyleBtnSecondary is a secondary button (dim gray background).
	StyleBtnSecondary = lipgloss.NewStyle().
				Background(ColorMuted).
				Foreground(lipgloss.Color("#000000")).
				Padding(0, 2)
)

// NullGlyph is the default visual glyph used for SQL NULL values.
const NullGlyph = "∅"

// NullCell returns the representation of a NULL value.
func NullCell() string { return StyleNull.Render(NullGlyph) }

// NullCellSelected returns the representation of a NULL value in a selected row.
func NullCellSelected() string { return StyleNullCursor.Render(NullGlyph) }

var (
	// StyleSidebarActiveTable marks the opened table when it is not under the
	// sidebar selection cursor.
	StyleSidebarActiveTable = lipgloss.NewStyle().
				Foreground(ColorAccent)

	// StylePane is the border of a non-focused workspace pane.
	StylePane = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder)

	// StylePaneFocus is the border of the focused workspace pane.
	StylePaneFocus = StylePane.
			BorderForeground(ColorAccent)

	// StylePaneTitle is the title shown inside the top border of a workspace pane.
	StylePaneTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent)

	// StyleBorderLine is the border rune line of a non-focused pane top border.
	StyleBorderLine = lipgloss.NewStyle().
			Foreground(ColorBorder)

	// StyleBorderLineFocus is the border rune line of the focused pane top border.
	StyleBorderLineFocus = lipgloss.NewStyle().
				Foreground(ColorAccent)

	// StyleOuterMargin insets the whole layout 1 char from the terminal edges.
	StyleOuterMargin = lipgloss.NewStyle().Margin(0, 1)
)
