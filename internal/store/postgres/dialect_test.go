package postgres

import "testing"

func TestQuoteIdent(t *testing.T) {
	cases := map[string]string{
		"users":      `"users"`,
		`we"ird`:     `"we""ird"`,
		"with space": `"with space"`,
	}
	for in, want := range cases {
		if got := QuoteIdent(in); got != want {
			t.Errorf("QuoteIdent(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLimit(t *testing.T) {
	if got := Limit(50, 100); got != "ORDER BY 1 LIMIT 50 OFFSET 100" {
		t.Errorf("Limit = %q", got)
	}
}
