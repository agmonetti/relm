// Command relm is a terminal database browser.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	_ "github.com/agmonetti/relm/internal/store/mssql"    // registers SQL Server
	_ "github.com/agmonetti/relm/internal/store/mysql"    // registers MySQL and MariaDB
	_ "github.com/agmonetti/relm/internal/store/postgres" // registers PostgreSQL
	_ "github.com/agmonetti/relm/internal/store/sqlite"   // registers SQLite
	"github.com/agmonetti/relm/internal/tui"
)

func main() {
	printLayout := flag.Bool("print-layout", false, "print the layout as text and exit (debug)")
	flag.Parse()
	if *printLayout {
		os.Exit(tui.PrintLayout())
	}

	p := tea.NewProgram(tui.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "relm: %v\n", err)
		os.Exit(1)
	}
}
