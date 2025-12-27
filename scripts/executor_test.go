package scripts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExecute(t *testing.T) {
	testCases := []struct {
		name      string
		setupFunc func() (string, *ScriptExecutor)
		script    string
		validate  func(t *testing.T, err error)
	}{
		{
			name: "empty script returns nil",
			setupFunc: func() (string, *ScriptExecutor) {
				dir := t.TempDir()
				return dir, NewScriptExecutor(dir)
			},
			script: "",
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "simple echo succeeds",
			setupFunc: func() (string, *ScriptExecutor) {
				dir := t.TempDir()
				return dir, NewScriptExecutor(dir)
			},
			script: "echo hello",
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "failing command returns error",
			setupFunc: func() (string, *ScriptExecutor) {
				dir := t.TempDir()
				return dir, NewScriptExecutor(dir)
			},
			script: "exit 1",
			validate: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "script execution failed")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir, executor := tc.setupFunc()
			err := executor.Execute(tc.script, dir, "test-pkg", "1.0.0", "postinstall")
			tc.validate(t, err)
		})
	}
}

func TestExecute_CreatesFile(t *testing.T) {
	testCases := []struct {
		name      string
		setupFunc func() (string, *ScriptExecutor, string)
		validate  func(t *testing.T, markerFile string, err error)
	}{
		{
			name: "script creates marker file",
			setupFunc: func() (string, *ScriptExecutor, string) {
				dir := t.TempDir()
				markerFile := filepath.Join(dir, "marker.txt")
				return dir, NewScriptExecutor(dir), markerFile
			},
			validate: func(t *testing.T, markerFile string, err error) {
				assert.NoError(t, err)
				_, statErr := os.Stat(markerFile)
				assert.NoError(t, statErr, "Marker file should exist")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir, executor, markerFile := tc.setupFunc()
			script := "echo test > " + markerFile
			err := executor.Execute(script, dir, "test-pkg", "1.0.0", "postinstall")
			tc.validate(t, markerFile, err)
		})
	}
}

func TestBuildEnvironment(t *testing.T) {
	testCases := []struct {
		name       string
		setupFunc  func() *ScriptExecutor
		pkgName    string
		pkgVersion string
		event      string
		validate   func(t *testing.T, env []string)
	}{
		{
			name: "contains npm lifecycle variables",
			setupFunc: func() *ScriptExecutor {
				return NewScriptExecutor("/path/to/node_modules")
			},
			pkgName:    "my-package",
			pkgVersion: "2.0.0",
			event:      "postinstall",
			validate: func(t *testing.T, env []string) {
				var hasEvent, hasName, hasVersion, hasPath bool
				for _, e := range env {
					if e == "npm_lifecycle_event=postinstall" {
						hasEvent = true
					}
					if e == "npm_package_name=my-package" {
						hasName = true
					}
					if e == "npm_package_version=2.0.0" {
						hasVersion = true
					}
					if strings.HasPrefix(e, "PATH=") && strings.Contains(e, ".bin") {
						hasPath = true
					}
				}
				assert.True(t, hasEvent, "should have npm_lifecycle_event")
				assert.True(t, hasName, "should have npm_package_name")
				assert.True(t, hasVersion, "should have npm_package_version")
				assert.True(t, hasPath, "should have .bin in PATH")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			executor := tc.setupFunc()
			env := executor.buildEnvironment(tc.pkgName, tc.pkgVersion, tc.event)
			tc.validate(t, env)
		})
	}
}

func TestSetEnv(t *testing.T) {
	testCases := []struct {
		name      string
		setupFunc func() *ScriptExecutor
		env       []string
		key       string
		value     string
		validate  func(t *testing.T, result []string)
	}{
		{
			name: "add new env var",
			setupFunc: func() *ScriptExecutor {
				return NewScriptExecutor("/path")
			},
			env:   []string{"FOO=bar"},
			key:   "BAZ",
			value: "qux",
			validate: func(t *testing.T, result []string) {
				assert.Equal(t, []string{"FOO=bar", "BAZ=qux"}, result)
			},
		},
		{
			name: "replace existing env var",
			setupFunc: func() *ScriptExecutor {
				return NewScriptExecutor("/path")
			},
			env:   []string{"FOO=bar", "BAZ=old"},
			key:   "BAZ",
			value: "new",
			validate: func(t *testing.T, result []string) {
				assert.Equal(t, []string{"FOO=bar", "BAZ=new"}, result)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			executor := tc.setupFunc()
			result := executor.setEnv(tc.env, tc.key, tc.value)
			tc.validate(t, result)
		})
	}
}
