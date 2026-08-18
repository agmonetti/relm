// Command relm is a terminal database browser.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/agmonetti/relm/internal/conn"
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
	readOnly := flag.Bool("read-only", false, "open every connection read-only (all engines)")
	flag.Parse()
	if *printLayout {
		os.Exit(tui.PrintLayout(*layoutW, *layoutH))
	}
	if flag.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "relm: too many arguments — flags go before the DSN: relm --read-only ./db.sqlite")
		os.Exit(1)
	}

	// optional DSN: `relm ./db.sqlite` or `relm postgres://...` connect
	// immediately, skipping the connection screen
	opts := tui.NewOpts{GlobalReadOnly: *readOnly}
	if flag.NArg() == 1 {
		cfg, err := conn.ParseDSN(flag.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "relm: %v\n", err)
			os.Exit(1)
		}
		opts.InitialCfg = &cfg
	}

	// Cell motion reports mouse movement only while a button is held, which is
	// exactly what the pane resize drag needs.
	p := tea.NewProgram(tui.New(opts), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "relm: %v\n", err)
		os.Exit(1)
	}
}
