package integrity

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"sort"
	"strings"
)

var (
	ErrIntegrityMismatch     = errors.New("integrity check failed: hash mismatch")
	ErrNoIntegrity           = errors.New("no integrity information available")
	ErrUnsupportedAlgorithm  = errors.New("unsupported hash algorithm")
	ErrInvalidIntegrityFormat = errors.New("invalid integrity format")
)

// IntegrityHash represents a parsed SRI hash
type IntegrityHash struct {
	Algorithm string // "sha512", "sha384", "sha256"
	Hash      string // base64-encoded hash
	Raw       string // original "{alg}-{hash}" string
}

// algorithmStrength defines priority order (higher = stronger)
var algorithmStrength = map[string]int{
	"sha512": 3,
	"sha384": 2,
	"sha256": 1,
}

// Validator handles SRI hash validation
type Validator struct{}

// New creates a new integrity Validator
func New() *Validator {
	return &Validator{}
}

// ParseIntegrity parses an SRI string into algorithm-hash pairs
// SRI format: "{algorithm}-{base64hash}" (space-separated for multiple)
// Returns hashes sorted by strength (sha512 first)
func ParseIntegrity(integrity string) ([]IntegrityHash, error) {
	if integrity == "" {
		return nil, ErrNoIntegrity
	}

	parts := strings.Fields(integrity)
	if len(parts) == 0 {
		return nil, ErrInvalidIntegrityFormat
	}

	var hashes []IntegrityHash
	for _, part := range parts {
		idx := strings.Index(part, "-")
		if idx == -1 {
			continue // Skip malformed entries
		}

		algorithm := part[:idx]
		hashValue := part[idx+1:]

		if _, ok := algorithmStrength[algorithm]; !ok {
			continue // Skip unsupported algorithms
		}

		hashes = append(hashes, IntegrityHash{
			Algorithm: algorithm,
			Hash:      hashValue,
			Raw:       part,
		})
	}

	if len(hashes) == 0 {
		return nil, ErrUnsupportedAlgorithm
	}

	// Sort by strength (strongest first)
	sort.Slice(hashes, func(i, j int) bool {
		return algorithmStrength[hashes[i].Algorithm] > algorithmStrength[hashes[j].Algorithm]
	})

	return hashes, nil
}

// ComputeHash computes the hash of a file using the specified algorithm
// Returns base64-encoded hash string
func ComputeHash(filePath, algorithm string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var h hash.Hash
	switch algorithm {
	case "sha512":
		h = sha512.New()
	case "sha384":
		h = sha512.New384()
	case "sha256":
		h = sha256.New()
	default:
		return "", ErrUnsupportedAlgorithm
	}

	// Stream the file to avoid loading entire tarball into memory
	if _, err := io.Copy(h, file); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

// ValidateFile validates a file against an SRI integrity string
// Uses the strongest available algorithm from the SRI string
// Returns the matched algorithm on success, or error on failure
func (v *Validator) ValidateFile(filePath, integrity string) (string, error) {
	hashes, err := ParseIntegrity(integrity)
	if err != nil {
		return "", err
	}

	// Try each hash starting with the strongest
	var lastErr error
	for _, ih := range hashes {
		computed, err := ComputeHash(filePath, ih.Algorithm)
		if err != nil {
			lastErr = err
			continue
		}

		if computed == ih.Hash {
			return ih.Algorithm, nil
		}

		// Hash computed but didn't match - this is a security failure
		lastErr = fmt.Errorf("%w: expected %s, got %s (algorithm: %s)",
			ErrIntegrityMismatch, ih.Hash, computed, ih.Algorithm)
	}

	if lastErr != nil {
		return "", lastErr
	}

	return "", ErrUnsupportedAlgorithm
}

// ValidateFileStrict validates a file and returns detailed error information
// This is the main entry point for tarball validation
func (v *Validator) ValidateFileStrict(filePath, integrity string) error {
	if integrity == "" {
		return ErrNoIntegrity
	}

	algorithm, err := v.ValidateFile(filePath, integrity)
	if err != nil {
		return err
	}

	_ = algorithm // Successfully validated
	return nil
}
