package transdep

import (
	"strings"
	"testing"
)

func TestInsertDependency(t *testing.T) {
	tests := []struct {
		name       string
		parent     Dependency
		transitive []Dependency
		expected   []Dependency
		wantErr    bool
	}{
		{
			name: "Connect parent dependency with transitive dependencies",
			parent: Dependency{
				Purl:    "pkg:/scanoss/scanoss.js",
				Version: "0.15.4",
			},
			transitive: []Dependency{
				{
					Purl:    "pkg:npm/tar",
					Version: "6.1.11",
				},
				{
					Purl:    "pkg:npm/typescript",
					Version: "4.0.2",
				},
				{
					Purl:    "pkg:npm/mocha",
					Version: "5.0.0",
				},
			},
			expected: []Dependency{
				{
					Purl:    "pkg:/scanoss/scanoss.js",
					Version: "0.15.5",
				},
				{
					Purl:    "pkg:npm/tar",
					Version: "6.1.11",
				},
				{
					Purl:    "pkg:npm/typescript",
					Version: "4.0.2",
				},
				{
					Purl:    "pkg:npm/mocha",
					Version: "5.0.0",
				},
			},
			wantErr: false,
		},
		{
			name: "Connect parent dependency with empty transitive dependencies",
			parent: Dependency{
				Purl:    "pkg:/scanoss/scanoss.js",
				Version: "0.15.4",
			},
			transitive: []Dependency{},
			expected: []Dependency{
				{
					Purl:    "pkg:/scanoss/scanoss.js",
					Version: "0.15.4",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := NewDepGraph()
			if len(tt.transitive) > 0 {
				for _, dependency := range tt.transitive {
					graph.Connect(tt.parent, dependency)
				}
			} else {
				graph.Connect(tt.parent, Dependency{})
			}

			if len(tt.expected) != len(graph.Flatten()) {
				t.Errorf("expected %v, but got %v", tt.expected, graph.Flatten())
			}
		})
	}
}

func TestGetFlattenGraph(t *testing.T) {
	graph := NewDepGraph()
	graph.Connect(
		Dependency{Purl: "pkg:/scanoss/scanoss.js", Version: "0.15.4"},
		Dependency{Purl: "pkg:npm/typescript", Version: "0.10.0"},
	)

	t.Run("Graph contains expected number of dependencies", func(t *testing.T) {
		if len(graph.Flatten()) != 2 {
			t.Errorf("expected 2 dependencies, but got %v", graph.Flatten())
		}
	})
	requiredDependencies := make(map[string]struct{})
	requiredDependencies["pkg:/scanoss/scanoss.js@0.15.4"] = struct{}{}
	requiredDependencies["pkg:npm/typescript@0.10.0"] = struct{}{}

	// Test cases
	t.Run("Graph contains expected dependencies", func(t *testing.T) {
		for _, dependency := range graph.Flatten() {
			key := dependency.Purl + "@" + dependency.Version
			if _, notExists := requiredDependencies[key]; !notExists {
				t.Errorf("dependency %s is required", key)
			}
		}
	})
}

func TestGetGraphString(t *testing.T) {
	// Initialize the dependency adjacencyList
	graph := NewDepGraph()

	// Connect dependencies
	parent := Dependency{Purl: "pkg:/scanoss/scanoss.js", Version: "0.15.4"}
	child := Dependency{Purl: "pkg:npm/typescript", Version: "0.10.0"}
	graph.Connect(parent, child)
	result := graph.String()
	// Check that both expected lines appear somewhere in the result
	if !strings.Contains(result, "pkg:npm/typescript --> null") {
		t.Errorf("Result missing 'pkg:npm/typescript --> null'")
	}

	if !strings.Contains(result, "pkg:/scanoss/scanoss.js --> pkg:npm/typescript") {
		t.Errorf("Result missing 'pkg:/scanoss/scanoss.js --> pkg:npm/typescript'")
	}
}
