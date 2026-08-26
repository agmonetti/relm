package conn

import (
	"fmt"
	"strings"
)

// Driver identifies one of the supported database engines.
type Driver string

// Supported engines across relational, document, key-value, wide-column, and graph paradigms.
const (
	DriverSQLite    Driver = "sqlite"
	DriverPostgres  Driver = "postgres"
	DriverMySQL     Driver = "mysql"
	DriverMariaDB   Driver = "mariadb"
	DriverMSSQL     Driver = "mssql"
	DriverMongo     Driver = "mongodb"
	DriverRedis     Driver = "redis"
	DriverCassandra Driver = "cassandra"
	DriverNeo4j     Driver = "neo4j"
)

// Drivers is the list of supported engines.
var Drivers = []Driver{
	DriverSQLite,
	DriverPostgres,
	DriverMySQL,
	DriverMariaDB,
	DriverMSSQL,
	DriverMongo,
	DriverRedis,
	DriverCassandra,
	DriverNeo4j,
}

// DefaultPort returns the default network port for each engine.
func DefaultPort(d Driver) int {
	switch d {
	case DriverPostgres:
		return 5432
	case DriverMySQL, DriverMariaDB:
		return 3306
	case DriverMSSQL:
		return 1433
	case DriverMongo:
		return 27017
	case DriverRedis:
		return 6379
	case DriverCassandra:
		return 9042
	case DriverNeo4j:
		return 7687
	}
	return 0
}

// NeedsNetwork indicates whether the engine connects over the network.
func NeedsNetwork(d Driver) bool {
	switch d {
	case DriverSQLite:
		return false
	}
	return true
}

// ConnectionConfig describes what we connect to, without knowing how to connect.
type ConnectionConfig struct {
	Driver Driver `json:"driver"`
	Name   string `json:"name"` // name used to save the connection

	// SQLite
	Path string `json:"path,omitempty"`
	// Opens database in read-only mode.
	ReadOnly bool `json:"read_only"`

	// Direct URI (optional; standard for MongoDB, Neo4j, Redis)
	URI string `json:"uri,omitempty"`

	// Network coordinates
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	Database string `json:"database,omitempty"` // Database name, Keyspace, or Redis DB index
	// TLS mode (prefer | require | verify-full | disable).
	SSLMode string `json:"ssl_mode,omitempty"`
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
	if c.URI != "" {
		return fmt.Sprintf("%s %s", c.Driver, redactURI(c.URI))
	}
	switch c.Driver {
	case DriverRedis:
		db := c.Database
		if db == "" {
			db = "0"
		}
		return fmt.Sprintf("redis %s:%d/%s", c.Host, c.Port, db)
	case DriverMongo:
		if c.User != "" {
			return fmt.Sprintf("mongodb %s@%s:%d/%s", c.User, c.Host, c.Port, c.Database)
		}
		return fmt.Sprintf("mongodb %s:%d/%s", c.Host, c.Port, c.Database)
	case DriverCassandra:
		return fmt.Sprintf("cassandra %s:%d/%s", c.Host, c.Port, c.Database)
	case DriverNeo4j:
		user := c.User
		if user == "" {
			user = "neo4j"
		}
		return fmt.Sprintf("neo4j %s@%s:%d/%s", user, c.Host, c.Port, c.Database)
	default:
		if c.User != "" {
			return fmt.Sprintf("%s@%s:%d/%s", c.User, c.Host, c.Port, c.Database)
		}
		return fmt.Sprintf("%s:%d/%s", c.Host, c.Port, c.Database)
	}
}

func redactURI(uri string) string {
	// hide passwords in URI if present
	if idx := strings.Index(uri, "://"); idx != -1 {
		prefix := uri[:idx+3]
		rest := uri[idx+3:]
		if at := strings.Index(rest, "@"); at != -1 {
			userinfo := rest[:at]
			if colon := strings.Index(userinfo, ":"); colon != -1 {
				userinfo = userinfo[:colon] + ":••••"
			}
			return prefix + userinfo + rest[at:]
		}
	}
	return uri
}
