// Package browser maneja el estado de navegación de tablas y filas,
// independiente de la TUI y del motor.
package browser

import (
	"sort"

	"relm/internal/store"
)

// PageSizeDefault es la cantidad de filas por página.
const PageSizeDefault = 50

// Browser mantiene el estado de navegación de la base activa.
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

// New carga las tablas de la base y selecciona la primera.
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

// Load recarga la lista de tablas.
func (b *Browser) Load(st store.Store) error {
	tables, err := st.Tables()
	if err != nil {
		return err
	}
	sort.Strings(tables)
	b.Tables = tables
	return nil
}

// SelectTable cambia la tabla activa, resetea la página y carga los datos.
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

// Refresh recarga la página actual de la tabla activa.
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

// NextPage avanza a la página siguiente si existe.
func (b *Browser) NextPage(st store.Store) error {
	if !b.HasNextPage() {
		return nil
	}
	b.Page++
	b.Cursor = 0
	return b.Refresh(st)
}

// PrevPage retrocede a la página anterior si existe.
func (b *Browser) PrevPage(st store.Store) error {
	if !b.HasPrevPage() {
		return nil
	}
	b.Page--
	b.Cursor = 0
	return b.Refresh(st)
}

// MoveCursor mueve la fila seleccionada dentro de la página visible.
func (b *Browser) MoveCursor(delta int) {
	b.Cursor += delta
	b.clampCursor()
}

// HasNextPage indica si hay más páginas después de la actual.
func (b *Browser) HasNextPage() bool {
	return (b.Page+1)*b.PageSize < b.TotalRows
}

// HasPrevPage indica si hay páginas anteriores a la actual.
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
