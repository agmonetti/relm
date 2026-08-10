package conn

import "fmt"

// Driver identifica uno de los cinco motores soportados.
type Driver string

// Motores soportados. No agregar más.
const (
	DriverSQLite   Driver = "sqlite"
	DriverPostgres Driver = "postgres"
	DriverMySQL    Driver = "mysql"
	DriverMariaDB  Driver = "mariadb"
	DriverMSSQL    Driver = "mssql"
)

// Drivers es la lista cerrada de motores soportados.
var Drivers = []Driver{
	DriverSQLite,
	DriverPostgres,
	DriverMySQL,
	DriverMariaDB,
	DriverMSSQL,
}

// DefaultPort devuelve el puerto por defecto de cada motor de red.
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

// NeedsNetwork indica si el motor se conecta por red (no SQLite).
func NeedsNetwork(d Driver) bool {
	switch d {
	case DriverSQLite:
		return false
	}
	return true
}

// ConnectionConfig describe a qué nos conectamos, sin saber cómo conectarse.
type ConnectionConfig struct {
	Driver Driver
	Name   string // nombre para guardar la conexión

	// SQLite
	Path string
	// Abre SQLite en modo solo lectura (mode=ro). Evita escrituras accidentales.
	ReadOnly bool

	// Red (postgres, mysql, mariadb, mssql)
	Host     string
	Port     int
	User     string
	Password string
	Database string
	// SSLMode controla TLS en PostgreSQL (prefer | require | verify-full | disable).
	// Vacío = default del motor.
	SSLMode string
}

// New crea una config por defecto para un driver.
func New(d Driver) ConnectionConfig {
	return ConnectionConfig{
		Driver: d,
		Port:   DefaultPort(d),
	}
}

// Label devuelve una descripción corta para el header de la TUI.
func (c ConnectionConfig) Label() string {
	if !NeedsNetwork(c.Driver) {
		return fmt.Sprintf("sqlite %s", c.Path)
	}
	return fmt.Sprintf("%s@%s:%d/%s", c.User, c.Host, c.Port, c.Database)
}
