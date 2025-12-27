package scripts

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsTrusted(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func() *TrustChecker
		packageName string
		validate    func(t *testing.T, result bool)
	}{
		{
			name: "trusted package",
			setupFunc: func() *TrustChecker {
				return NewTrustChecker([]string{"esbuild", "sharp"})
			},
			packageName: "esbuild",
			validate: func(t *testing.T, result bool) {
				assert.True(t, result, "Package should be trusted")
			},
		},
		{
			name: "untrusted package",
			setupFunc: func() *TrustChecker {
				return NewTrustChecker([]string{"esbuild", "sharp"})
			},
			packageName: "malicious-pkg",
			validate: func(t *testing.T, result bool) {
				assert.False(t, result, "Package should not be trusted")
			},
		},
		{
			name: "empty trusted list",
			setupFunc: func() *TrustChecker {
				return NewTrustChecker([]string{})
			},
			packageName: "any-package",
			validate: func(t *testing.T, result bool) {
				assert.False(t, result, "No packages should be trusted with empty list")
			},
		},
		{
			name: "scoped package trusted",
			setupFunc: func() *TrustChecker {
				return NewTrustChecker([]string{"@types/node", "esbuild"})
			},
			packageName: "@types/node",
			validate: func(t *testing.T, result bool) {
				assert.True(t, result, "Scoped package should be trusted")
			},
		},
		{
			name: "partial match not trusted",
			setupFunc: func() *TrustChecker {
				return NewTrustChecker([]string{"esbuild"})
			},
			packageName: "esbuild-loader",
			validate: func(t *testing.T, result bool) {
				assert.False(t, result, "Partial match should not be trusted")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checker := tc.setupFunc()
			result := checker.IsTrusted(tc.packageName)
			tc.validate(t, result)
		})
	}
}

func TestSetTrustedDependencies(t *testing.T) {
	testCases := []struct {
		name      string
		setupFunc func() *TrustChecker
		newDeps   []string
		validate  func(t *testing.T, tc *TrustChecker)
	}{
		{
			name: "set dependencies on empty checker",
			setupFunc: func() *TrustChecker {
				return NewTrustChecker([]string{})
			},
			newDeps: []string{"esbuild", "sharp"},
			validate: func(t *testing.T, tc *TrustChecker) {
				assert.True(t, tc.IsTrusted("esbuild"))
				assert.True(t, tc.IsTrusted("sharp"))
				assert.False(t, tc.IsTrusted("other"))
			},
		},
		{
			name: "replace existing dependencies",
			setupFunc: func() *TrustChecker {
				return NewTrustChecker([]string{"old-pkg"})
			},
			newDeps: []string{"new-pkg"},
			validate: func(t *testing.T, tc *TrustChecker) {
				assert.False(t, tc.IsTrusted("old-pkg"))
				assert.True(t, tc.IsTrusted("new-pkg"))
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			checker := tc.setupFunc()
			checker.SetTrustedDependencies(tc.newDeps)
			tc.validate(t, checker)
		})
	}
}
