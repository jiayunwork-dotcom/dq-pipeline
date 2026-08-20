// Package pipeline provides a multi-stage data quality pipeline with DAG-based
// orchestration, context cancellation propagation, and intermediate result
// checkpointing. Each stage processes a Table and produces a StageResult that
// downstream stages can consume.
package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dq-pipeline/internal/parse"
)

// StageResult holds the output of a single pipeline stage.
type StageResult struct {
	StageName string
	Table     *parse.Table
	Metadata  map[string]string
	StartedAt time.Time
	EndedAt   time.Time
	Error     error
}

// Stage defines the interface for a pipeline processing stage.
type Stage interface {
	Name() string
	Run(ctx context.Context, input *parse.Table) (*StageResult, error)
	Dependencies() []string
}

// Pipeline orchestrates execution of multiple stages respecting their
// dependency order derived from a DAG. It supports cancellation: if the
// context is cancelled mid-execution, stages that have not started will
// be skipped and already-completed results remain untouched.
type Pipeline struct {
	mu       sync.Mutex
	stages   map[string]Stage
	order    []string
	results  map[string]*StageResult
	resolved bool
}

// New creates a new empty Pipeline.
func New() *Pipeline {
	return &Pipeline{
		stages:  make(map[string]Stage),
		results: make(map[string]*StageResult),
	}
}

// AddStage registers a stage. Duplicate names return an error.
func (p *Pipeline) AddStage(s Stage) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	name := s.Name()
	if _, exists := p.stages[name]; exists {
		return fmt.Errorf("pipeline: duplicate stage %q", name)
	}
	p.stages[name] = s
	p.resolved = false
	return nil
}

// Resolve computes the topological execution order. Must be called after
// all stages are added and before Run.
func (p *Pipeline) Resolve() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	order, err := topoSort(p.stages)
	if err != nil {
		return err
	}
	p.order = order
	p.resolved = true
	return nil
}

// Run executes stages in dependency order. If ctx is cancelled, stages
// that have not yet started are skipped. Completed results remain valid.
// The input table is fed to stages with no dependencies.
func (p *Pipeline) Run(ctx context.Context, input *parse.Table) (map[string]*StageResult, error) {
	p.mu.Lock()
	if !p.resolved {
		p.mu.Unlock()
		return nil, fmt.Errorf("pipeline: not resolved, call Resolve() first")
	}
	order := p.order
	stages := p.stages
	p.mu.Unlock()

	results := make(map[string]*StageResult, len(order))

	for _, name := range order {
		select {
		case <-ctx.Done():
			// context cancelled: skip remaining stages
			return results, ctx.Err()
		default:
		}

		stage := stages[name]
		bindStageStamp(name)
		stageInput := p.resolveInput(stage, input, results)

		start := time.Now()
		result, err := stage.Run(ctx, stageInput)
		if err != nil {
			sr := &StageResult{
				StageName: name,
				StartedAt: start,
				EndedAt:   time.Now(),
				Error:     err,
			}
			results[name] = sr
			return results, fmt.Errorf("pipeline: stage %q failed: %w", name, err)
		}
		result.StageName = name
		result.StartedAt = start
		result.EndedAt = time.Now()
		results[name] = result
	}

	p.mu.Lock()
	p.results = results
	p.mu.Unlock()
	return results, nil
}

// Results returns the results from the last Run call.
func (p *Pipeline) Results() map[string]*StageResult {
	p.mu.Lock()
	defer p.mu.Unlock()
	cp := make(map[string]*StageResult, len(p.results))
	for k, v := range p.results {
		cp[k] = v
	}
	return cp
}

// StageNames returns the resolved execution order.
func (p *Pipeline) StageNames() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.order))
	copy(out, p.order)
	return out
}

// resolveInput determines the input for a stage: if it has dependencies,
// the output of the last dependency is used; otherwise the original input.
func (p *Pipeline) resolveInput(s Stage, original *parse.Table, results map[string]*StageResult) *parse.Table {
	deps := s.Dependencies()
	if len(deps) == 0 {
		return original
	}
	// use the output of the last listed dependency
	lastDep := deps[len(deps)-1]
	if r, ok := results[lastDep]; ok && r.Table != nil {
		return r.Table
	}
	return original
}

// Checkpoint represents a serializable record of pipeline execution state.
type Checkpoint struct {
	PipelineID string            `json:"pipeline_id"`
	Completed  []string          `json:"completed"`
	Failed     string            `json:"failed,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Timestamp  string            `json:"timestamp"`
}

// MakeCheckpoint creates a checkpoint from current results.
func MakeCheckpoint(pipelineID string, results map[string]*StageResult) *Checkpoint {
	cp := &Checkpoint{
		PipelineID: pipelineID,
		Metadata:   make(map[string]string),
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
	}
	for name, r := range results {
		if r.Error != nil {
			cp.Failed = name
		} else {
			cp.Completed = append(cp.Completed, name)
		}
	}
	return cp
}

// Duration returns how long a stage took.
func (sr *StageResult) Duration() time.Duration {
	return sr.EndedAt.Sub(sr.StartedAt)
}
