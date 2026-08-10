// Package postgres implementa la interfaz store.Store para PostgreSQL.
package postgres

import (
	"fmt"
	"strings"
)

// QuoteIdent escapa un identificador con comillas dobles.
func QuoteIdent(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

// Limit genera la cláusula de paginación de PostgreSQL.
func Limit(limit, offset int) string {
	return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
}
