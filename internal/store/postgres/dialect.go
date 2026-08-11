// Package postgres implements the store.Store interface for PostgreSQL.
package postgres

import (
	"fmt"
	"strings"
)

// QuoteIdent escapes an identifier with double quotes.
func QuoteIdent(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

// Limit builds the PostgreSQL pagination clause.
func Limit(limit, offset int) string {
	return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
}
