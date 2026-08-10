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

	// StyleBorder es una línea de separación.
	StyleBorder = lipgloss.NewStyle().Foreground(ColorBorder)

	// StyleLogo es el logo ASCII de la pantalla de conexión.
	StyleLogo = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	// StyleFieldLabel es el label de un campo del formulario de conexión.
	StyleFieldLabel = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorHeader)

	// StyleInputBox es la caja visible de un input del formulario.
	StyleInputBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	// StyleInputBoxFocus es la caja del input enfocado.
	StyleInputBoxFocus = StyleInputBox.
				BorderForeground(ColorAccent)

	// StyleBtnPrimary es el botón principal (fondo sólido teal).
	StyleBtnPrimary = lipgloss.NewStyle().
				Background(ColorPrimary).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).
				Padding(0, 3)

	// StyleBtnSecondary es un botón secundario (fondo gris tenue).
	StyleBtnSecondary = lipgloss.NewStyle().
				Background(ColorMuted).
				Foreground(lipgloss.Color("#000000")).
				Padding(0, 2)
)

// NullCell devuelve la representación de un valor NULL.
func NullCell() string { return StyleNull.Render("∅") }
