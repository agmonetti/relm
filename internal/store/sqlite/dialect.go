package sqlite

import (
	"fmt"
	"strings"
)

// QuoteIdent escapes an identifier with double quotes, as SQLite requires.
func QuoteIdent(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

// Limit builds the SQLite pagination clause.
func Limit(limit, offset int) string {
	return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
}
