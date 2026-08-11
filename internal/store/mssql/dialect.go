// Package mssql implements the store.Store interface for SQL Server.
package mssql

import (
	"fmt"
	"strings"
)

// QuoteIdent escapes an identifier with brackets.
func QuoteIdent(ident string) string {
	return "[" + strings.ReplaceAll(ident, "]", "]]") + "]"
}

// Limit builds the SQL Server pagination clause.
// SQL Server requires ORDER BY for OFFSET/FETCH; it orders by the first column.
func Limit(limit, offset int) string {
	return fmt.Sprintf("ORDER BY 1 OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", offset, limit)
}
