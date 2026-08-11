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

// Limit builds the PostgreSQL pagination clause. ORDER BY 1 keeps pages stable
// when the underlying rows change between refreshes.
func Limit(limit, offset int) string {
	return fmt.Sprintf("ORDER BY 1 LIMIT %d OFFSET %d", limit, offset)
}
