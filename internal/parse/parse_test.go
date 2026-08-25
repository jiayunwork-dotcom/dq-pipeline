package parse

import (
	"strings"
	"testing"
)

type errReader struct{}

func (errReader) Read(p []byte) (int, error) {
	return 0, errBoom
}

var errBoom = errString("boom")

type errString string

func (e errString) Error() string { return string(e) }

func TestParseInvalidDelimiter(t *testing.T) {
	_, err := Parse(strings.NewReader("a,b\n1,2\n"), 0x01)
	if err == nil {
		t.Fatal("expected error for control-char delimiter")
	}
}

func TestParseReaderError(t *testing.T) {
	_, err := Parse(errReader{}, ',')
	if err == nil {
		t.Fatal("expected error when reader fails")
	}
}

func TestParseBasic(t *testing.T) {
	in := "name,age\nAlice,30\nBob,25\n"
	tbl, err := Parse(strings.NewReader(in), ',')
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tbl.Header) != 2 || len(tbl.Rows) != 2 {
		t.Fatalf("got header=%d rows=%d", len(tbl.Header), len(tbl.Rows))
	}
}
