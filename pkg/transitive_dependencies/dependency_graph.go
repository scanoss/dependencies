package trasitive_dependencies

import (
	"fmt"
	"sort"
	"strings"
)

type Status string

const (
	NOT_VISITED Status = "NOT_VISITED"
	VISITED     Status = "VISITED"
	FINISHED    Status = "FINISHED"
)

// Node represents a node in the dependency graph
type Dependency struct {
	Purl Purl
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

func (dp *DepGraph) getOrCreateDependencyByPurl(purl Purl) *Dependency {
	if dp.index[purl] == nil {
		dp.index[purl] = &Dependency{
			Purl: purl,
		}
	}

	return dp.index[purl]
}

// parent -> children
func (dp *DepGraph) Insert(dep Purl, transitive Purl) {

	parent := dp.getOrCreateDependencyByPurl(dep)       // scanoss
	child := dp.getOrCreateDependencyByPurl(transitive) // eslinter

	if dp.graph[child] == nil {
		dp.graph[child] = []*Dependency{}
	}

	if dp.graph[parent] == nil {
		dp.graph[parent] = []*Dependency{child}
	} else {
		dp.graph[parent] = append(dp.graph[parent], child)
	}

}
func (dp *DepGraph) String() string {
	var result strings.Builder

	result.WriteString("Dependency Graph:\n")

	// Crear un slice de dependencias para ordenarlas (opcional)
	deps := make([]*Dependency, 0, len(dp.graph))
	for dep := range dp.graph {
		deps = append(deps, dep)
	}

	// Opcional: ordenar las dependencias por Purl para una salida consistente
	sort.Slice(deps, func(i, j int) bool {
		return string(deps[i].Purl) < string(deps[j].Purl)
	})

	// Iterar sobre cada nodo en el grafo
	for key, value := range dp.graph {
		children := value

		// Saltar nodos sin dependencias si se desea
		if len(children) == 0 {
			result.WriteString(fmt.Sprintf("%s (Without dependencies)\n", key.Purl))
			continue
		}

		// Para cada nodo, imprimir sus dependencias
		for _, child := range children {
			result.WriteString(fmt.Sprintf("%s --> %s\n", key.Purl, child.Purl))
		}
	}

	return result.String()
}

func (dp *DepGraph) Flatten() []Purl {
	purls := make([]Purl, 0, len(dp.graph))
	for key, _ := range dp.graph {
		purls = append(purls, key.Purl)
	}
	return purls
}
