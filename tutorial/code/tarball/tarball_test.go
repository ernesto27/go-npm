package tarball

import (
	"go-npm/manifest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownloadTarball_Download(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (string, manifest.NPMPackage)
		expectError bool
		validate    func(t *testing.T, tb *Tarball, version string, npmPackage manifest.NPMPackage, err error)
	}{
		{
			name: "Download express tarball successfully",
			setupFunc: func(t *testing.T) (string, manifest.NPMPackage) {
				version := "4.18.2"
				url := "https://registry.npmjs.org/express/-/express-4.18.2.tgz"
				pkg := manifest.NPMPackage{
					Name: "express",
					Versions: map[string]manifest.Version{
						version: {
							Dist: manifest.Dist{
								Tarball: url,
							},
						},
					},
				}
				return version, pkg
			},
			expectError: false,
			validate: func(t *testing.T, tb *Tarball, version string, npmPackage manifest.NPMPackage, err error) {
				assert.NoError(t, err, "Download should succeed")

				expectedFile := filepath.Join(tb.TarballPath, "express-4.18.2.tgz")
				info, statErr := os.Stat(expectedFile)
				assert.NoError(t, statErr, "Tarball file should exist")
				assert.Greater(t, info.Size(), int64(0), "File should not be empty")
			},
		},
		{
			name: "Error with invalid tarball URL",
			setupFunc: func(t *testing.T) (string, manifest.NPMPackage) {
				version := "1.0.0"
				url := "https://registry.npmjs.org/invalid-package-12345678/-/invalid-package-12345678-1.0.0.tgz"
				pkg := manifest.NPMPackage{
					Name: "invalid-package-12345678",
					Versions: map[string]manifest.Version{
						version: {
							Dist: manifest.Dist{
								Tarball: url,
							},
						},
					},
				}
				return version, pkg
			},
			expectError: true,
			validate: func(t *testing.T, tb *Tarball, version string, npmPackage manifest.NPMPackage, err error) {
				assert.Error(t, err, "Should return error for non-existent package")
				assert.Contains(t, err.Error(), "HTTP error", "Error should indicate HTTP status problem")

				expectedFile := filepath.Join(tb.TarballPath, "invalid-package-12345678-1.0.0.tgz")
				info, statErr := os.Stat(expectedFile)
				if statErr == nil {
					assert.Equal(t, int64(0), info.Size(), "File should be empty or not exist")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			version, pkg := tc.setupFunc(t)
			tarball := NewTarball()
			_, err := tarball.Download(version, pkg)

			if tc.expectError {
				assert.Error(t, err, "Expected an error")
			} else {
				assert.NoError(t, err, "Expected no error")
			}

			tc.validate(t, tarball, version, pkg, err)
		})
	}
}
