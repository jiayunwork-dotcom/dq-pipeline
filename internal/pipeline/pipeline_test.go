package pipeline

import (
	"context"
	"fmt"
	"testing"
	"time"

	"dq-pipeline/internal/parse"
)

func sampleTable() *parse.Table {
	return &parse.Table{
		Header: []string{"name", "age", "city"},
		Rows: [][]string{
			{"Alice", "30", "NYC"},
			{"Bob", "25", "LA"},
			{"Carol", "", "SF"},
		},
	}
}

func TestPipelineLinearExecution(t *testing.T) {
	p := New()

	s1 := PassthroughStage("stage-a", nil)
	s2 := PassthroughStage("stage-b", []string{"stage-a"})
	s3 := PassthroughStage("stage-c", []string{"stage-b"})

	p.AddStage(s1)
	p.AddStage(s2)
	p.AddStage(s3)

	if err := p.Resolve(); err != nil {
		t.Fatal(err)
	}

	results, err := p.Run(context.Background(), sampleTable())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("results = %d, want 3", len(results))
	}
	for _, name := range []string{"stage-a", "stage-b", "stage-c"} {
		if results[name] == nil {
			t.Errorf("missing result for %s", name)
		}
	}
}

func TestPipelineCancellationPreservesCompleted(t *testing.T) {
	p := New()

	s1 := NewFuncStage("fast", nil, func(_ context.Context, tbl *parse.Table) (*parse.Table, error) {
		return tbl, nil
	})

	s2 := NewFuncStage("slow", []string{"fast"}, func(ctx context.Context, tbl *parse.Table) (*parse.Table, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})

	p.AddStage(s1)
	p.AddStage(s2)
	p.Resolve()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	results, err := p.Run(ctx, sampleTable())
	if err == nil {
		t.Fatal("expected error from cancelled pipeline")
	}

	if r, ok := results["fast"]; !ok || r.Error != nil {
		t.Errorf("fast stage result: %v", results["fast"])
	}
}

func TestPipelineDuplicateStage(t *testing.T) {
	p := New()
	p.AddStage(PassthroughStage("dup", nil))
	err := p.AddStage(PassthroughStage("dup", nil))
	if err == nil {
		t.Error("expected error for duplicate stage name")
	}
}

func TestPipelineResolveBeforeRun(t *testing.T) {
	p := New()
	p.AddStage(PassthroughStage("x", nil))
	_, err := p.Run(context.Background(), sampleTable())
	if err == nil {
		t.Error("expected error when running unresolved pipeline")
	}
}

func TestPipelineStageError(t *testing.T) {
	p := New()
	bad := NewFuncStage("fail", nil, func(_ context.Context, _ *parse.Table) (*parse.Table, error) {
		return nil, fmt.Errorf("intentional failure")
	})
	p.AddStage(bad)
	p.Resolve()
	results, err := p.Run(context.Background(), sampleTable())
	if err == nil {
		t.Fatal("expected error from failed stage")
	}
	if results["fail"].Error == nil {
		t.Error("expected error in stage result")
	}
}

func TestPipelineFilterStage(t *testing.T) {
	p := New()
	p.AddStage(FilterStage("filter-age", "age", nil))
	p.Resolve()
	results, err := p.Run(context.Background(), sampleTable())
	if err != nil {
		t.Fatal(err)
	}
	if results["filter-age"].Table == nil {
		t.Fatal("nil output table")
	}
	if len(results["filter-age"].Table.Rows) != 2 {
		t.Errorf("rows = %d, want 2", len(results["filter-age"].Table.Rows))
	}
}

func TestPipelineStageNames(t *testing.T) {
	p := New()
	p.AddStage(PassthroughStage("b", []string{"a"}))
	p.AddStage(PassthroughStage("a", nil))
	p.Resolve()
	names := p.StageNames()
	if len(names) != 2 || names[0] != "a" || names[1] != "b" {
		t.Errorf("order = %v, want [a, b]", names)
	}
}

func TestMakeCheckpoint(t *testing.T) {
	results := map[string]*StageResult{
		"s1": {StageName: "s1"},
		"s2": {StageName: "s2", Error: fmt.Errorf("fail")},
	}
	cp := MakeCheckpoint("test-pipe", results)
	if cp.PipelineID != "test-pipe" {
		t.Errorf("PipelineID = %q", cp.PipelineID)
	}
	if cp.Failed != "s2" {
		t.Errorf("Failed = %q, want s2", cp.Failed)
	}
	if len(cp.Completed) != 1 || cp.Completed[0] != "s1" {
		t.Errorf("Completed = %v", cp.Completed)
	}
}

func TestStageResultDuration(t *testing.T) {
	sr := &StageResult{
		StartedAt: time.Now().Add(-100 * time.Millisecond),
		EndedAt:   time.Now(),
	}
	if sr.Duration() < 90*time.Millisecond {
		t.Errorf("duration too short: %v", sr.Duration())
	}
}
