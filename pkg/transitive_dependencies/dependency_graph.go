package transitive_dependencies

import (
	"fmt"
	"sort"
	"strings"
)

// Dependency represents a node in the dependency graph
type Dependency struct {
	Purl    string
	Version string
}

// DependencyGraph represents a directed graph of dependencies.
type DependencyGraph struct {
	// The Dependency type works as a map key because it contains only comparable types.
	// More info: https://go.dev/blog/maps#key-types
	dependenciesOf map[Dependency][]Dependency
}

// NewDepGraph creates and initializes a new empty dependency graph
func NewDepGraph() *DependencyGraph {
	return &DependencyGraph{
		dependenciesOf: make(map[Dependency][]Dependency),
	}
}

// isRegisteredDependency checks if a dependency is already present in the graph
// Returns true if the dependency exists, false otherwise
func (dg *DependencyGraph) isRegisteredDependency(d Dependency) bool {
	_, exists := dg.dependenciesOf[d]
	return exists
}

// registerDependency adds a new dependency to the graph with an empty list of children
// This creates a node in the graph without any outgoing edges
func (dg *DependencyGraph) registerDependency(d Dependency) {
	dg.dependenciesOf[d] = []Dependency{}
}

// Connect establishes a dependency relationship between root and child
// If either dependency doesn't exist in the graph, it will be registered first
// If child is empty, only the root dependency is registered without creating an edge
func (dg *DependencyGraph) Connect(root Dependency, child Dependency) {

	if !dg.isRegisteredDependency(root) {
		dg.registerDependency(root)
	}

	//If child is empty, don't create an entry in the graph
	if (child == Dependency{}) {
		return
	}

	if !dg.isRegisteredDependency(child) {
		dg.registerDependency(child)
	}

	dg.dependenciesOf[root] = append(dg.dependenciesOf[root], child)
}

// String generates a string representation of the graph
// The output shows all dependencies and their relationships
// Each line follows the format: "<dependency> --> <child_dependency>"
// Dependencies with no children show "<dependency> --> null"
func (dg *DependencyGraph) String() string {
	var result strings.Builder
	deps := make([]*Dependency, 0, len(dg.dependenciesOf))
	for dep := range dg.dependenciesOf {
		deps = append(deps, &dep)
	}

	sort.Slice(deps, func(i, j int) bool {
		return string(deps[i].Purl) < string(deps[j].Purl)
	})

	for key, value := range dg.dependenciesOf {
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

// Flatten returns a slice containing all dependencies in the graph
// This provides a flat list of all unique dependencies without their relationships
func (dg *DependencyGraph) Flatten() []Dependency {
	purls := make([]Dependency, 0, len(dg.dependenciesOf))
	for key, _ := range dg.dependenciesOf {
		purls = append(purls, Dependency{Purl: key.Purl, Version: key.Version})
	}
	return purls
}

// GetDependenciesCount returns the total number of unique dependencies in the graph
func (dg *DependencyGraph) GetDependenciesCount() int {
	return len(dg.dependenciesOf)
}
