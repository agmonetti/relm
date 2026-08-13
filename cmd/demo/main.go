// Command demo creates the example dataset to try relm without any setup:
//
//	go run ./cmd/demo                 # SQLite demo.db (no server needed)
//	go run ./cmd/demo --all           # every engine, incl. the 4 network ones
//	go run ./cmd/demo --postgres      # one engine
//
// For the network engines a server must already be running; the repo's
// compose.yaml starts all four (docker compose up -d) with the credentials the
// command uses by default. Credentials can be overridden with environment
// variables (POSTGRES_HOST, MYSQL_PASSWORD, MSSQL_HOST, ...).
//
// The dataset is the same 20 tables with a few thousand rows each (users,
// orders, products, payments, logs, ...), seeded deterministically, so
// pagination and the browser can be exercised with a real amount of data on
// every engine.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"  // MySQL and MariaDB
	_ "github.com/jackc/pgx/v5/stdlib"  // PostgreSQL
	_ "github.com/microsoft/go-mssqldb" // SQL Server
	_ "modernc.org/sqlite"              // SQLite (pure Go)

	"github.com/agmonetti/relm/internal/demo"
)

// default configs match the repo's compose.yaml.
var defaults = map[string]demo.Config{
	"postgres": {Host: "localhost", Port: 5432, User: "postgres", Password: "postgres", Database: "test"},
	"mysql":    {Host: "localhost", Port: 3306, User: "root", Password: "root", Database: "test"},
	"mariadb":  {Host: "localhost", Port: 3307, User: "root", Password: "root", Database: "test"},
	"mssql":    {Host: "localhost", Port: 1433, User: "sa", Password: "Str0ng!Passw0rd", Database: "master"},
}

// allEngines is the list in the same order as relm's engine selector.
var allEngines = []string{"sqlite", "postgres", "mysql", "mariadb", "mssql"}

func main() {
	var sqlite, postgres, mysql, mariadb, mssql bool
	var all bool
	flag.BoolVar(&all, "all", false, "seed every engine")
	flag.BoolVar(&sqlite, "sqlite", false, "seed the SQLite demo.db")
	flag.BoolVar(&postgres, "postgres", false, "seed PostgreSQL")
	flag.BoolVar(&mysql, "mysql", false, "seed MySQL")
	flag.BoolVar(&mariadb, "mariadb", false, "seed MariaDB")
	flag.BoolVar(&mssql, "mssql", false, "seed SQL Server")
	flag.Parse()

	engines := map[string]bool{
		"sqlite": sqlite, "postgres": postgres, "mysql": mysql,
		"mariadb": mariadb, "mssql": mssql,
	}
	if all {
		for _, e := range allEngines {
			engines[e] = true
		}
	}
	if !any(engines) {
		engines["sqlite"] = true // default: no server needed
	}

	fail := 0
	for _, e := range allEngines {
		if !engines[e] {
			continue
		}
		if err := seed(e); err != nil {
			fmt.Fprintf(os.Stderr, "demo %-8s: %v\n", e, err)
			fail++
			continue
		}
		fmt.Printf("Example database created: %s\n", hint(e))
	}
	if fail > 0 {
		os.Exit(1)
	}
}

func any(m map[string]bool) bool {
	for _, v := range m {
		if v {
			return true
		}
	}
	return false
}

func hint(e string) string {
	switch e {
	case "sqlite":
		return "open it with relm (engine SQLite, path demo.db)"
	case "postgres":
		return "open it with relm (PostgreSQL, localhost:5432, user postgres, database test)"
	case "mysql":
		return "open it with relm (MySQL, localhost:3306, user root, database test)"
	case "mariadb":
		return "open it with relm (MariaDB, localhost:3307, user root, database test)"
	case "mssql":
		return "open it with relm (SQL Server, localhost:1433, user sa, database master)"
	}
	return ""
}

func seed(engine string) error {
	cfg := cfgFor(engine)
	if engine == "sqlite" {
		path := cfg.Path
		if path == "" {
			path = "demo.db"
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	db, err := sql.Open(demo.DriverName(engine), demo.DSN(engine, cfg))
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("cannot reach the server (%s): %w", demo.DSN(engine, redact(cfg)), err)
	}

	start := time.Now()
	if err := demo.Seed(db, engine); err != nil {
		return err
	}
	fmt.Printf("  %s: 20 tables seeded in %s\n", engine, time.Since(start).Round(time.Millisecond))
	return nil
}

// cfgFor builds the demo config from the environment, falling back to the
// compose.yaml credentials. SQLite reads the optional first argument as the
// path, or writes demo.db.
func cfgFor(engine string) demo.Config {
	cfg := defaults[engine]
	getenv := func(name, def string) string {
		if v := os.Getenv(name); v != "" {
			return v
		}
		return def
	}

	switch engine {
	case "sqlite":
		path := "demo.db"
		if len(os.Args) > 1 && !isFlag(os.Args[1]) {
			path = os.Args[1]
		}
		if p := os.Getenv("DEMO_SQLITE_PATH"); p != "" {
			path = p
		}
		return demo.Config{Path: path}
	case "postgres":
		cfg.Host = getenv("POSTGRES_HOST", cfg.Host)
		cfg.Port = intEnv("POSTGRES_PORT", cfg.Port)
		cfg.User = getenv("POSTGRES_USER", cfg.User)
		cfg.Password = getenv("POSTGRES_PASSWORD", cfg.Password)
		cfg.Database = getenv("POSTGRES_DATABASE", cfg.Database)
	case "mysql":
		cfg.Host = getenv("MYSQL_HOST", cfg.Host)
		cfg.Port = intEnv("MYSQL_PORT", cfg.Port)
		cfg.User = getenv("MYSQL_USER", cfg.User)
		cfg.Password = getenv("MYSQL_PASSWORD", cfg.Password)
		cfg.Database = getenv("MYSQL_DATABASE", cfg.Database)
	case "mariadb":
		cfg.Host = getenv("MARIADB_HOST", cfg.Host)
		cfg.Port = intEnv("MARIADB_PORT", cfg.Port)
		cfg.User = getenv("MARIADB_USER", cfg.User)
		cfg.Password = getenv("MARIADB_PASSWORD", cfg.Password)
		cfg.Database = getenv("MARIADB_DATABASE", cfg.Database)
	case "mssql":
		cfg.Host = getenv("MSSQL_HOST", cfg.Host)
		cfg.Port = intEnv("MSSQL_PORT", cfg.Port)
		cfg.User = getenv("MSSQL_USER", cfg.User)
		cfg.Password = getenv("MSSQL_PASSWORD", cfg.Password)
		cfg.Database = getenv("MSSQL_DATABASE", cfg.Database)
	}
	return cfg
}

func intEnv(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func isFlag(s string) bool {
	return len(s) > 1 && s[0] == '-'
}

// redact hides the password from the DSN shown in error messages.
func redact(cfg demo.Config) demo.Config {
	cfg.Password = "xxxx"
	return cfg
}
