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
	layoutW := flag.Int("width", 0, "force the terminal width for --print-layout (0 = detect)")
	layoutH := flag.Int("height", 0, "force the terminal height for --print-layout (0 = detect)")
	flag.Parse()
	if *printLayout {
		os.Exit(tui.PrintLayout(*layoutW, *layoutH))
	}

	// Cell motion reports mouse movement only while a button is held, which is
	// exactly what the pane resize drag needs.
	p := tea.NewProgram(tui.New(), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "relm: %v\n", err)
		os.Exit(1)
	}
}
