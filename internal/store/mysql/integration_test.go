package mysql

import (
	"os"
	"testing"

	"relm/internal/conn"
)

func envCfg(t *testing.T, prefix string) conn.ConnectionConfig {
	t.Helper()
	host := os.Getenv(prefix + "_HOST")
	if host == "" {
		t.Skipf("env %s_HOST no seteada; salteando test de integración", prefix)
	}
	port := 3306
	return conn.ConnectionConfig{
		Driver:   conn.DriverMySQL,
		Host:     host,
		Port:     port,
		User:     os.Getenv(prefix + "_USER"),
		Password: os.Getenv(prefix + "_PASSWORD"),
		Database: os.Getenv(prefix + "_DATABASE"),
	}
}

// TestIntegrationMySQL ejercita el motor contra MySQL real.
func TestIntegrationMySQL(t *testing.T) {
	cfg := envCfg(t, "SQLISH_TEST_MYSQL")
	cfg.Driver = conn.DriverMySQL
	testStore(t, cfg)
}

// TestIntegrationMariaDB ejercita el motor contra MariaDB real.
func TestIntegrationMariaDB(t *testing.T) {
	cfg := envCfg(t, "SQLISH_TEST_MARIADB")
	cfg.Driver = conn.DriverMariaDB
	testStore(t, cfg)
}

func testStore(t *testing.T, cfg conn.ConnectionConfig) {
	var (
		s   *Store
		err error
	)
	if cfg.Driver == conn.DriverMariaDB {
		s, err = NewMariaDB(cfg)
	} else {
		s, err = NewMySQL(cfg)
	}
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer s.Close()

	if _, err := s.Exec("DROP TABLE IF EXISTS relm_test"); err != nil {
		t.Fatalf("drop: %v", err)
	}
	if _, err := s.Exec("CREATE TABLE relm_test (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255) NOT NULL, email VARCHAR(255))"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := s.Exec("INSERT INTO relm_test (name, email) VALUES ('Alice','a@t.com'), ('Bob','b@t.com')"); err != nil {
		t.Fatalf("insert: %v", err)
	}
	t.Cleanup(func() {
		s.Exec("DROP TABLE IF EXISTS relm_test")
	})

	tables, err := s.Tables()
	if err != nil {
		t.Fatalf("Tables: %v", err)
	}
	if !contains(tables, "relm_test") {
		t.Errorf("Tables no incluye relm_test: %v", tables)
	}

	cols, err := s.Columns("relm_test")
	if err != nil {
		t.Fatalf("Columns: %v", err)
	}
	if len(cols) != 3 {
		t.Fatalf("len(Columns) = %d, want 3", len(cols))
	}
	if !cols[0].PK || !cols[1].NotNull {
		t.Errorf("constraints: %+v", cols)
	}

	n, err := s.CountTable("relm_test")
	if err != nil {
		t.Fatalf("CountTable: %v", err)
	}
	if n != 2 {
		t.Errorf("CountTable = %d, want 2", n)
	}

	page, err := s.SelectTablePage("relm_test", 10, 0)
	if err != nil {
		t.Fatalf("SelectTablePage: %v", err)
	}
	if len(page.Rows) != 2 || page.Rows[0][1] != "Alice" {
		t.Errorf("page.Rows = %v", page.Rows)
	}

	if v, err := s.Version(); err != nil || v == "" {
		t.Errorf("Version = %q, err=%v", v, err)
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
