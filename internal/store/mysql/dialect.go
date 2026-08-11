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

// Limit builds the MySQL/MariaDB pagination clause.
func Limit(limit, offset int) string {
	return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
}
