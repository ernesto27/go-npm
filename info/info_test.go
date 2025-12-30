package info

import (
	"testing"

	"github.com/ernesto27/go-npm/manifest"
	"github.com/stretchr/testify/assert"
)

func TestExtractLicense(t *testing.T) {
	testCases := []struct {
		name       string
		pkgLicense any
		verLicense any
		expected   string
	}{
		{
			name:       "string license from version",
			pkgLicense: nil,
			verLicense: "MIT",
			expected:   "MIT",
		},
		{
			name:       "string license from package",
			pkgLicense: "Apache-2.0",
			verLicense: nil,
			expected:   "Apache-2.0",
		},
		{
			name:       "version license takes precedence",
			pkgLicense: "GPL",
			verLicense: "MIT",
			expected:   "MIT",
		},
		{
			name:       "object license with type field",
			pkgLicense: map[string]interface{}{"type": "ISC", "url": "https://example.com"},
			verLicense: nil,
			expected:   "ISC",
		},
		{
			name:       "empty string returns Unknown",
			pkgLicense: "",
			verLicense: "",
			expected:   "Unknown",
		},
		{
			name:       "nil returns Unknown",
			pkgLicense: nil,
			verLicense: nil,
			expected:   "Unknown",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractLicense(tc.pkgLicense, tc.verLicense)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractString(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "valid string",
			input:    "https://example.com",
			expected: "https://example.com",
		},
		{
			name:     "nil returns empty",
			input:    nil,
			expected: "",
		},
		{
			name:     "non-string returns empty",
			input:    123,
			expected: "",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractString(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractKeywords(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		expected []string
	}{
		{
			name:     "interface slice",
			input:    []interface{}{"react", "javascript", "ui"},
			expected: []string{"react", "javascript", "ui"},
		},
		{
			name:     "string slice",
			input:    []string{"node", "npm"},
			expected: []string{"node", "npm"},
		},
		{
			name:     "nil returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty slice",
			input:    []interface{}{},
			expected: []string{},
		},
		{
			name:     "mixed types filters non-strings",
			input:    []interface{}{"valid", 123, "also-valid"},
			expected: []string{"valid", "also-valid"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractKeywords(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractMaintainers(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		expected []manifest.Maintainer
	}{
		{
			name: "valid maintainers",
			input: []interface{}{
				map[string]interface{}{"name": "John", "email": "john@example.com"},
				map[string]interface{}{"name": "Jane", "email": "jane@example.com"},
			},
			expected: []manifest.Maintainer{
				{Name: "John", Email: "john@example.com"},
				{Name: "Jane", Email: "jane@example.com"},
			},
		},
		{
			name: "maintainer without email",
			input: []interface{}{
				map[string]interface{}{"name": "John"},
			},
			expected: []manifest.Maintainer{
				{Name: "John", Email: ""},
			},
		},
		{
			name: "skip entries without name",
			input: []interface{}{
				map[string]interface{}{"email": "no-name@example.com"},
				map[string]interface{}{"name": "Valid", "email": "valid@example.com"},
			},
			expected: []manifest.Maintainer{
				{Name: "Valid", Email: "valid@example.com"},
			},
		},
		{
			name:     "nil returns nil",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty slice",
			input:    []interface{}{},
			expected: []manifest.Maintainer{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractMaintainers(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestFormatBytes(t *testing.T) {
	testCases := []struct {
		name     string
		bytes    int
		expected string
	}{
		{
			name:     "bytes",
			bytes:    500,
			expected: "500 B",
		},
		{
			name:     "kilobytes",
			bytes:    1024,
			expected: "1.00 KB",
		},
		{
			name:     "kilobytes with decimals",
			bytes:    1536,
			expected: "1.50 KB",
		},
		{
			name:     "megabytes",
			bytes:    1048576,
			expected: "1.00 MB",
		},
		{
			name:     "large package",
			bytes:    171622,
			expected: "167.60 KB",
		},
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatBytes(tc.bytes)
			assert.Equal(t, tc.expected, result)
		})
	}
}
