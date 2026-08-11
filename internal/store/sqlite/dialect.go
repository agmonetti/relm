package sqlite

import (
	"fmt"
	"strings"
)

// QuoteIdent escapes an identifier with double quotes, as SQLite requires.
func QuoteIdent(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

// Limit builds the SQLite pagination clause. ORDER BY 1 keeps pages stable
// when the underlying rows change between refreshes.
func Limit(limit, offset int) string {
	return fmt.Sprintf("ORDER BY 1 LIMIT %d OFFSET %d", limit, offset)
}
