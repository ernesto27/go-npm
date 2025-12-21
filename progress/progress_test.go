package progress

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		validate func(t *testing.T, p *Progress)
	}{
		{
			name:    "Create progress with version",
			version: "1.0.0",
			validate: func(t *testing.T, p *Progress) {
				assert.NotNil(t, p, "Progress should not be nil")
				assert.NotNil(t, p.spinner, "Spinner should be initialized")
				assert.Equal(t, "1.0.0", p.version, "Version should be set")
				assert.NotNil(t, p.topLevel, "topLevel slice should be initialized")
				assert.Equal(t, 0, len(p.topLevel), "topLevel should be empty initially")
				assert.Equal(t, 0, p.totalCount, "totalCount should be 0 initially")
			},
		},
		{
			name:    "Create progress with empty version",
			version: "",
			validate: func(t *testing.T, p *Progress) {
				assert.NotNil(t, p, "Progress should not be nil")
				assert.Equal(t, "", p.version, "Version should be empty")
			},
		},
		{
			name:    "Create progress with dev version",
			version: "dev",
			validate: func(t *testing.T, p *Progress) {
				assert.NotNil(t, p, "Progress should not be nil")
				assert.Equal(t, "dev", p.version, "Version should be 'dev'")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := New(tc.version, false)
			tc.validate(t, p)
		})
	}
}

func TestAddTopLevel(t *testing.T) {
	testCases := []struct {
		name      string
		setupFunc func() *Progress
		packages  []PackageInfo
		validate  func(t *testing.T, p *Progress)
	}{
		{
			name: "Add single top-level package",
			setupFunc: func() *Progress {
				return New("1.0.0", false)
			},
			packages: []PackageInfo{
				{Name: "express", Version: "5.2.1"},
			},
			validate: func(t *testing.T, p *Progress) {
				assert.Equal(t, 1, len(p.topLevel), "Should have 1 package")
				assert.Equal(t, "express", p.topLevel[0].Name)
				assert.Equal(t, "5.2.1", p.topLevel[0].Version)
			},
		},
		{
			name: "Add multiple top-level packages",
			setupFunc: func() *Progress {
				return New("1.0.0", false)
			},
			packages: []PackageInfo{
				{Name: "express", Version: "5.2.1"},
				{Name: "lodash", Version: "4.17.21"},
				{Name: "react", Version: "18.2.0"},
			},
			validate: func(t *testing.T, p *Progress) {
				assert.Equal(t, 3, len(p.topLevel), "Should have 3 packages")
				assert.Equal(t, "express", p.topLevel[0].Name)
				assert.Equal(t, "lodash", p.topLevel[1].Name)
				assert.Equal(t, "react", p.topLevel[2].Name)
			},
		},
		{
			name: "Add scoped package",
			setupFunc: func() *Progress {
				return New("1.0.0", false)
			},
			packages: []PackageInfo{
				{Name: "@babel/core", Version: "7.28.5"},
			},
			validate: func(t *testing.T, p *Progress) {
				assert.Equal(t, 1, len(p.topLevel), "Should have 1 package")
				assert.Equal(t, "@babel/core", p.topLevel[0].Name)
				assert.Equal(t, "7.28.5", p.topLevel[0].Version)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.setupFunc()
			for _, pkg := range tc.packages {
				p.AddTopLevel(pkg.Name, pkg.Version)
			}
			tc.validate(t, p)
		})
	}
}

func TestIncrementCount(t *testing.T) {
	testCases := []struct {
		name          string
		setupFunc     func() *Progress
		incrementBy   int
		expectedCount int
	}{
		{
			name: "Increment once",
			setupFunc: func() *Progress {
				return New("1.0.0", false)
			},
			incrementBy:   1,
			expectedCount: 1,
		},
		{
			name: "Increment multiple times",
			setupFunc: func() *Progress {
				return New("1.0.0", false)
			},
			incrementBy:   5,
			expectedCount: 5,
		},
		{
			name: "Increment from non-zero",
			setupFunc: func() *Progress {
				p := New("1.0.0", false)
				p.totalCount = 10
				return p
			},
			incrementBy:   3,
			expectedCount: 13,
		},
		{
			name: "Increment many times (simulate large install)",
			setupFunc: func() *Progress {
				return New("1.0.0", false)
			},
			incrementBy:   149,
			expectedCount: 149,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := tc.setupFunc()
			for i := 0; i < tc.incrementBy; i++ {
				p.IncrementCount()
			}
			assert.Equal(t, tc.expectedCount, p.totalCount, "Total count should match expected")
		})
	}
}

func TestSetStatus(t *testing.T) {
	testCases := []struct {
		name     string
		message  string
		verbose  bool
		validate func(t *testing.T, p *Progress)
	}{
		{
			name:    "Set resolving status",
			message: "Resolving dependencies...",
			verbose: false,
			validate: func(t *testing.T, p *Progress) {
				assert.Equal(t, " Resolving dependencies...", p.spinner.Suffix)
			},
		},
		{
			name:    "Set fetching status",
			message: "Fetching express@5.2.1...",
			verbose: false,
			validate: func(t *testing.T, p *Progress) {
				assert.Equal(t, " Fetching express@5.2.1...", p.spinner.Suffix)
			},
		},
		{
			name:    "Set empty status",
			message: "",
			verbose: false,
			validate: func(t *testing.T, p *Progress) {
				assert.Equal(t, " ", p.spinner.Suffix)
			},
		},
		{
			name:    "Verbose true sets spinner suffix",
			message: "↓ express@5.2.1",
			verbose: true,
			validate: func(t *testing.T, p *Progress) {
				assert.Equal(t, " ↓ express@5.2.1", p.spinner.Suffix)
				assert.True(t, p.verbose)
			},
		},
		{
			name:    "Verbose false sets spinner suffix",
			message: "↓ lodash@4.17.21",
			verbose: false,
			validate: func(t *testing.T, p *Progress) {
				assert.Equal(t, " ↓ lodash@4.17.21", p.spinner.Suffix)
				assert.False(t, p.verbose)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := New("1.0.0", tc.verbose)
			p.SetStatus(tc.message)
			tc.validate(t, p)
		})
	}
}

func TestPackageInfo(t *testing.T) {
	testCases := []struct {
		name    string
		pkgInfo PackageInfo
	}{
		{
			name:    "Standard package",
			pkgInfo: PackageInfo{Name: "express", Version: "5.2.1"},
		},
		{
			name:    "Scoped package",
			pkgInfo: PackageInfo{Name: "@types/node", Version: "20.10.0"},
		},
		{
			name:    "Package with pre-release version",
			pkgInfo: PackageInfo{Name: "react", Version: "19.0.0-rc.1"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotEmpty(t, tc.pkgInfo.Name, "Name should not be empty")
			assert.NotEmpty(t, tc.pkgInfo.Version, "Version should not be empty")
		})
	}
}
