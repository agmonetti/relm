package conn

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ParseDSN converts a command-line argument into a ConnectionConfig:
//
//	relm ./app.db                     SQLite (a bare path, absolute or relative)
//	relm sqlite:/abs/app.db           SQLite via URI
//	relm postgres://u:p@host:5432/db  PostgreSQL (also mysql://, mariadb://)
//	relm sqlserver://u:p@host:1433?database=db
//
// Any engine accepts ?sslmode= or ?tls= in the URL query, which maps to
// ConnectionConfig.SSLMode. Missing host/port fall back to localhost and the
// engine default port.
func ParseDSN(dsn string) (ConnectionConfig, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return ConnectionConfig{}, fmt.Errorf("empty DSN")
	}

	u, err := url.Parse(dsn)
	if err != nil {
		return ConnectionConfig{}, fmt.Errorf("invalid DSN %q: %v", dsn, err)
	}

	var cfg ConnectionConfig
	q := u.Query()
	switch strings.ToLower(u.Scheme) {
	case "sqlite", "file":
		cfg.Driver = DriverSQLite
		cfg.Path = u.Path
		if cfg.Path == "" {
			cfg.Path = u.Opaque // e.g. "sqlite:relm.db", "file:relm.db"
		}
		return cfg, nil

	case "postgres":
		cfg.Driver = DriverPostgres
	case "mysql":
		cfg.Driver = DriverMySQL
	case "mariadb":
		cfg.Driver = DriverMariaDB
	case "sqlserver":
		cfg.Driver = DriverMSSQL

	case "":
		// no scheme → a plain SQLite file path
		return ConnectionConfig{Driver: DriverSQLite, Path: dsn}, nil

	default:
		return ConnectionConfig{}, fmt.Errorf("unsupported DSN scheme %q", u.Scheme)
	}

	if u.User != nil {
		cfg.User = u.User.Username()
		cfg.Password, _ = u.User.Password()
	}
	cfg.Host = u.Hostname()
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	cfg.Port = DefaultPort(cfg.Driver)
	if p := u.Port(); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil || n <= 0 || n > 65535 {
			return ConnectionConfig{}, fmt.Errorf("invalid DSN %q: bad port %q", dsn, p)
		}
		cfg.Port = n
	}
	if cfg.Driver == DriverMSSQL {
		cfg.Database = q.Get("database")
	} else if u.Path != "" {
		cfg.Database = strings.TrimPrefix(u.Path, "/")
		if decoded, err := url.PathUnescape(cfg.Database); err == nil {
			cfg.Database = decoded
		}
	}
	if v := q.Get("sslmode"); v != "" {
		cfg.SSLMode = v
	} else if v := q.Get("tls"); v != "" {
		cfg.SSLMode = v
	}
	return cfg, nil
}
