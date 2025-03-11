package transitive_dependencies

import (
	"testing"
)

// Tests were generated from Claude based on the BNF Grammar
// https://docs.npmjs.com/cli/v6/using-npm/semver#range-grammar
func TestPickFirstVersionFromNpmJsRange(t *testing.T) {
	tests := []struct {
		name        string
		requirement string
		ecosystem   string
		expected    string
		wantErr     bool
	}{
		// Simple Version Ranges
		{
			name:        "exact version",
			requirement: "1.2.3",
			ecosystem:   "npm",
			expected:    "1.2.3",
			wantErr:     false,
		},
		{
			name:        "explicit exact version",
			requirement: "=1.2.3",
			ecosystem:   "npm",
			expected:    "1.2.3",
			wantErr:     false,
		},
		{
			name:        "greater than version",
			requirement: ">1.2.3",
			ecosystem:   "npm",
			expected:    "1.2.3",
			wantErr:     false,
		},
		{
			name:        "greater than or equal to version",
			requirement: ">=1.2.3",
			ecosystem:   "npm",
			expected:    "1.2.3",
			wantErr:     false,
		},
		{
			name:        "less than version",
			requirement: "<1.2.3",
			ecosystem:   "npm",
			expected:    "1.2.3",
			wantErr:     false,
		},
		{
			name:        "less than or equal to version",
			requirement: "<=1.2.3",
			ecosystem:   "npm",
			expected:    "1.2.3",
			wantErr:     false,
		},

		// Partial Version Specifications
		{
			name:        "major version only",
			requirement: "1",
			ecosystem:   "npm",
			expected:    "1.0.0",
			wantErr:     false,
		},
		{
			name:        "major and minor version",
			requirement: "1.2",
			ecosystem:   "npm",
			expected:    "1.2.0",
			wantErr:     false,
		},
		{
			name:        "x wildcard for patch",
			requirement: "1.2.x",
			ecosystem:   "npm",
			expected:    "1.2.0",
			wantErr:     false,
		},
		{
			name:        "X wildcard for patch",
			requirement: "1.2.X",
			ecosystem:   "npm",
			expected:    "1.2.0",
			wantErr:     false,
		},
		{
			name:        "* wildcard for patch",
			requirement: "1.2.*",
			ecosystem:   "npm",
			expected:    "1.2.0",
			wantErr:     false,
		},
		{
			name:        "x wildcard for minor",
			requirement: "1.x",
			ecosystem:   "npm",
			expected:    "1.0.0",
			wantErr:     false,
		},
		{
			name:        "X wildcard for minor",
			requirement: "1.X",
			ecosystem:   "npm",
			expected:    "1.0.0",
			wantErr:     false,
		},
		{
			name:        "* wildcard for minor",
			requirement: "1.*",
			ecosystem:   "npm",
			expected:    "1.0.0",
			wantErr:     false,
		},

		// Hyphen Ranges
		{
			name:        "hyphen range",
			requirement: "1.2.3 - 2.3.4",
			ecosystem:   "npm",
			expected:    "1.2.3",
			wantErr:     false,
		},
		{
			name:        "hyphen range with partial start",
			requirement: "1.2 - 2.3.4",
			ecosystem:   "npm",
			expected:    "1.2.0",
			wantErr:     false,
		},
		{
			name:        "hyphen range with partial end",
			requirement: "1.2.3 - 2.3",
			ecosystem:   "npm",
			expected:    "1.2.3",
			wantErr:     false,
		},

		// Tilde Ranges
		{
			name:        "tilde range with full version",
			requirement: "~1.2.3",
			ecosystem:   "npm",
			expected:    "1.2.3",
			wantErr:     false,
		},
		{
			name:        "tilde range with partial version",
			requirement: "~1.2",
			ecosystem:   "npm",
			expected:    "1.2.0",
			wantErr:     false,
		},
		{
			name:        "tilde range with major only",
			requirement: "~1",
			ecosystem:   "npm",
			expected:    "1.0.0",
			wantErr:     false,
		},

		// Caret Ranges
		{
			name:        "caret range with full version",
			requirement: "^1.2.3",
			ecosystem:   "npm",
			expected:    "1.2.3",
			wantErr:     false,
		},
		{
			name:        "caret range with zero major",
			requirement: "^0.2.3",
			ecosystem:   "npm",
			expected:    "0.2.3",
			wantErr:     false,
		},
		{
			name:        "caret range with zero major and minor",
			requirement: "^0.0.3",
			ecosystem:   "npm",
			expected:    "0.0.3",
			wantErr:     false,
		},

		// Pre-release and Build Metadata
		{
			name:        "pre-release version",
			requirement: "1.2.3-beta",
			ecosystem:   "npm",
			expected:    "1.2.3-beta",
			wantErr:     false,
		},
		{
			name:        "build metadata",
			requirement: "1.2.3+build.123",
			ecosystem:   "npm",
			expected:    "1.2.3+build.123",
			wantErr:     false,
		},
		{
			name:        "pre-release and build metadata",
			requirement: "1.2.3-beta.2+build.456",
			ecosystem:   "npm",
			expected:    "1.2.3-beta.2+build.456",
			wantErr:     false,
		},

		// Wildcards
		{
			name:        "asterisk wildcard",
			requirement: "*",
			ecosystem:   "npm",
			expected:    "0.0.0",
			wantErr:     true,
		},
		{
			name:        "x wildcard",
			requirement: "x",
			ecosystem:   "npm",
			expected:    "0.0.0",
			wantErr:     true,
		},
		{
			name:        "X wildcard",
			requirement: "X",
			ecosystem:   "npm",
			expected:    "0.0.0",
			wantErr:     true,
		},

		// Combined Ranges (AND logic)
		{
			name:        "AND range",
			requirement: ">=1.2.3 <2.0.0",
			ecosystem:   "npm",
			expected:    "1.2.3",
			wantErr:     false,
		},
		{
			name:        "AND range with open lower bound",
			requirement: ">1.2.3 <=2.0.0",
			ecosystem:   "npm",
			expected:    "1.2.3",
			wantErr:     false,
		},

		// Logical OR Ranges
		{
			name:        "OR range with exact versions",
			requirement: "1.2.3 || 2.3.4",
			ecosystem:   "npm",
			expected:    "1.2.3",
			wantErr:     false,
		},
		{
			name:        "OR range with range and exact",
			requirement: ">=1.0.0 <1.5.0 || 2.0.0",
			ecosystem:   "npm",
			expected:    "1.0.0",
			wantErr:     false,
		},
		{
			name:        "OR range with caret and tilde",
			requirement: "^1.2.3 || ~2.3.4",
			ecosystem:   "npm",
			expected:    "1.2.3",
			wantErr:     false,
		},

		// Error cases
		{
			name:        "invalid version",
			requirement: "not.a.version",
			ecosystem:   "npm",
			expected:    "",
			wantErr:     true,
		},
		{
			name:        "invalid range",
			requirement: ">= garbage",
			ecosystem:   "npm",
			expected:    "",
			wantErr:     true,
		},
		{
			name:        "empty string",
			requirement: "",
			ecosystem:   "npm",
			expected:    "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := PickFirstVersionFromNpmJsRange(tt.requirement)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if version != tt.expected {
					t.Errorf("got %q, want %q", version, tt.expected)
				}
			}
		})
	}
}
