// Package mssql implementa la interfaz store.Store para SQL Server.
package mssql

import (
	"fmt"
	"strings"
)

// QuoteIdent escapa un identificador con corchetes.
func QuoteIdent(ident string) string {
	return "[" + strings.ReplaceAll(ident, "]", "]]") + "]"
}

// Limit genera la cláusula de paginación de SQL Server.
// SQL Server exige ORDER BY para OFFSET/FETCH; se ordena por la primera columna.
func Limit(limit, offset int) string {
	return fmt.Sprintf("ORDER BY 1 OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", offset, limit)
}
