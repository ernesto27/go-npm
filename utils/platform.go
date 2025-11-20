package utils

import (
	"runtime"
	"strings"
)

// GetCurrentOS returns the current operating system in npm format
// (linux, darwin, win32, freebsd, etc.)
func GetCurrentOS() string {
	switch runtime.GOOS {
	case "windows":
		return "win32"
	case "darwin":
		return "darwin"
	case "linux":
		return "linux"
	case "freebsd":
		return "freebsd"
	case "openbsd":
		return "openbsd"
	case "netbsd":
		return "netbsd"
	case "aix":
		return "aix"
	case "solaris":
		return "sunos"
	default:
		return runtime.GOOS
	}
}

// GetCurrentCPU returns the current CPU architecture in npm format
// (x64, arm64, arm, ia32, etc.)
func GetCurrentCPU() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x64"
	case "386":
		return "ia32"
	case "arm64":
		return "arm64"
	case "arm":
		return "arm"
	case "ppc64":
		return "ppc64"
	case "ppc64le":
		return "ppc64"
	case "s390x":
		return "s390x"
	case "mips":
		return "mips"
	case "mipsle":
		return "mipsel"
	case "mips64":
		return "mips64"
	case "mips64le":
		return "mips64el"
	default:
		return runtime.GOARCH
	}
}

// IsCompatiblePlatform checks if the current platform is compatible with the package
// based on the os and cpu constraints from the package manifest.
// Returns true if the package is compatible or if no constraints are specified.
func IsCompatiblePlatform(osConstraints []string, cpuConstraints []string) bool {
	currentOS := GetCurrentOS()
	currentCPU := GetCurrentCPU()

	// If no constraints specified, package is compatible
	if len(osConstraints) == 0 && len(cpuConstraints) == 0 {
		return true
	}

	// Check OS constraints
	if len(osConstraints) > 0 {
		osCompatible := checkConstraint(currentOS, osConstraints)
		if !osCompatible {
			return false
		}
	}

	// Check CPU constraints
	if len(cpuConstraints) > 0 {
		cpuCompatible := checkConstraint(currentCPU, cpuConstraints)
		if !cpuCompatible {
			return false
		}
	}

	return true
}

// checkConstraint checks if the current value matches the constraint list.
// Supports negation syntax (e.g., "!win32" means "not Windows").
func checkConstraint(current string, constraints []string) bool {
	hasPositive := false
	hasNegative := false

	for _, constraint := range constraints {
		// Handle negation (e.g., "!win32")
		if strings.HasPrefix(constraint, "!") {
			hasNegative = true
			excluded := strings.TrimPrefix(constraint, "!")
			if current == excluded {
				return false
			}
		} else {
			hasPositive = true
			if current == constraint {
				return true
			}
		}
	}

	if hasNegative && !hasPositive {
		return true
	}

	if hasPositive {
		return false
	}

	return true
}
