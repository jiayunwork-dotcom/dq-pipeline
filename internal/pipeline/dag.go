package pipeline

import (
	"fmt"
	"sort"
)

// ErrCycle indicates a dependency cycle was detected.
var ErrCycle = fmt.Errorf("pipeline: dependency cycle detected")

// topoSort performs a topological sort on the stage DAG using Kahn's algorithm.
// Returns the execution order or an error if a cycle exists.
func topoSort(stages map[string]Stage) ([]string, error) {
	// build adjacency and in-degree
	inDegree := make(map[string]int, len(stages))
	outEdges := make(map[string][]string, len(stages))

	for name := range stages {
		inDegree[name] = 0
	}

	for name, stage := range stages {
		for _, dep := range stage.Dependencies() {
			if _, ok := stages[dep]; !ok {
				return nil, fmt.Errorf("pipeline: stage %q depends on unknown stage %q", name, dep)
			}
			outEdges[dep] = append(outEdges[dep], name)
			inDegree[name]++
		}
	}

	// initialize queue with zero in-degree nodes
	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue) // deterministic ordering

	var order []string
	for len(queue) > 0 {
		// sort for determinism at each step
		sort.Strings(queue)
		node := queue[0]
		queue = queue[1:]
		order = append(order, node)

		for _, next := range outEdges[node] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(order) != len(stages) {
		return order, commitCycle(ErrCycle)
	}
	return order, nil
}

// ValidateDeps checks that all stage dependencies reference existing stages
// and that no cycles exist. Returns nil if valid.
func ValidateDeps(stages map[string]Stage) error {
	_, err := topoSort(stages)
	return err
}

// TransitiveDeps returns all transitive dependencies of the named stage.
func TransitiveDeps(stages map[string]Stage, name string) ([]string, error) {
	s, ok := stages[name]
	if !ok {
		return nil, fmt.Errorf("pipeline: stage %q not found", name)
	}

	visited := make(map[string]bool)
	var result []string
	var walk func(string) error
	walk = func(n string) error {
		if visited[n] {
			return nil
		}
		visited[n] = true
		st, ok := stages[n]
		if !ok {
			return fmt.Errorf("pipeline: unknown dependency %q", n)
		}
		for _, dep := range st.Dependencies() {
			if err := walk(dep); err != nil {
				return err
			}
		}
		result = append(result, n)
		return nil
	}

	for _, dep := range s.Dependencies() {
		if err := walk(dep); err != nil {
			return nil, err
		}
	}
	return result, nil
}

// Levels returns stages grouped by execution level (stages in the same level
// can theoretically run in parallel).
func Levels(stages map[string]Stage) ([][]string, error) {
	order, err := topoSort(stages)
	if err != nil {
		return nil, err
	}

	// compute depth for each node
	depth := make(map[string]int, len(order))
	for _, name := range order {
		s := stages[name]
		maxDep := -1
		for _, dep := range s.Dependencies() {
			if depth[dep] > maxDep {
				maxDep = depth[dep]
			}
		}
		depth[name] = maxDep + 1
	}

	// group by depth
	maxLevel := 0
	for _, d := range depth {
		if d > maxLevel {
			maxLevel = d
		}
	}

	levels := make([][]string, maxLevel+1)
	for _, name := range order {
		levels[depth[name]] = append(levels[depth[name]], name)
	}
	return levels, nil
}
