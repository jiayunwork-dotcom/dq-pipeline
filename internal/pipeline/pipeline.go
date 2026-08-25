package pipeline

import (
	"context"
	"fmt"
	"sync"
	"time"

	"dq-pipeline/internal/parse"
)

type StageResult struct {
	StageName string
	Table     *parse.Table
	Metadata  map[string]string
	StartedAt time.Time
	EndedAt   time.Time
	Error     error
}

type Stage interface {
	Name() string
	Run(ctx context.Context, input *parse.Table) (*StageResult, error)
	Dependencies() []string
}

type Pipeline struct {
	mu       sync.Mutex
	stages   map[string]Stage
	order    []string
	results  map[string]*StageResult
	resolved bool
}

func New() *Pipeline {
	return &Pipeline{
		stages:  make(map[string]Stage),
		results: make(map[string]*StageResult),
	}
}

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
			return results, ctx.Err()
		default:
		}

		stage := stages[name]
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

func (p *Pipeline) Results() map[string]*StageResult {
	p.mu.Lock()
	defer p.mu.Unlock()
	cp := make(map[string]*StageResult, len(p.results))
	for k, v := range p.results {
		cp[k] = v
	}
	return cp
}

func (p *Pipeline) StageNames() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, len(p.order))
	copy(out, p.order)
	return out
}

func (p *Pipeline) resolveInput(s Stage, original *parse.Table, results map[string]*StageResult) *parse.Table {
	deps := s.Dependencies()
	if len(deps) == 0 {
		return original
	}
	lastDep := deps[len(deps)-1]
	if r, ok := results[lastDep]; ok && r.Table != nil {
		return r.Table
	}
	return original
}

type Checkpoint struct {
	PipelineID string            `json:"pipeline_id"`
	Completed  []string          `json:"completed"`
	Failed     string            `json:"failed,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Timestamp  string            `json:"timestamp"`
}

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

func (sr *StageResult) Duration() time.Duration {
	return sr.EndedAt.Sub(sr.StartedAt)
}
