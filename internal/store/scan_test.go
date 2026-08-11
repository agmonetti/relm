package store

import (
	"testing"
	"time"
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
