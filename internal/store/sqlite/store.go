// Package sqlite implementa la interfaz store.Store para SQLite.
package sqlite

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"

	"relm/internal/conn"
	"relm/internal/store"
)

// Store es la implementación SQLite de store.Store.
type Store struct {
	db     *sql.DB
	driver conn.Driver
}

// New abre una base SQLite según la config.
func New(cfg conn.ConnectionConfig) (*Store, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("%w: falta el path del archivo", store.ErrConnection)
	}
	if _, err := os.Stat(cfg.Path); err != nil && !isMemory(cfg.Path) {
		return nil, fmt.Errorf("%w: %s: no such file", store.ErrConnection, cfg.Path)
	}
	dsn := fmt.Sprintf("file:%s?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000", cfg.Path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", store.ErrConnection, err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: %v", store.ErrConnection, err)
	}
	return &Store{db: db, driver: conn.DriverSQLite}, nil
}

func init() {
	store.Register(conn.DriverSQLite, func(cfg conn.ConnectionConfig) (store.Store, error) {
		return New(cfg)
	})
}

// isMemory reporta si el path es una base en memoria de SQLite.
func isMemory(path string) bool {
	return path == ":memory:" || path == "file::memory:"
}

func (s *Store) Driver() string { return string(s.driver) }

// Version devuelve la versión de SQLite.
func (s *Store) Version() (string, error) {
	var v string
	if err := s.db.QueryRow("SELECT sqlite_version()").Scan(&v); err != nil {
		return "", fmt.Errorf("store.Version: %w", err)
	}
	return v, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Tables devuelve las tablas de la base, ordenadas alfabéticamente.
func (s *Store) Tables() ([]string, error) {
	rows, err := s.db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("store.Tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("store.Tables: %w", err)
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

// Columns devuelve las columnas de una tabla.
func (s *Store) Columns(table string) ([]store.Column, error) {
	rows, err := s.db.Query("PRAGMA table_info(" + QuoteIdent(table) + ")")
	if err != nil {
		return nil, fmt.Errorf("store.Columns(%s): %w", table, err)
	}
	defer rows.Close()

	var cols []store.Column
	for rows.Next() {
		var (
			cid     int
			name    string
			typ     string
			notNull int
			def     sql.NullString
			pk      int
		)
		// PRAGMA table_info devuelve: cid, name, type, notnull, dflt_value, pk
		if err := rows.Scan(&cid, &name, &typ, &notNull, &def, &pk); err != nil {
			return nil, fmt.Errorf("store.Columns(%s): %w", table, err)
		}
		cols = append(cols, store.Column{
			Name:    name,
			Type:    typ,
			NotNull: notNull == 1,
			Default: def.String,
			PK:      pk > 0,
		})
	}
	return cols, rows.Err()
}

// Indexes devuelve los índices de una tabla.
func (s *Store) Indexes(table string) ([]store.Index, error) {
	rows, err := s.db.Query("PRAGMA index_list(" + QuoteIdent(table) + ")")
	if err != nil {
		return nil, fmt.Errorf("store.Indexes(%s): %w", table, err)
	}
	defer rows.Close()

	type idxMeta struct {
		name   string
		unique int
	}
	var metas []idxMeta
	for rows.Next() {
		var (
			seq     int
			name    string
			unique  int
			origin  string
			partial int
		)
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, fmt.Errorf("store.Indexes(%s): %w", table, err)
		}
		metas = append(metas, idxMeta{name: name, unique: unique})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var indexes []store.Index
	for _, m := range metas {
		cols, err := s.indexColumns(m.name)
		if err != nil {
			return nil, err
		}
		indexes = append(indexes, store.Index{
			Name:    m.name,
			Columns: cols,
			Unique:  m.unique == 1,
		})
	}
	return indexes, nil
}

func (s *Store) indexColumns(index string) ([]string, error) {
	rows, err := s.db.Query("PRAGMA index_info(" + QuoteIdent(index) + ")")
	if err != nil {
		return nil, fmt.Errorf("store.indexColumns(%s): %w", index, err)
	}
	defer rows.Close()

	var cols []string
	for rows.Next() {
		var (
			seqno int
			cid   int
			name  string
		)
		if err := rows.Scan(&seqno, &cid, &name); err != nil {
			return nil, fmt.Errorf("store.indexColumns(%s): %w", index, err)
		}
		cols = append(cols, name)
	}
	return cols, rows.Err()
}

// Query ejecuta SQL arbitrario y devuelve columnas y filas.
func (s *Store) Query(sql string) (*store.Result, error) {
	rows, err := s.db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return store.ScanResult(rows)
}

// Exec ejecuta SQL sin resultado (INSERT/UPDATE/DELETE) y devuelve filas afectadas.
func (s *Store) Exec(sql string) (int64, error) {
	res, err := s.db.Exec(sql)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return n, nil
}

// CountTable devuelve la cantidad de filas de una tabla.
func (s *Store) CountTable(table string) (int, error) {
	q := fmt.Sprintf("SELECT COUNT(*) FROM %s", QuoteIdent(table))
	var n int
	if err := s.db.QueryRow(q).Scan(&n); err != nil {
		return 0, fmt.Errorf("store.CountTable(%s): %w", table, err)
	}
	return n, nil
}

// SelectTablePage devuelve una página de filas de una tabla.
func (s *Store) SelectTablePage(table string, limit, offset int) (*store.Result, error) {
	q := fmt.Sprintf("SELECT * FROM %s %s", QuoteIdent(table), Limit(limit, offset))
	return s.Query(q)
}
