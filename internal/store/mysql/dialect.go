// Package mysql implements the store.Store interface for MySQL and MariaDB.
package mysql

import (
	"fmt"
	"strings"
)

// QuoteIdent escapes an identifier with backticks.
func QuoteIdent(ident string) string {
	return "`" + strings.ReplaceAll(ident, "`", "``") + "`"
}

// Limit builds the MySQL/MariaDB pagination clause. ORDER BY 1 keeps pages
// stable when the underlying rows change between refreshes.
func Limit(limit, offset int) string {
	return fmt.Sprintf("ORDER BY 1 LIMIT %d OFFSET %d", limit, offset)
}
