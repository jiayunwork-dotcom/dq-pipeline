package pipeline

import (
	"fmt"
	"sort"
)

var ErrCycle = fmt.Errorf("pipeline: dependency cycle detected")

func topoSort(stages map[string]Stage) ([]string, error) {
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

	var queue []string
	for name, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, name)
		}
	}
	sort.Strings(queue)

	var order []string
	for len(queue) > 0 {
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
		return nil, ErrCycle
	}
	return order, nil
}

func ValidateDeps(stages map[string]Stage) error {
	_, err := topoSort(stages)
	return err
}

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

func Levels(stages map[string]Stage) ([][]string, error) {
	order, err := topoSort(stages)
	if err != nil {
		return nil, err
	}

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
