package tui

import (
	"fmt"
	"os"

	term "github.com/charmbracelet/x/term"

	"github.com/agmonetti/relm/internal/browser"
	"github.com/agmonetti/relm/internal/editor"
	"github.com/agmonetti/relm/internal/store"
	"github.com/agmonetti/relm/internal/tui/screens"
)

// PrintLayout renders the connection screen and a sample workspace as plain
// text (no TUI) together with the terminal size the app would use. It is meant
// to diagnose layout problems on any terminal: `relm --print-layout`.
func PrintLayout() int {
	w, h, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		w, h = 120, 30
	}
	w, h = correctSize(w, h)

	fmt.Printf("terminal size: %dx%d (after size guard)\n", w, h)

	fmt.Println("\n--- connect screen ---")
	fmt.Println(screens.NewConnScreen(nil).View(w, h))

	fmt.Println("\n--- workspace ---")
	layout := screens.ComputeLayout(w, h-2, true, 0, 0)
	fmt.Println(screens.RenderWorkspace(sampleBrowser(), screens.NewEditorScreen(),
		sampleEditor(), screens.FocusSidebar, false, layout, 0, w, h-2))

	fmt.Println("\n--- workspace (no sidebar, small terminal) ---")
	layout = screens.ComputeLayout(50, h-2, true, 0, 0)
	fmt.Println(screens.RenderWorkspace(sampleBrowser(), screens.NewEditorScreen(),
		sampleEditor(), screens.FocusSidebar, false, layout, 0, 50, h-2))

	return 0
}

// sampleBrowser builds a synthetic browser with a few tables and rows so the
// workspace can be rendered without a real database.
func sampleBrowser() *browser.Browser {
	return &browser.Browser{
		Tables:      []string{"orders", "sessions", "users"},
		ActiveTable: "users",
		Columns: []store.Column{
			{Name: "id", Type: "INTEGER", PK: true, NotNull: true},
			{Name: "name", Type: "TEXT", NotNull: true},
			{Name: "email", Type: "TEXT"},
		},
		Rows:      [][]string{{"1", "Alice", "alice@test.com"}, {"2", "Bob", "bob@test.com"}},
		Indexes:   []store.Index{{Name: "idx_email", Columns: []string{"email"}, Unique: true}},
		TotalRows: 2,
		PageSize:  browser.PageSizeDefault,
	}
}

func sampleEditor() *editor.Editor {
	return &editor.Editor{Buffer: "SELECT * FROM users LIMIT 10", History: editor.NewHistory()}
}
