package pipeline

import (
	"context"
	"fmt"
	"sync"

	"dq-pipeline/internal/parse"
)

type StageFactory func(config map[string]string) (Stage, error)

type Registry struct {
	mu        sync.RWMutex
	factories map[string]StageFactory
}

func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[string]StageFactory),
	}
}

func (r *Registry) Register(kind string, f StageFactory) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.factories[kind]; exists {
		return fmt.Errorf("registry: kind %q already registered", kind)
	}
	r.factories[kind] = f
	return nil
}

func (r *Registry) Create(kind string, config map[string]string) (Stage, error) {
	r.mu.RLock()
	f, ok := r.factories[kind]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("registry: unknown kind %q", kind)
	}
	return f(config)
}

func (r *Registry) Kinds() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.factories))
	for k := range r.factories {
		out = append(out, k)
	}
	return out
}

func (r *Registry) Has(kind string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.factories[kind]
	return ok
}

type FuncStage struct {
	name string
	deps []string
	fn   func(ctx context.Context, input *parse.Table) (*parse.Table, error)
}

func NewFuncStage(name string, deps []string, fn func(ctx context.Context, input *parse.Table) (*parse.Table, error)) *FuncStage {
	return &FuncStage{name: name, deps: deps, fn: fn}
}

func (s *FuncStage) Name() string           { return s.name }
func (s *FuncStage) Dependencies() []string { return s.deps }

func (s *FuncStage) Run(ctx context.Context, input *parse.Table) (*StageResult, error) {
	out, err := s.fn(ctx, input)
	if err != nil {
		return nil, err
	}
	return &StageResult{
		Table:    out,
		Metadata: map[string]string{"rows": fmt.Sprintf("%d", len(out.Rows))},
	}, nil
}

func PassthroughStage(name string, deps []string) *FuncStage {
	return NewFuncStage(name, deps, func(_ context.Context, t *parse.Table) (*parse.Table, error) {
		return t, nil
	})
}

func FilterStage(name, column string, deps []string) *FuncStage {
	return NewFuncStage(name, deps, func(_ context.Context, t *parse.Table) (*parse.Table, error) {
		ci := -1
		for i, h := range t.Header {
			if h == column {
				ci = i
				break
			}
		}
		if ci < 0 {
			return nil, fmt.Errorf("filter stage: column %q not found", column)
		}
		out := &parse.Table{Header: append([]string(nil), t.Header...)}
		for _, row := range t.Rows {
			if ci < len(row) && row[ci] != "" {
				out.Rows = append(out.Rows, append([]string(nil), row...))
			}
		}
		return out, nil
	})
}
