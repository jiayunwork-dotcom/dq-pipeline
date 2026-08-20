package pipeline

import (
	"testing"
)

func TestTopoSortLinear(t *testing.T) {
	stages := map[string]Stage{
		"c": PassthroughStage("c", []string{"b"}),
		"b": PassthroughStage("b", []string{"a"}),
		"a": PassthroughStage("a", nil),
	}
	order, err := topoSort(stages)
	if err != nil {
		t.Fatal(err)
	}
	// a must come before b, b before c
	idx := make(map[string]int)
	for i, name := range order {
		idx[name] = i
	}
	if idx["a"] >= idx["b"] || idx["b"] >= idx["c"] {
		t.Errorf("order = %v, want a < b < c", order)
	}
}

func TestTopoSortCycleDetection(t *testing.T) {
	stages := map[string]Stage{
		"a": PassthroughStage("a", []string{"c"}),
		"b": PassthroughStage("b", []string{"a"}),
		"c": PassthroughStage("c", []string{"b"}),
	}
	_, err := topoSort(stages)
	if err == nil {
		t.Fatal("expected cycle error")
	}
}

func TestTopoSortDiamond(t *testing.T) {
	stages := map[string]Stage{
		"a": PassthroughStage("a", nil),
		"b": PassthroughStage("b", []string{"a"}),
		"c": PassthroughStage("c", []string{"a"}),
		"d": PassthroughStage("d", []string{"b", "c"}),
	}
	order, err := topoSort(stages)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 4 {
		t.Fatalf("order len = %d", len(order))
	}
	idx := make(map[string]int)
	for i, name := range order {
		idx[name] = i
	}
	if idx["a"] >= idx["b"] || idx["a"] >= idx["c"] || idx["b"] >= idx["d"] || idx["c"] >= idx["d"] {
		t.Errorf("invalid order: %v", order)
	}
}

func TestTopoSortMissingDep(t *testing.T) {
	stages := map[string]Stage{
		"a": PassthroughStage("a", []string{"missing"}),
	}
	_, err := topoSort(stages)
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}

func TestTransitiveDeps(t *testing.T) {
	stages := map[string]Stage{
		"a": PassthroughStage("a", nil),
		"b": PassthroughStage("b", []string{"a"}),
		"c": PassthroughStage("c", []string{"b"}),
	}
	deps, err := TransitiveDeps(stages, "c")
	if err != nil {
		t.Fatal(err)
	}
	// c depends on b and transitively a
	if len(deps) != 2 {
		t.Fatalf("transitive deps = %v, want 2", deps)
	}
}

func TestLevels(t *testing.T) {
	stages := map[string]Stage{
		"a": PassthroughStage("a", nil),
		"b": PassthroughStage("b", nil),
		"c": PassthroughStage("c", []string{"a", "b"}),
		"d": PassthroughStage("d", []string{"c"}),
	}
	levels, err := Levels(stages)
	if err != nil {
		t.Fatal(err)
	}
	if len(levels) != 3 {
		t.Fatalf("levels = %d, want 3", len(levels))
	}
	// level 0: a, b; level 1: c; level 2: d
	if len(levels[0]) != 2 {
		t.Errorf("level 0 = %v, want 2 items", levels[0])
	}
}

func TestValidateDeps(t *testing.T) {
	good := map[string]Stage{
		"a": PassthroughStage("a", nil),
		"b": PassthroughStage("b", []string{"a"}),
	}
	if err := ValidateDeps(good); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	bad := map[string]Stage{
		"x": PassthroughStage("x", []string{"y"}),
		"y": PassthroughStage("y", []string{"x"}),
	}
	if err := ValidateDeps(bad); err == nil {
		t.Error("expected cycle error")
	}
}
