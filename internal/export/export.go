// Package export serializes a store.Result into CSV or JSON files. It is a thin
// layer over the in-memory result: the store already stringified every cell, so
// exporting never touches the database or a driver.
package export

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"io"

	"github.com/agmonetti/relm/internal/store"
)

// WriteCSV writes the result as RFC 4180 CSV with the columns as the header
// row. SQL NULL becomes an empty field (CSV has no null concept); a NULL and an
// empty string therefore look the same, which is the accepted CSV behavior.
func WriteCSV(res *store.Result, w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(res.Columns); err != nil {
		return err
	}
	for _, row := range res.Rows {
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// WriteJSON writes the result as a JSON array of objects, one per row, with the
// columns as keys in their original order. SQL NULL becomes JSON null; every
// other value is a string (relm's model is strings-only). HTML escaping is
// disabled so a value like "a < b" is exported verbatim.
func WriteJSON(res *store.Result, w io.Writer) error {
	rows := make([]orderedRow, len(res.Rows))
	for i, row := range res.Rows {
		vals := make([]any, len(row))
		for j := range row {
			if isNull(res.Nulls, i, j) {
				vals[j] = nil
			} else {
				vals[j] = row[j]
			}
		}
		rows[i] = orderedRow{keys: res.Columns, values: vals}
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(rows)
}

// isNull reports whether the cell at (row, col) was SQL NULL, guarding against
// a missing or shorter Nulls matrix (nil means no NULL in the result).
func isNull(nulls [][]bool, row, col int) bool {
	return nulls != nil && row < len(nulls) && col < len(nulls[row]) && nulls[row][col]
}

// orderedRow marshals as a JSON object keeping the key order of the columns,
// which json.Marshal on a map would lose (map keys sort alphabetically).
type orderedRow struct {
	keys   []string
	values []any
}

// MarshalJSON implements json.Marshaler. json.Marshal would HTML-escape "<",
// "&" and ">" in the cell values, so each key and value is encoded with an
// escape-less encoder to keep the data verbatim.
func (r orderedRow) MarshalJSON() ([]byte, error) {
	var b []byte
	b = append(b, '{')
	for i, k := range r.keys {
		if i > 0 {
			b = append(b, ',')
		}
		kb, err := jsonNoEscape(k)
		if err != nil {
			return nil, err
		}
		vb, err := jsonNoEscape(r.values[i])
		if err != nil {
			return nil, err
		}
		b = append(b, kb...)
		b = append(b, ':')
		b = append(b, vb...)
	}
	b = append(b, '}')
	return b, nil
}

// jsonNoEscape encodes v as JSON without HTML escaping and without the trailing
// newline the Encoder appends.
func jsonNoEscape(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(b.Bytes(), []byte("\n")), nil
}
