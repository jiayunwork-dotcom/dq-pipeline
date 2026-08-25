package quality

import (
	"strings"
	"testing"

	"dq-pipeline/internal/parse"
)

func tbl(csv string) *parse.Table {
	t, _ := parse.Parse(strings.NewReader(csv), ',')
	return t
}

func TestEvalEnum(t *testing.T) {
	data := tbl("status\nactive\ninactive\npending\nunknown\n")
	rules := []Rule{{Column: "status", Kind: "enum", Pattern: "active|inactive|pending"}}
	viols, err := EvaluateExtended(data, rules)
	if err != nil {
		t.Fatal(err)
	}
	if len(viols) != 1 {
		t.Errorf("violations = %d, want 1 (unknown)", len(viols))
	}
}

func TestEvalLength(t *testing.T) {
	data := tbl("code\nAB\nABCDE\nA\nABCDEFGH\n")
	rules := []Rule{{Column: "code", Kind: "length", Min: 2, Max: 5}}
	viols, _ := EvaluateExtended(data, rules)
	if len(viols) != 2 {
		t.Errorf("violations = %d, want 2", len(viols))
	}
}

func TestEvalDate(t *testing.T) {
	data := tbl("dt\n2024-01-15\nnot-a-date\n2024-12-31\n")
	rules := []Rule{{Column: "dt", Kind: "date", Pattern: "2006-01-02"}}
	viols, _ := EvaluateExtended(data, rules)
	if len(viols) != 1 {
		t.Errorf("violations = %d, want 1", len(viols))
	}
}

func TestEvalTypeInt(t *testing.T) {
	data := tbl("num\n42\n3.14\nhello\n7\n")
	rules := []Rule{{Column: "num", Kind: "type", Pattern: "int"}}
	viols, _ := EvaluateExtended(data, rules)
	if len(viols) != 2 {
		t.Errorf("violations = %d, want 2", len(viols))
	}
}

func TestEvalTypeBool(t *testing.T) {
	data := tbl("flag\ntrue\nfalse\n1\n0\nyes\n")
	rules := []Rule{{Column: "flag", Kind: "type", Pattern: "bool"}}
	viols, _ := EvaluateExtended(data, rules)
	if len(viols) != 1 {
		t.Errorf("violations = %d, want 1", len(viols))
	}
}

func TestEvalCrossfieldGT(t *testing.T) {
	data := tbl("min,max\n1,10\n5,3\n2,2\n")
	rules := []Rule{{Column: "min", Kind: "crossfield", Pattern: "max gt min"}}
	viols, err := EvaluateExtended(data, rules)
	if err != nil {
		t.Fatal(err)
	}
	if len(viols) != 2 {
		t.Errorf("violations = %d, want 2", len(viols))
	}
}

func TestEvalCrossfieldMissingColumn(t *testing.T) {
	data := tbl("a\n1\n")
	rules := []Rule{{Column: "a", Kind: "crossfield", Pattern: "a gt missing"}}
	_, err := EvaluateExtended(data, rules)
	if err == nil {
		t.Error("expected error for missing column")
	}
}

func TestSummarize(t *testing.T) {
	rep := Report{
		TotalRows:   100,
		TotalChecks: 300,
		Score:       0.95,
		Violations:  []Violation{{Column: "x"}},
		ByColumn:    map[string]int{"x": 1},
	}
	s := Summarize(rep)
	if !strings.Contains(s, "0.9500") {
		t.Errorf("summary missing score: %q", s)
	}
}

func TestMergeViolations(t *testing.T) {
	a := []Violation{{Rule: 0, Column: "a", Row: 1, Reason: "x"}}
	b := []Violation{
		{Rule: 0, Column: "a", Row: 1, Reason: "x"},
		{Rule: 1, Column: "b", Row: 2, Reason: "y"},
	}
	merged := MergeViolations(a, b)
	if len(merged) != 2 {
		t.Errorf("merged = %d, want 2", len(merged))
	}
}
