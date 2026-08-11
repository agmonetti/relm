// Package styles centralizes all lipgloss.Style used across the project.
package styles

import "github.com/charmbracelet/lipgloss"

// Colors that adapt to the terminal theme.
var (
	ColorPrimary = lipgloss.AdaptiveColor{Light: "#1D9E75", Dark: "#5DCAA5"} // teal
	ColorAccent  = lipgloss.AdaptiveColor{Light: "#534AB7", Dark: "#AFA9EC"} // purple
	ColorMuted   = lipgloss.AdaptiveColor{Light: "#888780", Dark: "#B4B2A9"} // gray
	ColorError   = lipgloss.AdaptiveColor{Light: "#A32D2D", Dark: "#F09595"} // red
	ColorNull    = lipgloss.AdaptiveColor{Light: "#B4B2A9", Dark: "#5F5E5A"} // gray tenue
	ColorBorder  = lipgloss.AdaptiveColor{Light: "#D3D1C7", Dark: "#444441"} // gray borde
	ColorHeader  = lipgloss.AdaptiveColor{Light: "#4A4A4A", Dark: "#DDDDDD"} // texto
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

	// StyleFooter is the screen footer.
	StyleFooter = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// StyleNull is the NULL value marker.
	StyleNull = lipgloss.NewStyle().
			Foreground(ColorNull)

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
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	// StyleInputBoxFocus is the box of the focused input.
	StyleInputBoxFocus = StyleInputBox.
				BorderForeground(ColorAccent)

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

// NullCell returns the representation of a NULL value.
func NullCell() string { return StyleNull.Render("∅") }

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
)
