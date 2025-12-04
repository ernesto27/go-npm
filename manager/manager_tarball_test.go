package manager

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateUniqueTarballNameRealWorldScenarios(t *testing.T) {
	// Test real-world collision scenarios that caused bugs
	testCases := []struct {
		name               string
		packages           []struct{ name, version string }
		expectAllDifferent bool
		description        string
	}{
		{
			name: "Jest test suite with @jest/expect and expect",
			packages: []struct{ name, version string }{
				{"@jest/expect", "30.2.0"},
				{"expect", "30.2.0"},
				{"@jest/globals", "30.2.0"},
				{"jest-circus", "30.2.0"},
			},
			expectAllDifferent: true,
			description:        "Jest packages should all have unique tarball names",
		},
		{
			name: "Express with qs and @types/qs",
			packages: []struct{ name, version string }{
				{"express", "5.0.1"},
				{"qs", "6.14.0"},
				{"@types/qs", "6.14.0"},
				{"@types/express", "5.0.0"},
			},
			expectAllDifferent: true,
			description:        "Express and its dependencies should have unique tarball names",
		},
		{
			name: "TypeScript project with multiple @types packages",
			packages: []struct{ name, version string }{
				{"@types/node", "20.0.0"},
				{"@types/react", "18.0.0"},
				{"react", "18.0.0"},
				{"node", "1.0.0"},
			},
			expectAllDifferent: true,
			description:        "Type definitions should not collide with actual packages",
		},
		{
			name: "Babel packages with scoped and non-scoped variants",
			packages: []struct{ name, version string }{
				{"@babel/core", "7.25.3"},
				{"@babel/traverse", "7.25.3"},
				{"core", "1.0.0"},
				{"traverse", "0.6.8"},
			},
			expectAllDifferent: true,
			description:        "Scoped Babel packages should not collide with non-scoped packages",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filenames := make(map[string]string)

			for _, pkg := range tc.packages {
				filename := generateUniqueTarballName(pkg.name, pkg.version)

				if existing, exists := filenames[filename]; exists {
					if tc.expectAllDifferent {
						t.Errorf("Collision detected: %s and %s both produce filename %s",
							existing, pkg.name, filename)
					}
				} else {
					filenames[filename] = pkg.name
				}
			}

			if tc.expectAllDifferent {
				assert.Equal(t, len(tc.packages), len(filenames),
					"all packages should have unique tarball names: %s", tc.description)
			}
		})
	}
}
