// Package store define la interfaz Store, el contrato que implementa cada motor.
// Ninguna otra capa importa database/sql ni drivers directamente.
package store

import (
	"fmt"
	"sync"

	"relm/internal/conn"
)

// Column describe una columna de una tabla.
type Column struct {
	Name    string
	Type    string
	NotNull bool
	Default string
	PK      bool
}

// Result es el resultado de una query. Toda celda se convierte a string en el
// store; la UI nunca hace type assertions.
type Result struct {
	Columns  []string
	Rows     [][]string
	Count    int   // cantidad de filas del resultado (no paginado)
	Affected int64 // filas afectadas por un Exec; -1 si no aplica (query de lectura)
}

// Index describe un índice de una tabla.
type Index struct {
	Name    string
	Columns []string
	Unique  bool
}

// Store es la capa de acceso a datos, neutral respecto al motor.
type Store interface {
	// Conexión
	Driver() string
	Version() (string, error)
	Close() error

	// Introspección de esquema
	Tables() ([]string, error)
	Columns(table string) ([]Column, error)
	Indexes(table string) ([]Index, error)

	// Ejecución arbitraria
	Query(sql string) (*Result, error)
	Exec(sql string) (int64, error)

	// Dialecto: paginación y conteo generados por el motor
	CountTable(table string) (int, error)
	SelectTablePage(table string, limit, offset int) (*Result, error)
}

// Constructor es la función que crea un Store para un motor.
type Constructor func(conn.ConnectionConfig) (Store, error)

var (
	registryMu sync.RWMutex
	registry   = map[conn.Driver]Constructor{}
)

// Register registra un constructor de motor. Se llama desde el init() de cada
// paquete de motor. Es configuración en tiempo de compilación, no estado de runtime.
func Register(d conn.Driver, c Constructor) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[d] = c
}

// New crea un Store según la config. El motor debe haber sido registrado.
func New(cfg conn.ConnectionConfig) (Store, error) {
	registryMu.RLock()
	c, ok := registry[cfg.Driver]
	registryMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedDriver, cfg.Driver)
	}
	return c(cfg)
}
