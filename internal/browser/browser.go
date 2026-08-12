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

	// keyset pagination state (only when the active table has a single-column
	// primary key); otherwise the browser falls back to OFFSET pagination
	orderBy string
	keyIdx  int
	cur     []string // cur[p] = key of the last row of page p-1; "" for page 0
	hasNext bool
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

	b.orderBy, b.keyIdx = keysetKey(cols)
	b.cur = []string{""}
	b.hasNext = false
	return b.Refresh(st)
}

// keysetKey returns the single-column primary key for keyset pagination, or an
// empty key when the table has no PK or a composite PK (OFTSET fallback).
func keysetKey(cols []store.Column) (string, int) {
	var pk []int
	for i, c := range cols {
		if c.PK {
			pk = append(pk, i)
		}
	}
	if len(pk) == 1 {
		return cols[pk[0]].Name, pk[0]
	}
	return "", -1
}

// Refresh reloads the current page of the active table. In keyset mode the
// page is re-fetched from its stored cursor, so refreshing does not move the
// visible rows.
func (b *Browser) Refresh(st store.Store) error {
	if b.ActiveTable == "" {
		return nil
	}
	total, err := st.CountTable(b.ActiveTable)
	if err != nil {
		return err
	}
	b.TotalRows = total

	if b.orderBy == "" {
		res, err := st.SelectTablePage(b.ActiveTable, b.PageSize, b.Page*b.PageSize)
		if err != nil {
			return err
		}
		b.Rows = res.Rows
		b.clampCursor()
		return nil
	}

	// keyset: fetch the page after cur[Page], asking for one extra row to
	// know whether another page follows
	res, err := st.SelectTableKeysetPage(b.ActiveTable, b.orderBy, b.PageSize+1, b.cur[b.Page])
	if err != nil {
		return err
	}
	b.hasNext = len(res.Rows) > b.PageSize
	rows := res.Rows
	if len(rows) > b.PageSize {
		rows = rows[:b.PageSize]
	}
	b.Rows = rows
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
	if err := b.refreshMeta(st); err != nil {
		return err
	}
	return b.Refresh(st)
}

// refreshMeta reloads the columns and indexes of the active table and
// recomputes the keyset state, so DDL from the editor (drop/recreate, alter)
// does not leave a stale ordering key or column list.
func (b *Browser) refreshMeta(st store.Store) error {
	cols, err := st.Columns(b.ActiveTable)
	if err != nil {
		return err
	}
	b.Columns = cols

	indexes, err := st.Indexes(b.ActiveTable)
	if err != nil {
		return err
	}
	b.Indexes = indexes

	orderBy, keyIdx := keysetKey(cols)
	if orderBy != b.orderBy {
		// the ordering key changed or disappeared: restart the navigation
		b.Page = 0
		b.Cursor = 0
		b.cur = []string{""}
		b.hasNext = false
	}
	b.orderBy = orderBy
	b.keyIdx = keyIdx
	return nil
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
	if b.orderBy == "" {
		b.Page++
		b.Cursor = 0
		return b.Refresh(st)
	}
	last, ok := b.lastKey()
	if !ok {
		return nil
	}
	b.cur = append(b.cur, last)
	b.Page++
	b.Cursor = 0
	return b.Refresh(st)
}

// PrevPage goes back to the previous page if it exists.
func (b *Browser) PrevPage(st store.Store) error {
	if !b.HasPrevPage() {
		return nil
	}
	if b.orderBy == "" {
		b.Page--
		b.Cursor = 0
		return b.Refresh(st)
	}
	// re-fetching forward from the previous page's cursor and trimming to the
	// page size yields exactly the previous page (it is full by construction)
	b.cur = b.cur[:b.Page]
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
	if b.orderBy != "" {
		return b.hasNext
	}
	return (b.Page+1)*b.PageSize < b.TotalRows
}

// HasPrevPage reports whether there are pages before the current one.
func (b *Browser) HasPrevPage() bool {
	return b.Page > 0
}

// lastKey returns the key value of the last row of the current page.
func (b *Browser) lastKey() (string, bool) {
	if len(b.Rows) == 0 || b.keyIdx < 0 {
		return "", false
	}
	return b.Rows[len(b.Rows)-1][b.keyIdx], true
}

func (b *Browser) clampCursor() {
	if b.Cursor < 0 {
		b.Cursor = 0
	}
	if max := len(b.Rows) - 1; b.Cursor > max {
		b.Cursor = max
	}
}
