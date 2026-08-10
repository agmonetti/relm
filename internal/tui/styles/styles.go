// Package styles centraliza todos los lipgloss.Style del proyecto.
package styles

import "github.com/charmbracelet/lipgloss"

// Colores adaptables al tema del terminal.
var (
	ColorPrimary = lipgloss.AdaptiveColor{Light: "#1D9E75", Dark: "#5DCAA5"} // teal
	ColorAccent  = lipgloss.AdaptiveColor{Light: "#534AB7", Dark: "#AFA9EC"} // purple
	ColorMuted   = lipgloss.AdaptiveColor{Light: "#888780", Dark: "#B4B2A9"} // gray
	ColorError   = lipgloss.AdaptiveColor{Light: "#A32D2D", Dark: "#F09595"} // red
	ColorNull    = lipgloss.AdaptiveColor{Light: "#B4B2A9", Dark: "#5F5E5A"} // gray tenue
	ColorBorder  = lipgloss.AdaptiveColor{Light: "#D3D1C7", Dark: "#444441"} // gray borde
	ColorHeader  = lipgloss.AdaptiveColor{Light: "#4A4A4A", Dark: "#DDDDDD"} // texto
)

// Estilos precalculados a nivel package. No crear estilos en cada View().
var (
	// StyleHeader es el texto del header.
	StyleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	// StyleHeaderDim es la parte tenue del header.
	StyleHeaderDim = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// StyleSidebarActive es el cursor > de la tabla activa.
	StyleSidebarActive = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	// StyleSidebarItem es una tabla normal del sidebar.
	StyleSidebarItem = lipgloss.NewStyle().Foreground(ColorHeader)

	// StyleError es el texto de error.
	StyleError = lipgloss.NewStyle().
			Foreground(ColorError)

	// StyleFooter es el pie de pantalla.
	StyleFooter = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// StyleNull es el marcador de valores NULL.
	StyleNull = lipgloss.NewStyle().
			Foreground(ColorNull)

	// StyleCursor es la fila seleccionada en el browser.
	StyleCursor = lipgloss.NewStyle().
			Background(ColorAccent).
			Foreground(lipgloss.Color("#FFFFFF"))

	// StyleColHeader es el encabezado de columna.
	StyleColHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent)

	// StyleAccentInput resalta el input enfocado.
	StyleAccentInput = lipgloss.NewStyle().
			Foreground(ColorAccent)

	// StyleBordered es el contenedor con borde.
	StyleBordered = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder)

	// StyleBorder es una línea de separación.
	StyleBorder = lipgloss.NewStyle().Foreground(ColorBorder)
)

// NullCell devuelve la representación de un valor NULL.
func NullCell() string { return StyleNull.Render("∅") }
