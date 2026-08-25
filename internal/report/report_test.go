package report

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"dq-pipeline/internal/parse"
	"dq-pipeline/internal/quality"
)

type failWriter struct{ fail bool }

func (f failWriter) Write(p []byte) (int, error) {
	if f.fail {
		return 0, errors.New("write fail")
	}
	return len(p), nil
}

func TestWriteReportOK(t *testing.T) {
	rep := quality.Report{TotalRows: 2, TotalChecks: 4, Score: 0.5}
	var buf bytes.Buffer
	if err := WriteReport(&buf, rep); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "total_rows: 2") {
		t.Fatalf("report missing content: %q", buf.String())
	}
}

func TestWriteReportFlushError(t *testing.T) {
	rep := quality.Report{TotalRows: 1, TotalChecks: 1}
	if err := WriteReport(failWriter{true}, rep); err == nil {
		t.Fatal("expected flush error to propagate")
	}
}

func TestWriteCleanedOK(t *testing.T) {
	tbl := &parse.Table{Header: []string{"a", "b"}, Rows: [][]string{{"1", "2"}}}
	var buf bytes.Buffer
	if err := WriteCleaned(&buf, tbl, ','); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "a,b") {
		t.Fatalf("cleaned missing header: %q", buf.String())
	}
}
