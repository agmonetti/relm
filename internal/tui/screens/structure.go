package screens

import (
	"fmt"
	"strings"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/tui/styles"
)

// RenderStructure renderiza columnas e índices de la tabla activa.
func RenderStructure(b *browser.Browser, width, height int) string {
	var sb strings.Builder

	sb.WriteString(styles.StyleHeader.Render("Columnas") + "\n")
	for _, c := range b.Columns {
		var flags []string
		if c.PK {
			flags = append(flags, "PK")
		}
		if c.NotNull {
			flags = append(flags, "NN")
		}
		def := ""
		if c.Default != "" {
			def = fmt.Sprintf(" DEF %s", c.Default)
		}
		line := fmt.Sprintf("  %-24s %-12s %s%s", c.Name, c.Type, strings.Join(flags, " "), def)
		sb.WriteString(styles.StyleSidebarItem.Render(line) + "\n")
	}

	indexes := b.Indexes
	sb.WriteString("\n" + styles.StyleHeader.Render("Índices") + "\n")
	if len(indexes) == 0 {
		sb.WriteString(styles.StyleHeaderDim.Render("  sin índices") + "\n")
	}
	for _, ix := range indexes {
		uniq := ""
		if ix.Unique {
			uniq = " UNIQUE"
		}
		line := fmt.Sprintf("  %-24s (%s)%s", ix.Name, strings.Join(ix.Columns, ", "), uniq)
		sb.WriteString(styles.StyleSidebarItem.Render(line) + "\n")
	}
	return sb.String()
}
