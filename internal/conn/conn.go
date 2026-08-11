package conn

import "fmt"

// Driver identifies one of the five supported engines.
type Driver string

// Supported engines. Do not add more.
const (
	DriverSQLite   Driver = "sqlite"
	DriverPostgres Driver = "postgres"
	DriverMySQL    Driver = "mysql"
	DriverMariaDB  Driver = "mariadb"
	DriverMSSQL    Driver = "mssql"
)

// Drivers is the closed list of supported engines.
var Drivers = []Driver{
	DriverSQLite,
	DriverPostgres,
	DriverMySQL,
	DriverMariaDB,
	DriverMSSQL,
}

// DefaultPort returns the default port for each networked engine.
func DefaultPort(d Driver) int {
	switch d {
	case DriverPostgres:
		return 5432
	case DriverMySQL, DriverMariaDB:
		return 3306
	case DriverMSSQL:
		return 1433
	}
	return 0
}

// NeedsNetwork indicates whether the engine connects over the network (not SQLite).
func NeedsNetwork(d Driver) bool {
	switch d {
	case DriverSQLite:
		return false
	}
	return true
}

// ConnectionConfig describes what we connect to, without knowing how to connect.
type ConnectionConfig struct {
	Driver Driver
	Name   string // name used to save the connection

	// SQLite
	Path string
	// Opens SQLite in read-only mode (mode=ro). Prevents accidental writes.
	ReadOnly bool

	// Network (postgres, mysql, mariadb, mssql)
	Host     string
	Port     int
	User     string
	Password string
	Database string
	// TLS mode for PostgreSQL (prefer | require | verify-full | disable).
	// Empty means the engine default.
	SSLMode string
}

// New creates a default config for a driver.
func New(d Driver) ConnectionConfig {
	return ConnectionConfig{
		Driver: d,
		Port:   DefaultPort(d),
	}
}

// Label returns a short description for the TUI header.
func (c ConnectionConfig) Label() string {
	if !NeedsNetwork(c.Driver) {
		return fmt.Sprintf("sqlite %s", c.Path)
	}
	return fmt.Sprintf("%s@%s:%d/%s", c.User, c.Host, c.Port, c.Database)
}
