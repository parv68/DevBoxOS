package orchestrator

import (
	"fmt"
)

// Graph resolves service dependency ordering using topological sort.
type Graph struct {
	nodes      map[string][]string // node -> dependencies (prerequisites)
	dependents map[string][]string // node -> list of nodes that depend on it
	inDegree   map[string]int      // node -> number of prerequisites
}

// NewGraph creates a new dependency graph.
func NewGraph() *Graph {
	return &Graph{
		nodes:      make(map[string][]string),
		dependents: make(map[string][]string),
		inDegree:   make(map[string]int),
	}
}

// AddNode adds a service with its dependencies.
func (g *Graph) AddNode(name string, dependsOn []string) {
	g.nodes[name] = dependsOn
	g.inDegree[name] += len(dependsOn)

	for _, dep := range dependsOn {
		g.dependents[dep] = append(g.dependents[dep], name)
		if _, exists := g.inDegree[dep]; !exists {
			g.inDegree[dep] = 0
		}
	}
}

// Resolve returns services in startup order (dependencies first).
func (g *Graph) Resolve() ([]string, error) {
	// Kahn's algorithm for topological sort
	queue := make([]string, 0)
	for node, degree := range g.inDegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}

	var result []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)

		// Reduce in-degree for nodes that depend on this node
		for _, dependent := range g.dependents[node] {
			g.inDegree[dependent]--
			if g.inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(result) != len(g.inDegree) {
		return nil, fmt.Errorf("circular dependency detected: not all services could be resolved")
	}

	return result, nil
}

// Reverse returns services in shutdown order (dependents first).
func (g *Graph) Reverse() ([]string, error) {
	startOrder, err := g.Resolve()
	if err != nil {
		return nil, err
	}

	reversed := make([]string, len(startOrder))
	for i, v := range startOrder {
		reversed[len(startOrder)-1-i] = v
	}

	return reversed, nil
}
