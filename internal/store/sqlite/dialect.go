package sqlite

import (
	"fmt"
	"strings"
)

// QuoteIdent escapa un identificador con comillas dobles, como exige SQLite.
func QuoteIdent(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

// Limit genera la cláusula de paginación de SQLite.
func Limit(limit, offset int) string {
	return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
}
