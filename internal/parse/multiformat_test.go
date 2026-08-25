package parse

import (
	"strings"
	"testing"
)

func TestDetectFormatCSV(t *testing.T) {
	if f := DetectFormat("name,age,city"); f != "csv" {
		t.Errorf("format = %q, want csv", f)
	}
}

func TestDetectFormatTSV(t *testing.T) {
	if f := DetectFormat("name\tage\tcity"); f != "tsv" {
		t.Errorf("format = %q, want tsv", f)
	}
}

func TestDetectFormatJSON(t *testing.T) {
	if f := DetectFormat(`[{"name":"Alice"}]`); f != "json" {
		t.Errorf("format = %q, want json", f)
	}
	if f := DetectFormat(`{"data":[]}`); f != "json" {
		t.Errorf("format = %q, want json", f)
	}
}

func TestParseJSONArray(t *testing.T) {
	input := `[{"name":"Alice","age":30},{"name":"Bob","age":25}]`
	tbl, err := ParseJSON(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(tbl.Header) < 2 {
		t.Fatalf("headers = %v", tbl.Header)
	}
	if len(tbl.Rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(tbl.Rows))
	}
}

func TestParseJSONEmptyArray(t *testing.T) {
	tbl, err := ParseJSON(strings.NewReader("[]"))
	if err != nil {
		t.Fatal(err)
	}
	if len(tbl.Rows) != 0 {
		t.Errorf("rows = %d, want 0", len(tbl.Rows))
	}
}

func TestParseJSONMissingKeys(t *testing.T) {
	input := `[{"a":"1","b":"2"},{"a":"3","c":"4"}]`
	tbl, err := ParseJSON(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(tbl.Header) != 3 {
		t.Fatalf("headers = %v, want 3 columns", tbl.Header)
	}
}

func TestParseFixed(t *testing.T) {
	input := "NAME     AGE CITY     \nAlice    30  NYC      \nBob      25  LA       \n"
	tbl, err := ParseFixed(strings.NewReader(input), []int{9, 4, 9})
	if err != nil {
		t.Fatal(err)
	}
	if len(tbl.Header) != 3 {
		t.Fatalf("headers = %v", tbl.Header)
	}
	if tbl.Header[0] != "NAME" {
		t.Errorf("header[0] = %q", tbl.Header[0])
	}
	if len(tbl.Rows) != 2 {
		t.Fatalf("rows = %d", len(tbl.Rows))
	}
	if tbl.Rows[0][0] != "Alice" {
		t.Errorf("row[0][0] = %q", tbl.Rows[0][0])
	}
}

func TestParseFixedNoWidths(t *testing.T) {
	_, err := ParseFixed(strings.NewReader("hello"), nil)
	if err == nil {
		t.Error("expected error for no widths")
	}
}

func TestAutoParseCSV(t *testing.T) {
	input := "a,b\n1,2\n3,4\n"
	tbl, err := AutoParse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(tbl.Rows) != 2 {
		t.Errorf("rows = %d", len(tbl.Rows))
	}
}

func TestAutoParseJSON(t *testing.T) {
	input := `[{"x":"1"},{"x":"2"}]`
	tbl, err := AutoParse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(tbl.Rows) != 2 {
		t.Errorf("rows = %d", len(tbl.Rows))
	}
}

func TestAutoParseTSV(t *testing.T) {
	input := "a\tb\n1\t2\n"
	tbl, err := AutoParse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if tbl.Header[0] != "a" || tbl.Header[1] != "b" {
		t.Errorf("header = %v", tbl.Header)
	}
}
