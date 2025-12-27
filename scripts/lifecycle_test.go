package scripts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractScripts(t *testing.T) {
	testCases := []struct {
		name     string
		input    any
		validate func(t *testing.T, result map[string]string)
	}{
		{
			name:  "nil scripts returns nil",
			input: nil,
			validate: func(t *testing.T, result map[string]string) {
				assert.Nil(t, result)
			},
		},
		{
			name: "map[string]interface{} extracts scripts",
			input: map[string]interface{}{
				"preinstall":  "echo pre",
				"postinstall": "echo post",
			},
			validate: func(t *testing.T, result map[string]string) {
				assert.Equal(t, "echo pre", result["preinstall"])
				assert.Equal(t, "echo post", result["postinstall"])
			},
		},
		{
			name: "map[string]string returns directly",
			input: map[string]string{
				"prepare": "npm run build",
			},
			validate: func(t *testing.T, result map[string]string) {
				assert.Equal(t, "npm run build", result["prepare"])
			},
		},
		{
			name:  "unsupported type returns nil",
			input: "invalid",
			validate: func(t *testing.T, result map[string]string) {
				assert.Nil(t, result)
			},
		},
		{
			name: "mixed types filters non-strings",
			input: map[string]interface{}{
				"postinstall": "echo post",
				"invalid":     123,
			},
			validate: func(t *testing.T, result map[string]string) {
				assert.Equal(t, "echo post", result["postinstall"])
				_, exists := result["invalid"]
				assert.False(t, exists, "non-string values should be filtered")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractScripts(tc.input)
			tc.validate(t, result)
		})
	}
}

func TestRunPackageScripts(t *testing.T) {
	testCases := []struct {
		name      string
		setupFunc func() (string, *LifecycleManager)
		scripts   any
		pkgName   string
		validate  func(t *testing.T, err error)
	}{
		{
			name: "ignoreScripts skips execution",
			setupFunc: func() (string, *LifecycleManager) {
				dir := t.TempDir()
				return dir, NewLifecycleManager(dir, true)
			},
			scripts: map[string]string{"postinstall": "exit 1"},
			pkgName: "test-pkg",
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err, "Should skip scripts when ignoreScripts is true")
			},
		},
		{
			name: "untrusted package skips execution",
			setupFunc: func() (string, *LifecycleManager) {
				dir := t.TempDir()
				lm := NewLifecycleManager(dir, false)
				lm.SetTrustedDependencies([]string{"other-pkg"})
				return dir, lm
			},
			scripts: map[string]string{"postinstall": "exit 1"},
			pkgName: "untrusted-pkg",
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err, "Should skip scripts for untrusted packages")
			},
		},
		{
			name: "nil scripts returns nil",
			setupFunc: func() (string, *LifecycleManager) {
				dir := t.TempDir()
				lm := NewLifecycleManager(dir, false)
				lm.SetTrustedDependencies([]string{"test-pkg"})
				return dir, lm
			},
			scripts: nil,
			pkgName: "test-pkg",
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "trusted package runs scripts",
			setupFunc: func() (string, *LifecycleManager) {
				dir := t.TempDir()
				lm := NewLifecycleManager(dir, false)
				lm.SetTrustedDependencies([]string{"test-pkg"})
				return dir, lm
			},
			scripts: map[string]string{"postinstall": "echo done"},
			pkgName: "test-pkg",
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir, lm := tc.setupFunc()
			err := lm.RunPackageScripts(tc.pkgName, "1.0.0", dir, tc.scripts)
			tc.validate(t, err)
		})
	}
}

func TestRunRootPackageScripts(t *testing.T) {
	testCases := []struct {
		name      string
		setupFunc func() (string, *LifecycleManager)
		scripts   any
		validate  func(t *testing.T, err error)
	}{
		{
			name: "bypasses trust check",
			setupFunc: func() (string, *LifecycleManager) {
				dir := t.TempDir()
				lm := NewLifecycleManager(dir, false)
				lm.SetTrustedDependencies([]string{})
				return dir, lm
			},
			scripts: map[string]string{"postinstall": "echo root"},
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err, "Root package should bypass trust check")
			},
		},
		{
			name: "respects ignoreScripts",
			setupFunc: func() (string, *LifecycleManager) {
				dir := t.TempDir()
				return dir, NewLifecycleManager(dir, true)
			},
			scripts: map[string]string{"postinstall": "exit 1"},
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err, "Should skip even root scripts when ignoreScripts is true")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir, lm := tc.setupFunc()
			err := lm.RunRootPackageScripts("my-app", "1.0.0", dir, tc.scripts)
			tc.validate(t, err)
		})
	}
}

func TestRunPrepare(t *testing.T) {
	testCases := []struct {
		name      string
		setupFunc func() (string, *LifecycleManager)
		scripts   any
		validate  func(t *testing.T, err error)
	}{
		{
			name: "runs prepare script",
			setupFunc: func() (string, *LifecycleManager) {
				dir := t.TempDir()
				return dir, NewLifecycleManager(dir, false)
			},
			scripts: map[string]string{"prepare": "echo prepare"},
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "no prepare script returns nil",
			setupFunc: func() (string, *LifecycleManager) {
				dir := t.TempDir()
				return dir, NewLifecycleManager(dir, false)
			},
			scripts: map[string]string{"postinstall": "echo post"},
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "ignoreScripts skips prepare",
			setupFunc: func() (string, *LifecycleManager) {
				dir := t.TempDir()
				return dir, NewLifecycleManager(dir, true)
			},
			scripts: map[string]string{"prepare": "exit 1"},
			validate: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir, lm := tc.setupFunc()
			err := lm.RunPrepare("my-app", "1.0.0", dir, tc.scripts)
			tc.validate(t, err)
		})
	}
}
