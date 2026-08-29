package store

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"
)

// ScanResult reads all rows from a Rows and converts them to a Result.
// NULL becomes ""; binary, time and numeric types are serialized to string.
func ScanResult(rows *sql.Rows) (*Result, error) {
	return ScanResultMax(rows, 0)
}

// ScanResultMax reads up to max rows from Rows (0 = unlimited) and converts
// them to a Result. When the result set is longer than max, the scan stops
// early and Result.Truncated is set, so the caller can avoid loading an
// unbounded number of rows into memory.
func ScanResultMax(rows *sql.Rows, max int) (*Result, error) {
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
		if max > 0 && len(res.Rows) >= max {
			res.Truncated = true
			break
		}
		vals := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		row := make([]string, len(columns))
		nulls := make([]bool, len(columns))
		for i, v := range vals {
			if v == nil {
				nulls[i] = true
			}
			row[i] = Stringify(v)
		}
		res.Rows = append(res.Rows, row)
		res.Nulls = append(res.Nulls, nulls)
	}
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
		if len(t) == 1 && (t[0] == 0 || t[0] == 1) {
			return strconv.Itoa(int(t[0]))
		}
		if isPrintableUTF8(t) {
			return string(t)
		}
		// binary value: show as 0x… instead of raw bytes
		return "0x" + hex.EncodeToString(t)
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

func isPrintableUTF8(b []byte) bool {
	if !utf8.Valid(b) {
		return false
	}
	for _, r := range string(b) {
		if !unicode.IsPrint(r) && r != '\t' && r != '\n' && r != '\r' {
			return false
		}
	}
	return true
}
