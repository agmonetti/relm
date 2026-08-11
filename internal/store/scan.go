package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

// ScanResult reads all rows from a Rows and converts them to a Result.
// NULL becomes ""; binary, time and numeric types are serialized to string.
func ScanResult(rows *sql.Rows) (*Result, error) {
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}
	columns := make([]string, len(colTypes))
	for i, ct := range colTypes {
		columns[i] = ct.Name()
	}

	var res Result
	res.Affected = -1 // read query
	res.Columns = columns
	for rows.Next() {
		vals := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make([]string, len(columns))
		for i, v := range vals {
			row[i] = Stringify(v)
		}
		res.Rows = append(res.Rows, row)
	}
	res.Count = len(res.Rows)
	return &res, rows.Err()
}

// Stringify converts a value scanned from database/sql into a string.
func Stringify(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	case time.Time:
		return t.Format("2006-01-02 15:04:05")
	case bool:
		return strconv.FormatBool(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	default:
		return fmt.Sprintf("%v", t)
	}
}
