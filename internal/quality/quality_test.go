package quality

import (
	"math"
	"strings"
	"testing"

	"dq-pipeline/internal/parse"
)

func tableFrom(s string) *parse.Table {
	t, err := parse.Parse(strings.NewReader(s), ',')
	if err != nil {
		panic(err)
	}
	return t
}

func TestEvaluateMissingColumn(t *testing.T) {
	tbl := tableFrom("name,age\nAlice,30\n")
	rules := []Rule{{Column: "missing", Kind: "notnull"}}
	if _, err := Evaluate(tbl, rules); err == nil {
		t.Fatal("expected error for rule on missing column")
	}
}

func TestEvaluateScoreAndViolations(t *testing.T) {
	tbl := tableFrom("name,age\nAlice,30\n,25\nCarol,200\n")
	rules := []Rule{
		{Column: "name", Kind: "notnull"},
		{Column: "age", Kind: "range", Min: 0, Max: 120},
	}
	rep, err := Evaluate(tbl, rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rep.TotalChecks != 6 {
		t.Fatalf("TotalChecks=%d want 6", rep.TotalChecks)
	}
	if len(rep.Violations) != 2 {
		t.Fatalf("Violations=%d want 2", len(rep.Violations))
	}
	want := 1 - 2.0/6.0
	if math.Abs(rep.Score-want) > 1e-9 {
		t.Fatalf("Score=%v want %v", rep.Score, want)
	}
	if rep.Violations[0].Row != 1 || rep.Violations[0].Rule != 0 {
		t.Fatalf("sort order wrong: %+v", rep.Violations[0])
	}
}

func TestEvaluateUnique(t *testing.T) {
	tbl := tableFrom("id\nA1\nA2\nA1\n")
	rules := []Rule{{Column: "id", Kind: "unique"}}
	rep, err := Evaluate(tbl, rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rep.Violations) != 2 {
		t.Fatalf("unique violations=%d want 2 (both A1 rows)", len(rep.Violations))
	}
}
