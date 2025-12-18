package yarnlock

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
)

const (
	yarnV1Header = "# yarn lockfile v1"
)

// YarnLockEntry represents a single package entry in yarn.lock
type YarnLockEntry struct {
	Name                 string            // Package name (extracted from key)
	Version              string            // Resolved version
	Resolved             string            // Download URL
	Integrity            string            // SHA hash for verification
	Dependencies         map[string]string // Sub-dependencies with version ranges
	OptionalDependencies map[string]string // Optional sub-dependencies
}

// YarnLock represents the parsed yarn.lock file
type YarnLock struct {
	Entries map[string]YarnLockEntry // Keyed by "name@version" (deduplicated)
}

// YarnLockParser provides methods for parsing yarn.lock files
type YarnLockParser struct{}

// NewYarnLockParser creates a new YarnLockParser instance
func NewYarnLockParser() *YarnLockParser {
	return &YarnLockParser{}
}

// IsYarnV1 checks if the content is a Yarn v1 lockfile
func (p *YarnLockParser) IsYarnV1(content []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == yarnV1Header {
			return true
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		return false
	}
	return false
}

// ParseContent parses yarn.lock content from bytes
func (p *YarnLockParser) ParseContent(content []byte) (*YarnLock, error) {
	if !p.IsYarnV1(content) {
		return nil, fmt.Errorf("unsupported yarn.lock format: only v1 is supported")
	}

	yarnLock := &YarnLock{
		Entries: make(map[string]YarnLockEntry),
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	var currentEntry *YarnLockEntry
	var currentKeys []string
	var inDependencies bool
	var inOptionalDependencies bool

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if !strings.HasPrefix(line, " ") && strings.HasSuffix(line, ":") {
			if currentEntry != nil && len(currentKeys) > 0 {
				saveEntry(yarnLock, currentEntry, currentKeys)
			}

			keys, err := parseEntryHeader(line)
			if err != nil {
				return nil, fmt.Errorf("failed to parse entry header '%s': %w", line, err)
			}

			currentKeys = keys
			name := extractPackageName(keys[0])
			currentEntry = &YarnLockEntry{
				Name:                 name,
				Dependencies:         make(map[string]string),
				OptionalDependencies: make(map[string]string),
			}
			inDependencies = false
			inOptionalDependencies = false
			continue
		}

		if currentEntry == nil {
			continue
		}

		trimmed := strings.TrimLeft(line, " ")
		indent := len(line) - len(trimmed)

		switch indent {
		case 2:
			inDependencies = false
			inOptionalDependencies = false

			if strings.HasPrefix(trimmed, "version ") {
				currentEntry.Version = parseQuotedValue(trimmed[8:])
			} else if strings.HasPrefix(trimmed, "resolved ") {
				currentEntry.Resolved = parseQuotedValue(trimmed[9:])
			} else if strings.HasPrefix(trimmed, "integrity ") {
				currentEntry.Integrity = parseQuotedValue(trimmed[10:])
			} else if trimmed == "dependencies:" {
				inDependencies = true
			} else if trimmed == "optionalDependencies:" {
				inOptionalDependencies = true
			}
		case 4:
			if inDependencies {
				name, version := parseDependencyLine(trimmed)
				if name != "" {
					currentEntry.Dependencies[name] = version
				}
			} else if inOptionalDependencies {
				name, version := parseDependencyLine(trimmed)
				if name != "" {
					currentEntry.OptionalDependencies[name] = version
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading yarn.lock: %w", err)
	}

	if currentEntry != nil && len(currentKeys) > 0 {
		saveEntry(yarnLock, currentEntry, currentKeys)
	}

	return yarnLock, nil
}

// Parse reads and parses a yarn.lock file from the given path
func Parse(filePath string) (*YarnLock, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read yarn.lock: %w", err)
	}
	return ParseContent(content)
}

// ParseContent parses yarn.lock content from bytes
func ParseContent(content []byte) (*YarnLock, error) {
	if !IsYarnV1(content) {
		return nil, fmt.Errorf("unsupported yarn.lock format: only v1 is supported")
	}

	yarnLock := &YarnLock{
		Entries: make(map[string]YarnLockEntry),
	}

	scanner := bufio.NewScanner(bytes.NewReader(content))
	var currentEntry *YarnLockEntry
	var currentKeys []string
	var inDependencies bool
	var inOptionalDependencies bool

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check if this is a package header (not indented, ends with colon)
		if !strings.HasPrefix(line, " ") && strings.HasSuffix(line, ":") {
			// Save previous entry if exists
			if currentEntry != nil && len(currentKeys) > 0 {
				saveEntry(yarnLock, currentEntry, currentKeys)
			}

			// Parse new entry header
			keys, err := parseEntryHeader(line)
			if err != nil {
				return nil, fmt.Errorf("failed to parse entry header '%s': %w", line, err)
			}

			currentKeys = keys
			name := extractPackageName(keys[0])
			currentEntry = &YarnLockEntry{
				Name:                 name,
				Dependencies:         make(map[string]string),
				OptionalDependencies: make(map[string]string),
			}
			inDependencies = false
			inOptionalDependencies = false
			continue
		}

		if currentEntry == nil {
			continue
		}

		// Count leading spaces to determine indentation level
		trimmed := strings.TrimLeft(line, " ")
		indent := len(line) - len(trimmed)

		// Handle field parsing based on indentation
		switch indent {
		case 2:
			// Top-level field within an entry
			inDependencies = false
			inOptionalDependencies = false

			if strings.HasPrefix(trimmed, "version ") {
				currentEntry.Version = parseQuotedValue(trimmed[8:])
			} else if strings.HasPrefix(trimmed, "resolved ") {
				currentEntry.Resolved = parseQuotedValue(trimmed[9:])
			} else if strings.HasPrefix(trimmed, "integrity ") {
				currentEntry.Integrity = parseQuotedValue(trimmed[10:])
			} else if trimmed == "dependencies:" {
				inDependencies = true
			} else if trimmed == "optionalDependencies:" {
				inOptionalDependencies = true
			}
		case 4:
			// Dependency entry (4 spaces indentation)
			if inDependencies {
				name, version := parseDependencyLine(trimmed)
				if name != "" {
					currentEntry.Dependencies[name] = version
				}
			} else if inOptionalDependencies {
				name, version := parseDependencyLine(trimmed)
				if name != "" {
					currentEntry.OptionalDependencies[name] = version
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading yarn.lock: %w", err)
	}

	// Save the last entry
	if currentEntry != nil && len(currentKeys) > 0 {
		saveEntry(yarnLock, currentEntry, currentKeys)
	}

	return yarnLock, nil
}

// IsYarnV1 checks if the content is a Yarn v1 lockfile
func IsYarnV1(content []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Check if this line is the yarn v1 header
		// Real yarn.lock files may have comments before the version header
		if line == yarnV1Header {
			return true
		}
		// Skip other comment lines (they start with #)
		if strings.HasPrefix(line, "#") {
			continue
		}
		// If we hit a non-comment, non-empty line without finding the header, it's not v1
		return false
	}
	return false
}

// saveEntry saves the entry to the YarnLock, deduplicating by name@version
func saveEntry(yarnLock *YarnLock, entry *YarnLockEntry, keys []string) {
	// Use name@version as the deduplicated key
	dedupeKey := entry.Name + "@" + entry.Version
	yarnLock.Entries[dedupeKey] = *entry
}

// parseEntryHeader parses a package entry header line
// Examples:
//   - "express@^4.18.2:"
//   - "accepts@~1.3.8, accepts@~1.3.9:"
//   - "@types/node@^18.0.0:"
func parseEntryHeader(line string) ([]string, error) {
	// Remove trailing colon
	line = strings.TrimSuffix(line, ":")

	// Split by ", " to handle multiple version selectors
	parts := strings.Split(line, ", ")
	keys := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		// Remove surrounding quotes if present
		part = strings.Trim(part, "\"")
		if part != "" {
			keys = append(keys, part)
		}
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("empty entry header")
	}

	return keys, nil
}

// extractPackageName extracts the package name from a key like "express@^4.18.2"
// Handles scoped packages like "@types/node@^18.0.0"
func extractPackageName(key string) string {
	// Remove surrounding quotes
	key = strings.Trim(key, "\"")

	// For scoped packages (@scope/name@version), we need to find the last @
	// that separates the name from the version
	if strings.HasPrefix(key, "@") {
		// Scoped package: @scope/name@version
		// Find the @ after the scope/name part
		slashIdx := strings.Index(key, "/")
		if slashIdx == -1 {
			return key
		}
		// Look for @ after the slash
		remainder := key[slashIdx:]
		atIdx := strings.LastIndex(remainder, "@")
		if atIdx == -1 || atIdx == 0 {
			return key
		}
		return key[:slashIdx+atIdx]
	}

	// Non-scoped package: name@version
	atIdx := strings.LastIndex(key, "@")
	if atIdx == -1 || atIdx == 0 {
		return key
	}
	return key[:atIdx]
}

// parseQuotedValue extracts a value from a quoted string
// Example: "1.3.8" -> 1.3.8
func parseQuotedValue(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"")
	return s
}

// parseDependencyLine parses a dependency line like:
// mime-types "~2.1.34"
// "@types/node" "^18.0.0"
func parseDependencyLine(line string) (name, version string) {
	line = strings.TrimSpace(line)

	// Use regex to handle both quoted and unquoted package names
	// Pattern: packageName "version" or "packageName" "version"
	re := regexp.MustCompile(`^"?(@?[^"\s]+)"?\s+"?([^"]+)"?$`)
	matches := re.FindStringSubmatch(line)

	if len(matches) == 3 {
		return matches[1], matches[2]
	}

	// Fallback: split by space and clean up
	parts := strings.SplitN(line, " ", 2)
	if len(parts) == 2 {
		name = strings.Trim(parts[0], "\"")
		version = strings.Trim(parts[1], "\"")
		return name, version
	}

	return "", ""
}

// GetEntryByName returns an entry by package name (first match)
func (y *YarnLock) GetEntryByName(name string) (*YarnLockEntry, bool) {
	for key, entry := range y.Entries {
		if entry.Name == name || strings.HasPrefix(key, name+"@") {
			return &entry, true
		}
	}
	return nil, false
}

// GetAllPackageNames returns a list of all unique package names
func (y *YarnLock) GetAllPackageNames() []string {
	seen := make(map[string]bool)
	names := make([]string, 0)

	for _, entry := range y.Entries {
		if !seen[entry.Name] {
			seen[entry.Name] = true
			names = append(names, entry.Name)
		}
	}

	return names
}
