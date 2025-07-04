package transdep

import (
	"testing"

	"github.com/package-url/packageurl-go"
)

// Tests were generated from Claude based on NPMJS versions BNF Grammar.
func TestPickFirstVersionFromRange(t *testing.T) {
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
			expected:    "",
			wantErr:     true,
		},
		{
			name:        "major and minor version",
			requirement: "1.2",
			ecosystem:   "npm",
			expected:    "",
			wantErr:     true,
		},
		{
			name:        "x wildcard for patch",
			requirement: "1.2.x",
			ecosystem:   "npm",
			expected:    "",
			wantErr:     true,
		},
		{
			name:        "X wildcard for patch",
			requirement: "1.2.X",
			ecosystem:   "npm",
			expected:    "",
			wantErr:     true,
		},
		{
			name:        "* wildcard for patch",
			requirement: "1.2.*",
			ecosystem:   "npm",
			expected:    "",
			wantErr:     true,
		},
		{
			name:        "x wildcard for minor",
			requirement: "1.x",
			ecosystem:   "npm",
			expected:    "",
			wantErr:     true,
		},
		{
			name:        "X wildcard for minor",
			requirement: "1.X",
			ecosystem:   "npm",
			expected:    "",
			wantErr:     true,
		},
		{
			name:        "* wildcard for minor",
			requirement: "1.*",
			ecosystem:   "npm",
			expected:    "",
			wantErr:     true,
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
			expected:    "2.3.4",
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
			wantErr:     true,
		},
		{
			name:        "tilde range with major only",
			requirement: "~1",
			ecosystem:   "npm",
			expected:    "1.0.0",
			wantErr:     true,
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
			expected:    "1.2.3",
			wantErr:     false,
		},
		{
			name:        "build metadata",
			requirement: "1.2.3+build.123",
			ecosystem:   "npm",
			expected:    "1.2.3",
			wantErr:     false,
		},
		{
			name:        "pre-release and build metadata",
			requirement: "1.2.3-beta.2+build.456",
			ecosystem:   "npm",
			expected:    "1.2.3",
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
			version, err := PickFirstVersionFromRange(tt.requirement)
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
func TestGetPurlFromPackageName(t *testing.T) {
	tests := []struct {
		name      string
		purlName  string
		version   string
		ecosystem string
		expected  string
		wantErr   bool
	}{
		{
			name:      "npmjs ecosystem",
			purlName:  "scanoss",
			version:   "1.2.0",
			ecosystem: "npm",
			expected:  "pkg:npm/scanoss@1.2.0",
			wantErr:   false,
		},
		{
			name:      "maven ecosystem",
			purlName:  "ai.databand/dbnd-api-deequ",
			version:   "1.2.0",
			ecosystem: "maven",
			expected:  "pkg:maven/ai.databand%2Fdbnd-api-deequ@1.2.0",
			wantErr:   false,
		},
		{
			name:      "ruby ecosystem",
			purlName:  "spree_repeat_order",
			version:   "2.1.4",
			ecosystem: "gem",
			expected:  "pkg:gem/spree_repeat_order@2.1.4",
			wantErr:   false,
		},
		{
			name:      "crates ecosystem",
			purlName:  "tecla_client",
			version:   "1.0.1",
			ecosystem: "crates",
			expected:  "pkg:crates/tecla_client@1.0.1",
			wantErr:   false,
		},
		{
			name:      "composer ecosystem",
			purlName:  "tecla_client",
			version:   "1.0.1",
			ecosystem: "composer",
			expected:  "pkg:composer/tecla_client@1.0.1",
			wantErr:   false,
		},
		{
			name:      "composer ecosystem",
			purlName:  "php-extended/php-checksum-interface",
			version:   "8.0.5",
			ecosystem: "composer",
			expected:  "pkg:composer/php-extended%2Fphp-checksum-interface@8.0.5",
			wantErr:   false,
		},
		{
			name:      "empty ecosystem",
			purlName:  "scanoss",
			version:   "1.2.0",
			ecosystem: "",
			expected:  "pkg:npm/scanoss@1.2.0",
			wantErr:   true,
		},
		{
			name:      "invalid ecosystem",
			purlName:  "scanoss",
			version:   "1.2.0",
			ecosystem: "npn",
			expected:  "pkg:npm/scanoss@1.2.0",
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			purl, err := GetPurlFromPurlName(tt.purlName, tt.version, tt.ecosystem)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %s, got nil\n", tt.name)
				}
				return
			}
			if purl.String() != tt.expected {
				t.Errorf("got %q, want %q", purl.String(), tt.expected)
			}
		})
	}
}

func TestExtractPackageIdentifierFromPurl(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "valid package identifier",
			input:    "pkg:npm/scanoss",
			expected: "scanoss",
			wantErr:  false,
		},
		{
			name:     "invalid package identifier",
			input:    "p:npm/scanoss",
			expected: "pkg:maven/ai.databand%2Fdbnd-api-deequ@1.2.0",
			wantErr:  true,
		},
		{
			name:     "get package identifier for maven",
			input:    "pkg:maven/ai.databand%2Fdbnd-api-deequ",
			expected: "ai.databand/dbnd-api-deequ",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packageIdentifier, err := ExtractPackageIdentifierFromPurl(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %s, got nil\n", tt.name)
				}
				return
			}
			if packageIdentifier != tt.expected {
				t.Errorf("got %q, want %q", packageIdentifier, tt.expected)
			}
		})
	}
}

func TestGetPurlWithoutVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    *packageurl.PackageURL
		expected string
		wantErr  bool
	}{
		{
			name: "",
			input: packageurl.NewPackageURL(
				"npm",     // type
				"",        // namespace
				"scanoss", // name
				"1.2.0",   // version
				nil,       // qualifiers
				"",        // subpath
			),
			expected: "pkg:npm/scanoss",
			wantErr:  false,
		},
		{
			name: "",
			input: packageurl.NewPackageURL(
				"npm",     // type
				"",        // namespace
				"scanoss", // name
				"",        // version
				nil,       // qualifiers
				"",        // subpath
			),
			expected: "pkg:npm/scanoss",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			purl, err := GetPurlWithoutVersion(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %s, got nil\n", tt.name)
				}
			} else {
				if purl != tt.expected {
					t.Errorf("got %q, want %q", purl, tt.expected)
				}
			}
		})
	}
}

func TestGetMaxLimit(t *testing.T) {
	tests := []struct {
		name          string
		limit         *int
		defaultLimit  int
		maxLimit      int
		expectedLimit int
	}{
		{
			name:          "Should use default limit when limit is nil",
			limit:         nil,
			defaultLimit:  10,
			maxLimit:      10,
			expectedLimit: 10,
		},
		{
			name:          "Should use max limit when limit set exceeds it",
			limit:         func() *int { i := 13; return &i }(), // Limit = 13
			defaultLimit:  10,
			maxLimit:      10,
			expectedLimit: 10,
		},
		{
			name:          "Should use default limit when limit is zero",
			limit:         func() *int { i := 0; return &i }(),
			defaultLimit:  5,
			maxLimit:      15,
			expectedLimit: 5,
		},
		{
			name:          "Should use specified limit when within valid range",
			limit:         func() *int { i := 8; return &i }(),
			defaultLimit:  5,
			maxLimit:      15,
			expectedLimit: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := GetMaxLimit(tt.maxLimit, tt.defaultLimit, tt.limit)
			if resp != tt.expectedLimit {
				t.Errorf("got %d, want %d", resp, tt.expectedLimit)
			}
		})
	}
}
