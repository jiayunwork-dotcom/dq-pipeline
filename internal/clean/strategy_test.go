package clean

import (
	"testing"

	"dq-pipeline/internal/parse"
)

func TestFillConstant(t *testing.T) {
	tbl := &parse.Table{
		Header: []string{"name"},
		Rows:   [][]string{{"Alice"}, {""}, {"Carol"}},
	}
	out, err := ApplyFill(tbl, []FillConfig{{Column: "name", Strategy: FillConstant, Constant: "UNKNOWN"}})
	if err != nil {
		t.Fatal(err)
	}
	if out.Rows[1][0] != "UNKNOWN" {
		t.Errorf("filled = %q, want UNKNOWN", out.Rows[1][0])
	}
}

func TestFillMean(t *testing.T) {
	tbl := &parse.Table{
		Header: []string{"val"},
		Rows:   [][]string{{"10"}, {""}, {"30"}, {""}},
	}
	out, err := ApplyFill(tbl, []FillConfig{{Column: "val", Strategy: FillMean}})
	if err != nil {
		t.Fatal(err)
	}
	// mean of 10, 30 = 20
	if out.Rows[1][0] != "20" {
		t.Errorf("filled = %q, want 20", out.Rows[1][0])
	}
}

func TestFillMedian(t *testing.T) {
	tbl := &parse.Table{
		Header: []string{"val"},
		Rows:   [][]string{{"10"}, {""}, {"20"}, {"30"}, {""}},
	}
	out, err := ApplyFill(tbl, []FillConfig{{Column: "val", Strategy: FillMedian}})
	if err != nil {
		t.Fatal(err)
	}
	// median of 10, 20, 30 = 20
	if out.Rows[1][0] != "20" {
		t.Errorf("filled = %q, want 20", out.Rows[1][0])
	}
}

func TestFillForward(t *testing.T) {
	tbl := &parse.Table{
		Header: []string{"status"},
		Rows:   [][]string{{"active"}, {""}, {""}, {"inactive"}, {""}},
	}
	out, err := ApplyFill(tbl, []FillConfig{{Column: "status", Strategy: FillForward}})
	if err != nil {
		t.Fatal(err)
	}
	if out.Rows[1][0] != "active" || out.Rows[2][0] != "active" {
		t.Errorf("forward fill failed: %v", out.Rows)
	}
	if out.Rows[4][0] != "inactive" {
		t.Errorf("row4 = %q, want inactive", out.Rows[4][0])
	}
}

func TestFillBackward(t *testing.T) {
	tbl := &parse.Table{
		Header: []string{"x"},
		Rows:   [][]string{{""}, {""}, {"C"}, {""}, {"E"}},
	}
	out, err := ApplyFill(tbl, []FillConfig{{Column: "x", Strategy: FillBackward}})
	if err != nil {
		t.Fatal(err)
	}
	if out.Rows[0][0] != "C" || out.Rows[1][0] != "C" {
		t.Errorf("backward fill: %v", out.Rows)
	}
	if out.Rows[3][0] != "E" {
		t.Errorf("row3 = %q", out.Rows[3][0])
	}
}

func TestFillMissingColumn(t *testing.T) {
	tbl := &parse.Table{Header: []string{"a"}, Rows: [][]string{{"1"}}}
	_, err := ApplyFill(tbl, []FillConfig{{Column: "missing", Strategy: FillConstant}})
	if err == nil {
		t.Error("expected error for missing column")
	}
}

func TestFillPreservesOriginal(t *testing.T) {
	tbl := &parse.Table{
		Header: []string{"val"},
		Rows:   [][]string{{"10"}, {""}, {"30"}},
	}
	_, _ = ApplyFill(tbl, []FillConfig{{Column: "val", Strategy: FillMean}})
	if tbl.Rows[1][0] != "" {
		t.Errorf("original modified: %q", tbl.Rows[1][0])
	}
}

func TestClampValues(t *testing.T) {
	tbl := &parse.Table{
		Header: []string{"score"},
		Rows:   [][]string{{"5"}, {"150"}, {"-10"}, {"50"}},
	}
	out, err := ClampValues(tbl, "score", 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if out.Rows[1][0] != "100" {
		t.Errorf("clamped high = %q, want 100", out.Rows[1][0])
	}
	if out.Rows[2][0] != "0" {
		t.Errorf("clamped low = %q, want 0", out.Rows[2][0])
	}
	if out.Rows[3][0] != "50" {
		t.Errorf("unchanged = %q, want 50", out.Rows[3][0])
	}
}

func TestNormalizeColumn(t *testing.T) {
	tbl := &parse.Table{
		Header: []string{"val"},
		Rows:   [][]string{{"0"}, {"50"}, {"100"}},
	}
	out, err := NormalizeColumn(tbl, "val")
	if err != nil {
		t.Fatal(err)
	}
	if out.Rows[0][0] != "0.000000" {
		t.Errorf("min normalized = %q, want 0.000000", out.Rows[0][0])
	}
	if out.Rows[1][0] != "0.500000" {
		t.Errorf("mid normalized = %q, want 0.500000", out.Rows[1][0])
	}
	if out.Rows[2][0] != "1.000000" {
		t.Errorf("max normalized = %q, want 1.000000", out.Rows[2][0])
	}
}

func TestNormalizeMissingColumn(t *testing.T) {
	tbl := &parse.Table{Header: []string{"a"}, Rows: [][]string{{"1"}}}
	_, err := NormalizeColumn(tbl, "missing")
	if err == nil {
		t.Error("expected error for missing column")
	}
}
