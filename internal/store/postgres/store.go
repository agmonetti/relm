package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // registra el driver "pgx"

	"github.com/agmonetti/relm/internal/conn"
	"github.com/agmonetti/relm/internal/store"
)

// Store es la implementación PostgreSQL de store.Store.
type Store struct {
	db *sql.DB
}

// New abre una conexión a PostgreSQL.
func New(cfg conn.ConnectionConfig) (*Store, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("%w: falta el host", store.ErrConnection)
	}
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:   "/" + url.PathEscape(cfg.Database),
	}
	q := u.Query()
	sslmode := cfg.SSLMode
	if sslmode == "" {
		sslmode = "prefer"
	}
	q.Set("sslmode", sslmode)
	u.RawQuery = q.Encode()

	db, err := sql.Open("pgx", u.String())
	if err != nil {
		return nil, fmt.Errorf("%w: %v", store.ErrConnection, err)
	}
	db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: %v", store.ErrConnection, err)
	}
	return &Store{db: db}, nil
}

func init() {
	store.Register(conn.DriverPostgres, func(cfg conn.ConnectionConfig) (store.Store, error) {
		return New(cfg)
	})
}

func (s *Store) Driver() string { return "postgres" }

// Version devuelve la versión del servidor.
func (s *Store) Version() (string, error) {
	var v string
	if err := s.db.QueryRow("SELECT version()").Scan(&v); err != nil {
		return "", fmt.Errorf("store.Version: %w", err)
	}
	return v, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Tables devuelve las tablas del schema actual (search_path).
func (s *Store) Tables() ([]string, error) {
	rows, err := s.db.Query(`
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = current_schema() AND table_type = 'BASE TABLE'
		ORDER BY table_name`)
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

// Columns devuelve las columnas de una tabla del schema actual.
func (s *Store) Columns(table string) ([]store.Column, error) {
	rows, err := s.db.Query(`
		SELECT c.column_name, c.data_type,
		       (c.is_nullable = 'NO'),
		       COALESCE(c.column_default, ''),
		       CASE WHEN pk.column_name IS NOT NULL THEN true ELSE false END
		FROM information_schema.columns c
		LEFT JOIN (
			SELECT ku.table_name, ku.column_name
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage ku
			  ON tc.constraint_name = ku.constraint_name
			 AND tc.table_schema = ku.table_schema
			WHERE tc.constraint_type = 'PRIMARY KEY' AND tc.table_schema = current_schema()
		) pk ON c.table_name = pk.table_name AND c.column_name = pk.column_name
		WHERE c.table_schema = current_schema() AND c.table_name = $1
		ORDER BY c.ordinal_position`, table)
	if err != nil {
		return nil, fmt.Errorf("store.Columns(%s): %w", table, err)
	}
	defer rows.Close()

	var cols []store.Column
	for rows.Next() {
		var c store.Column
		if err := rows.Scan(&c.Name, &c.Type, &c.NotNull, &c.Default, &c.PK); err != nil {
			return nil, fmt.Errorf("store.Columns(%s): %w", table, err)
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

// Indexes devuelve los índices de una tabla del schema actual.
func (s *Store) Indexes(table string) ([]store.Index, error) {
	rows, err := s.db.Query(`
		SELECT i.relname, a.attname, ix.indisunique
		FROM pg_class t
		JOIN pg_index ix ON t.oid = ix.indrelid
		JOIN pg_class i ON i.oid = ix.indexrelid
		JOIN pg_namespace n ON t.relnamespace = n.oid
		JOIN unnest(ix.indkey) WITH ORDINALITY AS k(attnum, ord) ON true
		JOIN pg_attribute a ON t.oid = a.attrelid AND a.attnum = k.attnum
		WHERE n.nspname = current_schema() AND t.relname = $1
		ORDER BY i.relname, k.ord`, table)
	if err != nil {
		return nil, fmt.Errorf("store.Indexes(%s): %w", table, err)
	}
	defer rows.Close()

	var indexes []store.Index
	byName := map[string]*store.Index{}
	var order []string
	for rows.Next() {
		var name, col string
		var unique bool
		if err := rows.Scan(&name, &col, &unique); err != nil {
			return nil, fmt.Errorf("store.Indexes(%s): %w", table, err)
		}
		ix, ok := byName[name]
		if !ok {
			ix = &store.Index{Name: name, Unique: unique}
			byName[name] = ix
			order = append(order, name)
		}
		ix.Columns = append(ix.Columns, col)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, name := range order {
		indexes = append(indexes, *byName[name])
	}
	return indexes, nil
}

// Query ejecuta SQL arbitrario.
func (s *Store) Query(sql string) (*store.Result, error) {
	rows, err := s.db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return store.ScanResult(rows)
}

// Exec ejecuta SQL sin resultado.
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
