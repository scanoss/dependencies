package transitive_dependencies

import (
	"fmt"
	"sort"
	"strings"
)

type Status string

// Node represents a node in the dependency graph
type Dependency struct {
	Purl    string
	Version string
}

type Purl string

// DepGraph represents a directed acyclic graph of dependencies
type DepGraph struct {
	index map[Purl]*Dependency

	graph map[*Dependency][]*Dependency

	visited map[string]Status
}

func NewDepGraph() *DepGraph {
	return &DepGraph{
		graph:   make(map[*Dependency][]*Dependency),
		index:   make(map[Purl]*Dependency),
		visited: make(map[string]Status),
	}
}

func (dp *DepGraph) getOrCreateDependencyByPurl(d Dependency) *Dependency {
	key := Purl(d.Purl + "@" + d.Version)
	if dp.index[key] == nil {
		dp.index[key] = &Dependency{
			Purl:    d.Purl,
			Version: d.Version,
		}
	}

	return dp.index[key]
}

func (dp *DepGraph) Insert(dep Dependency, transitive Dependency) {
	// Get or create the parent dependency node
	parent := dp.getOrCreateDependencyByPurl(dep)

	// Check if transitive dependency is empty
	isEmptyTransitive := transitive == Dependency{}

	// Initialize parent's adjacency list if needed
	if dp.graph[parent] == nil {
		dp.graph[parent] = []*Dependency{}
	}

	// If transitive is empty, we're just ensuring parent exists in the graph
	if isEmptyTransitive {
		return
	}

	// Get or create the child dependency node
	child := dp.getOrCreateDependencyByPurl(transitive)

	if dp.graph[child] == nil {
		dp.graph[child] = []*Dependency{}
	}
	// Add child to parent's adjacency list
	dp.graph[parent] = append(dp.graph[parent], child)
}

func (dp *DepGraph) String() string {
	var result strings.Builder
	deps := make([]*Dependency, 0, len(dp.graph))
	for dep := range dp.graph {
		deps = append(deps, dep)
	}

	sort.Slice(deps, func(i, j int) bool {
		return string(deps[i].Purl) < string(deps[j].Purl)
	})

	for key, value := range dp.graph {
		children := value

		if len(children) == 0 {
			result.WriteString(fmt.Sprintf("%s --> null\n", key.Purl))
			continue
		}

		for _, child := range children {
			result.WriteString(fmt.Sprintf("%s --> %s\n", key.Purl, child.Purl))
		}
	}

	return result.String()
}

func (dp *DepGraph) Flatten() []Dependency {
	purls := make([]Dependency, 0, len(dp.graph))
	for key, _ := range dp.graph {
		purls = append(purls, Dependency{Purl: key.Purl, Version: key.Version})
	}
	return purls
}

func (dp *DepGraph) GetDependenciesCount() int {
	return len(dp.graph)
}
