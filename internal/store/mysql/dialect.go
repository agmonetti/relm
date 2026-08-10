// Package mysql implementa la interfaz store.Store para MySQL y MariaDB.
package mysql

import (
	"fmt"
	"strings"
)

// QuoteIdent escapa un identificador con backticks.
func QuoteIdent(ident string) string {
	return "`" + strings.ReplaceAll(ident, "`", "``") + "`"
}

// Limit genera la cláusula de paginación de MySQL/MariaDB.
func Limit(limit, offset int) string {
	return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
}
