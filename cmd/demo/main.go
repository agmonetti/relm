// Command demo creates a sample SQLite database (demo.db) to try relm without
// needing the sqlite3 CLI. Cross-platform replacement for `make demo`.
package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	path := "demo.db"
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	if err := create(path); err != nil {
		fmt.Fprintf(os.Stderr, "demo: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Example database created: %s - open it with relm (engine SQLite, path %s)\n", path, path)
}

func create(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return err
	}
	defer db.Close()

	stmts := []string{
		"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, email TEXT)",
		"CREATE TABLE orders (id INTEGER PRIMARY KEY, total REAL, user_id INTEGER)",
		"INSERT INTO users (name, email) VALUES ('Alice','alice@test.com'), ('Bob','bob@test.com'), ('Carol','carol@test.com')",
		"INSERT INTO orders (total, user_id) VALUES (19.99, 1), (4.50, 2), (129.00, 1)",
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("%s: %w", s, err)
		}
	}
	return nil
}
