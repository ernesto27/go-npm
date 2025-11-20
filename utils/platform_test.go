package utils

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCurrentOS(t *testing.T) {
	result := GetCurrentOS()

	// Verify it returns a non-empty string
	assert.NotEmpty(t, result)

	// Verify it returns expected value based on runtime.GOOS
	switch runtime.GOOS {
	case "windows":
		assert.Equal(t, "win32", result)
	case "darwin":
		assert.Equal(t, "darwin", result)
	case "linux":
		assert.Equal(t, "linux", result)
	case "freebsd":
		assert.Equal(t, "freebsd", result)
	case "openbsd":
		assert.Equal(t, "openbsd", result)
	case "netbsd":
		assert.Equal(t, "netbsd", result)
	case "aix":
		assert.Equal(t, "aix", result)
	case "solaris":
		assert.Equal(t, "sunos", result)
	default:
		assert.Equal(t, runtime.GOOS, result)
	}
}

func TestGetCurrentCPU(t *testing.T) {
	result := GetCurrentCPU()

	// Verify it returns a non-empty string
	assert.NotEmpty(t, result)

	// Verify it returns expected value based on runtime.GOARCH
	switch runtime.GOARCH {
	case "amd64":
		assert.Equal(t, "x64", result)
	case "386":
		assert.Equal(t, "ia32", result)
	case "arm64":
		assert.Equal(t, "arm64", result)
	case "arm":
		assert.Equal(t, "arm", result)
	case "ppc64":
		assert.Equal(t, "ppc64", result)
	case "ppc64le":
		assert.Equal(t, "ppc64", result)
	case "s390x":
		assert.Equal(t, "s390x", result)
	case "mips":
		assert.Equal(t, "mips", result)
	case "mipsle":
		assert.Equal(t, "mipsel", result)
	case "mips64":
		assert.Equal(t, "mips64", result)
	case "mips64le":
		assert.Equal(t, "mips64el", result)
	default:
		assert.Equal(t, runtime.GOARCH, result)
	}
}

func TestIsCompatiblePlatform(t *testing.T) {
	currentOS := GetCurrentOS()
	currentCPU := GetCurrentCPU()

	testCases := []struct {
		name           string
		osConstraints  []string
		cpuConstraints []string
		expected       bool
	}{
		{
			name:           "no constraints should be compatible",
			osConstraints:  []string{},
			cpuConstraints: []string{},
			expected:       true,
		},
		{
			name:           "nil constraints should be compatible",
			osConstraints:  nil,
			cpuConstraints: nil,
			expected:       true,
		},
		{
			name:           "matching OS constraint",
			osConstraints:  []string{currentOS},
			cpuConstraints: []string{},
			expected:       true,
		},
		{
			name:           "non-matching OS constraint",
			osConstraints:  []string{"nonexistent-os"},
			cpuConstraints: []string{},
			expected:       false,
		},
		{
			name:           "negated OS constraint - excluded",
			osConstraints:  []string{"!" + currentOS},
			cpuConstraints: []string{},
			expected:       false,
		},
		{
			name:           "negated OS constraint - not excluded",
			osConstraints:  []string{"!nonexistent-os"},
			cpuConstraints: []string{},
			expected:       true,
		},
		{
			name:           "matching CPU constraint",
			osConstraints:  []string{},
			cpuConstraints: []string{currentCPU},
			expected:       true,
		},
		{
			name:           "non-matching CPU constraint",
			osConstraints:  []string{},
			cpuConstraints: []string{"nonexistent-cpu"},
			expected:       false,
		},
		{
			name:           "both OS and CPU matching",
			osConstraints:  []string{currentOS},
			cpuConstraints: []string{currentCPU},
			expected:       true,
		},
		{
			name:           "OS matching but CPU not matching",
			osConstraints:  []string{currentOS},
			cpuConstraints: []string{"nonexistent-cpu"},
			expected:       false,
		},
		{
			name:           "OS not matching but CPU matching",
			osConstraints:  []string{"nonexistent-os"},
			cpuConstraints: []string{currentCPU},
			expected:       false,
		},
		{
			name:           "multiple OS constraints with match",
			osConstraints:  []string{"win32", "darwin", currentOS},
			cpuConstraints: []string{},
			expected:       true,
		},
		{
			name:           "multiple OS constraints without match",
			osConstraints:  []string{"os1", "os2", "os3"},
			cpuConstraints: []string{},
			expected:       false,
		},
		{
			name:           "mixed positive and negative OS constraints - excluded",
			osConstraints:  []string{"win32", "!" + currentOS},
			cpuConstraints: []string{},
			expected:       false,
		},
		{
			name:           "mixed positive and negative OS constraints - not excluded but not included",
			osConstraints:  []string{"win32", "!darwin"},
			cpuConstraints: []string{},
			expected:       currentOS == "win32",
		},
		{
			name:           "only negative constraints - multiple exclusions",
			osConstraints:  []string{"!win32", "!darwin"},
			cpuConstraints: []string{},
			expected:       currentOS != "win32" && currentOS != "darwin",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := IsCompatiblePlatform(tc.osConstraints, tc.cpuConstraints)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCheckConstraint(t *testing.T) {
	testCases := []struct {
		name        string
		current     string
		constraints []string
		expected    bool
	}{
		{
			name:        "exact match",
			current:     "linux",
			constraints: []string{"linux"},
			expected:    true,
		},
		{
			name:        "no match in positive constraints",
			current:     "linux",
			constraints: []string{"win32", "darwin"},
			expected:    false,
		},
		{
			name:        "match in multiple positive constraints",
			current:     "darwin",
			constraints: []string{"linux", "darwin", "win32"},
			expected:    true,
		},
		{
			name:        "negated constraint - excluded",
			current:     "win32",
			constraints: []string{"!win32"},
			expected:    false,
		},
		{
			name:        "negated constraint - not excluded",
			current:     "linux",
			constraints: []string{"!win32"},
			expected:    true,
		},
		{
			name:        "multiple negated constraints - excluded by one",
			current:     "darwin",
			constraints: []string{"!win32", "!darwin"},
			expected:    false,
		},
		{
			name:        "multiple negated constraints - not excluded by any",
			current:     "linux",
			constraints: []string{"!win32", "!darwin"},
			expected:    true,
		},
		{
			name:        "mixed positive and negative - positive match",
			current:     "linux",
			constraints: []string{"linux", "!win32"},
			expected:    true,
		},
		{
			name:        "mixed positive and negative - negative match (excluded)",
			current:     "win32",
			constraints: []string{"linux", "!win32"},
			expected:    false,
		},
		{
			name:        "mixed positive and negative - no positive match",
			current:     "freebsd",
			constraints: []string{"linux", "!win32"},
			expected:    false,
		},
		{
			name:        "empty constraints",
			current:     "linux",
			constraints: []string{},
			expected:    true,
		},
		{
			name:        "only positive constraints with match",
			current:     "x64",
			constraints: []string{"x64", "arm64"},
			expected:    true,
		},
		{
			name:        "only positive constraints without match",
			current:     "ia32",
			constraints: []string{"x64", "arm64"},
			expected:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := checkConstraint(tc.current, tc.constraints)
			assert.Equal(t, tc.expected, result)
		})
	}
}
