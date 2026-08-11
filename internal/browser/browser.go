// Package browser manages the state for navigating tables and rows,
// independent of the TUI and the engine.
package browser

import (
	"sort"

	"github.com/agmonetti/relm/internal/store"
)

// PageSizeDefault is the number of rows per page.
const PageSizeDefault = 50

// Browser keeps the navigation state of the active database.
type Browser struct {
	Tables      []string
	ActiveTable string
	Columns     []store.Column
	Indexes     []store.Index
	Page        int
	PageSize    int
	TotalRows   int
	Rows        [][]string
	Cursor      int
}

// New loads the tables of the database and selects the first one.
func New(st store.Store) (*Browser, error) {
	b := &Browser{PageSize: PageSizeDefault}
	if err := b.Load(st); err != nil {
		return nil, err
	}
	if len(b.Tables) > 0 {
		if err := b.SelectTable(b.Tables[0], st); err != nil {
			return nil, err
		}
	}
	return b, nil
}

// Load reloads the list of tables.
func (b *Browser) Load(st store.Store) error {
	tables, err := st.Tables()
	if err != nil {
		return err
	}
	sort.Strings(tables)
	b.Tables = tables
	return nil
}

// SelectTable switches the active table, resets the page and loads its data.
func (b *Browser) SelectTable(name string, st store.Store) error {
	b.ActiveTable = name
	b.Page = 0
	b.Cursor = 0

	cols, err := st.Columns(name)
	if err != nil {
		return err
	}
	b.Columns = cols

	indexes, err := st.Indexes(name)
	if err != nil {
		return err
	}
	b.Indexes = indexes
	return b.Refresh(st)
}

// Refresh reloads the current page of the active table.
func (b *Browser) Refresh(st store.Store) error {
	if b.ActiveTable == "" {
		return nil
	}
	total, err := st.CountTable(b.ActiveTable)
	if err != nil {
		return err
	}
	b.TotalRows = total

	res, err := st.SelectTablePage(b.ActiveTable, b.PageSize, b.Page*b.PageSize)
	if err != nil {
		return err
	}
	b.Rows = res.Rows
	b.clampCursor()
	return nil
}

// Reload reloads the table list and the active table data. If the database
// had no active table (e.g. you created a table from the editor) or the active
// one no longer exists, it selects the first. Covers tables created/dropped
// externally or from the editor.
func (b *Browser) Reload(st store.Store) error {
	if err := b.Load(st); err != nil {
		return err
	}
	if b.ActiveTable == "" || !hasString(b.Tables, b.ActiveTable) {
		if len(b.Tables) > 0 {
			return b.SelectTable(b.Tables[0], st)
		}
		b.ActiveTable = ""
		b.Columns = nil
		b.Indexes = nil
		b.Rows = nil
		b.TotalRows = 0
		b.Page = 0
		b.Cursor = 0
		return nil
	}
	return b.Refresh(st)
}

func hasString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

// NextPage advances to the next page if it exists.
func (b *Browser) NextPage(st store.Store) error {
	if !b.HasNextPage() {
		return nil
	}
	b.Page++
	b.Cursor = 0
	return b.Refresh(st)
}

// PrevPage goes back to the previous page if it exists.
func (b *Browser) PrevPage(st store.Store) error {
	if !b.HasPrevPage() {
		return nil
	}
	b.Page--
	b.Cursor = 0
	return b.Refresh(st)
}

// MoveCursor moves the selected row within the visible page.
func (b *Browser) MoveCursor(delta int) {
	b.Cursor += delta
	b.clampCursor()
}

// HasNextPage reports whether there are more pages after the current one.
func (b *Browser) HasNextPage() bool {
	return (b.Page+1)*b.PageSize < b.TotalRows
}

// HasPrevPage reports whether there are pages before the current one.
func (b *Browser) HasPrevPage() bool {
	return b.Page > 0
}

func (b *Browser) clampCursor() {
	if b.Cursor < 0 {
		b.Cursor = 0
	}
	if max := len(b.Rows) - 1; b.Cursor > max {
		b.Cursor = max
	}
}
