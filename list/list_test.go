package list

import (
	"strings"
	"testing"

	"github.com/ernesto27/go-npm/packagejson"
	"github.com/ernesto27/go-npm/utils"
)

func TestLister_Print(t *testing.T) {
	lock := &packagejson.PackageLock{
		Dependencies: map[string]string{
			"express": "4.18.2",
		},
		DevDependencies: map[string]string{
			"jest": "29.5.0",
		},
		Packages: map[string]packagejson.PackageItem{
			"node_modules/express": {
				Version: "4.18.2",
				Dependencies: map[string]string{
					"accepts": "1.3.8",
				},
			},
			"node_modules/jest": {
				Version: "29.5.0",
			},
			"node_modules/express/node_modules/accepts": {
				Version: "1.3.8",
			},
		},
	}

	tests := []struct {
		name        string
		showAll     bool
		projectName string
		version     string
		want        []string
	}{
		{
			name:        "Basic listing",
			showAll:     false,
			projectName: "test-project",
			version:     "1.0.0",
			want: []string{
				"test-project@1.0.0",
				"├── express@4.18.2",
				"└── jest@29.5.0 (dev)",
				"3 packages",
			},
		},
		{
			name:        "Listing with sub-dependencies",
			showAll:     true,
			projectName: "test-project",
			version:     "1.0.0",
			want: []string{
				"test-project@1.0.0",
				"├── express@4.18.2",
				"│   └── accepts@1.3.8",
				"└── jest@29.5.0 (dev)",
				"3 packages",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := New(lock, tt.projectName, tt.version)
			l.ShowAll = tt.showAll

			output := utils.CaptureStdout(func() {
				l.Print()
			})

			for _, w := range tt.want {
				if !strings.Contains(output, w) {
					t.Errorf("Print() output = %q, want to contain %q", output, w)
				}
			}
		})
	}
}
