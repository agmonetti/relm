package export

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/agmonetti/relm/internal/store"
)

func res(cols []string, rows [][]string, nulls [][]bool) *store.Result {
	return &store.Result{Columns: cols, Rows: rows, Nulls: nulls}
}

func TestWriteCSV_Basic(t *testing.T) {
	r := res(
		[]string{"id", "name"},
		[][]string{{"1", "Alice"}, {"2", "Bob"}},
		nil,
	)
	var b bytes.Buffer
	if err := WriteCSV(r, &b); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	want := "id,name\n1,Alice\n2,Bob\n"
	if b.String() != want {
		t.Errorf("CSV = %q, want %q", b.String(), want)
	}
}

func TestWriteCSV_Quoting(t *testing.T) {
	r := res(
		[]string{"a", "b", "c"},
		[][]string{{"with, comma", `with "quotes"`, "line\nbreak"}},
		nil,
	)
	var b bytes.Buffer
	if err := WriteCSV(r, &b); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	if !strings.HasPrefix(b.String(), "a,b,c\n") {
		t.Errorf("header = %q", b.String())
	}
	body := strings.TrimPrefix(b.String(), "a,b,c\n")
	for _, want := range []string{`"with, comma"`, `"with ""quotes"""`, `"line`} {
		if !strings.Contains(body, want) {
			t.Errorf("row %q does not contain %q", body, want)
		}
	}
}

func TestWriteCSV_NullIsEmpty(t *testing.T) {
	r := res(
		[]string{"a", "b"},
		[][]string{{"x", ""}},
		[][]bool{{false, true}},
	)
	var b bytes.Buffer
	if err := WriteCSV(r, &b); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	if b.String() != "a,b\nx,\n" {
		t.Errorf("CSV = %q", b.String())
	}
}

func TestWriteJSON_NullAndEmptyString(t *testing.T) {
	r := res(
		[]string{"a", "b", "c"},
		[][]string{{"1", "", "text"}},
		[][]bool{{false, true, false}},
	)
	var b bytes.Buffer
	if err := WriteJSON(r, &b); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal(b.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v (%q)", err, b.String())
	}
	if len(got) != 1 {
		t.Fatalf("rows = %d, want 1", len(got))
	}
	row := got[0]
	if row["a"] != "1" {
		t.Errorf("a = %#v, want string \"1\"", row["a"])
	}
	if v, ok := row["b"]; !ok || v != nil {
		t.Errorf("b = %#v, want JSON null", row["b"])
	}
	if row["c"] != "text" {
		t.Errorf("c = %#v, want \"text\"", row["c"])
	}
}

func TestWriteJSON_KeepsColumnOrder(t *testing.T) {
	r := res(
		[]string{"zeta", "alpha", "middle"},
		[][]string{{"1", "2", "3"}},
		nil,
	)
	var b bytes.Buffer
	if err := WriteJSON(r, &b); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	want := `[{"zeta":"1","alpha":"2","middle":"3"}]`
	if got := strings.TrimRight(b.String(), "\n"); got != want {
		t.Errorf("JSON = %s, want %s", got, want)
	}
}

func TestWriteJSON_NoHTMLEscaping(t *testing.T) {
	r := res(
		[]string{"v"},
		[][]string{{"a < b & c > d"}},
		nil,
	)
	var b bytes.Buffer
	if err := WriteJSON(r, &b); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if strings.Contains(b.String(), `\u003c`) || strings.Contains(b.String(), `\u0026`) {
		t.Errorf("value was HTML-escaped: %s", b.String())
	}
}

func TestWriteJSON_NilNullsAreNoNulls(t *testing.T) {
	r := res(
		[]string{"a"},
		[][]string{{""}},
		nil,
	)
	var b bytes.Buffer
	if err := WriteJSON(r, &b); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	want := `[{"a":""}]`
	if got := strings.TrimRight(b.String(), "\n"); got != want {
		t.Errorf("JSON = %s, want %s (empty string, not null)", got, want)
	}
}
