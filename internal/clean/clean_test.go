package clean

import (
	"testing"

	"dq-pipeline/internal/parse"
	"dq-pipeline/internal/quality"
)

func TestCleanDropsCriticalRows(t *testing.T) {
	tbl := &parse.Table{
		Header: []string{"name", "age"},
		Rows: [][]string{
			{"Alice", "30"},
			{"", "25"},
			{"Carol", "200"},
		},
	}
	rules := []quality.Rule{{Column: "name", Kind: "notnull", Critical: true}}
	rep, _ := quality.Evaluate(tbl, rules)
	cleaned := Clean(tbl, rules, rep, Options{DropCritical: true})
	if len(cleaned.Rows) != 2 {
		t.Fatalf("rows=%d want 2", len(cleaned.Rows))
	}
	if cleaned.Rows[0][0] != "Alice" || cleaned.Rows[1][0] != "Carol" {
		t.Fatalf("kept wrong rows: %v", cleaned.Rows)
	}
}

func TestCleanTrim(t *testing.T) {
	tbl := &parse.Table{
		Header: []string{"name", "age"},
		Rows:   [][]string{{"  Alice  ", "30"}},
	}
	rep, _ := quality.Evaluate(tbl, nil)
	cleaned := Clean(tbl, nil, rep, Options{TrimWhitespace: true})
	if cleaned.Rows[0][0] != "Alice" {
		t.Fatalf("trim failed: %q", cleaned.Rows[0][0])
	}
}

func TestCleanPreservesOriginal(t *testing.T) {
	tbl := &parse.Table{
		Header: []string{"x"},
		Rows:   [][]string{{"  hello  "}, {"  world  "}},
	}
	rep, _ := quality.Evaluate(tbl, nil)
	_ = Clean(tbl, nil, rep, Options{TrimWhitespace: true})
	// original should not be modified
	if tbl.Rows[0][0] != "  hello  " {
		t.Errorf("original modified: %q", tbl.Rows[0][0])
	}
}

func TestCleanNoDropWhenNotCritical(t *testing.T) {
	tbl := &parse.Table{
		Header: []string{"val"},
		Rows:   [][]string{{"a"}, {""}, {"c"}},
	}
	rules := []quality.Rule{{Column: "val", Kind: "notnull", Critical: false}}
	rep, _ := quality.Evaluate(tbl, rules)
	cleaned := Clean(tbl, rules, rep, Options{DropCritical: true})
	// non-critical violations should not cause drops
	if len(cleaned.Rows) != 3 {
		t.Errorf("rows = %d, want 3 (no drop for non-critical)", len(cleaned.Rows))
	}
}

func TestCleanEmptyTable(t *testing.T) {
	tbl := &parse.Table{Header: []string{"a"}, Rows: nil}
	rep, _ := quality.Evaluate(tbl, nil)
	cleaned := Clean(tbl, nil, rep, Options{TrimWhitespace: true, DropCritical: true})
	if len(cleaned.Rows) != 0 {
		t.Errorf("rows = %d, want 0", len(cleaned.Rows))
	}
	if len(cleaned.Header) != 1 || cleaned.Header[0] != "a" {
		t.Errorf("header = %v", cleaned.Header)
	}
}
