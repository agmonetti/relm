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
		{"mysql bit true", []byte{1}, "1"},
		{"mysql bit false", []byte{0}, "0"},
		{"control bytes", []byte{0x01, 0x02, 0x03}, "0x010203"},
		{"utf8 accented bytes", []byte("válido ñañdú"), "válido ñañdú"},
		{"binary bytes", []byte{0x00, 0xff, 0x10}, "0x00ff10"},
		{"time", time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC), "2024-01-02 03:04:05"},
		{"bool", true, "true"},
		{"int", int64(42), "42"},
		{"float", 3.5, "3.5"},
	}
	for _, tc := range cases {
		if got := Stringify(tc.in); got != tc.want {
			t.Errorf("Stringify(%v) [%s] = %q, want %q", tc.in, tc.name, got, tc.want)
		}
	}
}

func TestScanResultMax_TracksNulls(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("CREATE TABLE t (a TEXT, b TEXT)"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := db.Exec("INSERT INTO t VALUES ('x', NULL), (NULL, 'y'), (NULL, NULL)"); err != nil {
		t.Fatalf("insert: %v", err)
	}

	rows, err := db.QueryContext(context.Background(), "SELECT * FROM t")
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer rows.Close()
	res, err := ScanResultMax(rows, 0)
	if err != nil {
		t.Fatalf("ScanResultMax: %v", err)
	}
	want := [][]bool{
		{false, true},
		{true, false},
		{true, true},
	}
	if len(res.Nulls) != len(want) {
		t.Fatalf("Nulls rows = %d, want %d", len(res.Nulls), len(want))
	}
	for i := range want {
		if res.Nulls[i][0] != want[i][0] || res.Nulls[i][1] != want[i][1] {
			t.Errorf("row %d Nulls = %v, want %v", i, res.Nulls[i], want[i])
		}
	}
	if res.Rows[0][1] != "" || res.Rows[1][0] != "" {
		t.Errorf("NULL cells should stringify to empty, got %v", res.Rows)
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
