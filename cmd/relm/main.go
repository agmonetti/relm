// Command relm es un browser de bases de datos para la terminal.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	_ "relm/internal/store/mssql"    // registra SQL Server
	_ "relm/internal/store/mysql"    // registra MySQL y MariaDB
	_ "relm/internal/store/postgres" // registra PostgreSQL
	_ "relm/internal/store/sqlite"   // registra SQLite
	"relm/internal/tui"
)

func main() {
	p := tea.NewProgram(tui.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "relm: %v\n", err)
		os.Exit(1)
	}
}
