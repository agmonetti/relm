package store

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestStringify(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want string
	}{
		{"nil", nil, ""},
		{"string", "abc", "abc"},
		{"text bytes", []byte("hello"), "hello"},
		{"binary bytes", []byte{0x00, 0xff, 0x10}, "0x00ff10"},
		{"time", time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC), "2024-01-02 03:04:05"},
		{"bool", true, "true"},
		{"int", int64(42), "42"},
		{"float", 3.5, "3.5"},
	}
	for _, tc := range cases {
		if got := Stringify(tc.in); got != tc.want {
			t.Errorf("Stringify(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestScanResultMaxCapsRows(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	// :memory: databases are per-connection; pin one so the table exists.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("CREATE TABLE t (x INT)"); err != nil {
		t.Fatalf("create: %v", err)
	}
	for i := 0; i < 5; i++ {
		if _, err := db.Exec("INSERT INTO t VALUES (?)", i); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	rows, err := db.QueryContext(context.Background(), "SELECT * FROM t")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	res, err := ScanResultMax(rows, 3)
	rows.Close() // the cursor is abandoned mid-stream; release it before the next query
	if err != nil {
		t.Fatalf("ScanResultMax: %v", err)
	}
	if len(res.Rows) != 3 {
		t.Errorf("Rows = %d, want 3", len(res.Rows))
	}
	if !res.Truncated {
		t.Error("Truncated = false, want true")
	}

	rows2, err := db.QueryContext(context.Background(), "SELECT * FROM t")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	res2, err := ScanResultMax(rows2, 0) // 0 = unlimited
	if err != nil {
		t.Fatalf("ScanResultMax: %v", err)
	}
	if len(res2.Rows) != 5 || res2.Truncated {
		t.Errorf("unlimited: Rows=%d Truncated=%v, want 5/false", len(res2.Rows), res2.Truncated)
	}

	rows3, err := db.QueryContext(context.Background(), "SELECT * FROM t")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	res3, err := ScanResultMax(rows3, 10) // more than available
	if err != nil {
		t.Fatalf("ScanResultMax: %v", err)
	}
	if len(res3.Rows) != 5 || res3.Truncated {
		t.Errorf("over-limit: Rows=%d Truncated=%v, want 5/false", len(res3.Rows), res3.Truncated)
	}
}
