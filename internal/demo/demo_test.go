package demo

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestSeedSQLite(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)

	if err := Seed(db, "sqlite"); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	count := func(table string) int {
		var n int
		if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		return n
	}
	for table, want := range map[string]int{
		"users": 2000, "customers": 2500, "orders": 6000, "logs": 30000, "sessions": 12000,
	} {
		if got := count(table); got != want {
			t.Errorf("%s = %d, want %d", table, got, want)
		}
	}

	// deterministic: seeding again over the same tables yields the same counts
	if err := Seed(db, "sqlite"); err != nil {
		t.Fatalf("reseed: %v", err)
	}
	for table, want := range map[string]int{
		"users": 2000, "orders": 6000, "logs": 30000,
	} {
		if got := count(table); got != want {
			t.Errorf("reseed %s = %d, want %d", table, got, want)
		}
	}
}

func TestDSN(t *testing.T) {
	cfg := Config{Host: "localhost", Port: 5432, User: "u", Password: "p", Database: "db"}
	if got := DSN("postgres", cfg); got != "postgres://u:p@localhost:5432/db?sslmode=disable" {
		t.Errorf("postgres DSN = %q", got)
	}
	if got := DSN("mysql", cfg); got != "u:p@tcp(localhost:5432)/db" {
		t.Errorf("mysql DSN = %q", got)
	}
	if got := DSN("sqlite", Config{Path: "demo.db"}); got != "demo.db" {
		t.Errorf("sqlite DSN = %q", got)
	}
	if got := DSN("mssql", cfg); got != "sqlserver://u:p@localhost:5432?database=db" {
		t.Errorf("mssql DSN = %q", got)
	}
}

func TestSchemaTranslatesPerEngine(t *testing.T) {
	for _, driver := range []string{"sqlite", "postgres", "mysql", "mariadb", "mssql"} {
		stmts := Schema(driver)
		if len(stmts) != 20 {
			t.Errorf("%s: %d tables, want 20", driver, len(stmts))
		}
		// every table must have an autoincrement id
		joined := ""
		for _, s := range stmts {
			joined += s + "\n"
		}
		switch driver {
		case "postgres":
			if !contains(joined, "BIGSERIAL PRIMARY KEY") {
				t.Errorf("postgres: missing BIGSERIAL id")
			}
		case "mysql", "mariadb":
			if !contains(joined, "AUTO_INCREMENT PRIMARY KEY") {
				t.Errorf("%s: missing AUTO_INCREMENT id", driver)
			}
		case "mssql":
			if !contains(joined, "IDENTITY(1,1) PRIMARY KEY") {
				t.Errorf("mssql: missing IDENTITY id")
			}
		default:
			if !contains(joined, "INTEGER PRIMARY KEY") {
				t.Errorf("sqlite: missing INTEGER PRIMARY KEY id")
			}
		}
	}
}

func contains(hay, needle string) bool {
	for i := 0; i+len(needle) <= len(hay); i++ {
		if hay[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
