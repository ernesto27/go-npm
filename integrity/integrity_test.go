package integrity

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIntegrity(t *testing.T) {
	testCases := []struct {
		name           string
		integrity      string
		expectError    error
		expectedCount  int
		expectedFirst  string // algorithm of first (strongest) hash
	}{
		{
			name:           "Single sha512 hash",
			integrity:      "sha512-5cvg6CtKwfgdmVqY1WIiXKc3Q1bkRqGLi+2W/6ao+6Y7gu/RCwRuAhGEzh5B4KlszSuTLgZYuqFqo5bImjNKng==",
			expectError:    nil,
			expectedCount:  1,
			expectedFirst:  "sha512",
		},
		{
			name:           "Single sha256 hash",
			integrity:      "sha256-abc123def456==",
			expectError:    nil,
			expectedCount:  1,
			expectedFirst:  "sha256",
		},
		{
			name:           "Multiple hashes - sha512 and sha256",
			integrity:      "sha256-abc123== sha512-xyz789==",
			expectError:    nil,
			expectedCount:  2,
			expectedFirst:  "sha512", // sha512 should be sorted first (strongest)
		},
		{
			name:           "Multiple hashes - all three algorithms",
			integrity:      "sha256-aaa== sha384-bbb== sha512-ccc==",
			expectError:    nil,
			expectedCount:  3,
			expectedFirst:  "sha512",
		},
		{
			name:          "Empty string",
			integrity:     "",
			expectError:   ErrNoIntegrity,
			expectedCount: 0,
		},
		{
			name:          "Whitespace only",
			integrity:     "   ",
			expectError:   ErrInvalidIntegrityFormat,
			expectedCount: 0,
		},
		{
			name:          "Invalid format - no dash",
			integrity:     "sha512abc123",
			expectError:   ErrUnsupportedAlgorithm,
			expectedCount: 0,
		},
		{
			name:          "Unsupported algorithm only",
			integrity:     "md5-abc123==",
			expectError:   ErrUnsupportedAlgorithm,
			expectedCount: 0,
		},
		{
			name:           "Mixed supported and unsupported",
			integrity:      "md5-xxx== sha256-abc123==",
			expectError:    nil,
			expectedCount:  1,
			expectedFirst:  "sha256",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hashes, err := ParseIntegrity(tc.integrity)

			if tc.expectError != nil {
				assert.ErrorIs(t, err, tc.expectError)
				return
			}

			assert.NoError(t, err)
			assert.Len(t, hashes, tc.expectedCount)
			if tc.expectedCount > 0 {
				assert.Equal(t, tc.expectedFirst, hashes[0].Algorithm)
			}
		})
	}
}

func TestComputeHash(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) string // returns filepath
		algorithm   string
		expectError bool
		validate    func(t *testing.T, hash string, filePath string)
	}{
		{
			name: "Compute sha512 hash",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(filePath, []byte("test content"), 0644)
				assert.NoError(t, err)
				return filePath
			},
			algorithm:   "sha512",
			expectError: false,
			validate: func(t *testing.T, hash string, filePath string) {
				// Compute expected hash
				content, _ := os.ReadFile(filePath)
				h := sha512.Sum512(content)
				expected := base64.StdEncoding.EncodeToString(h[:])
				assert.Equal(t, expected, hash)
			},
		},
		{
			name: "Compute sha256 hash",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(filePath, []byte("test content"), 0644)
				assert.NoError(t, err)
				return filePath
			},
			algorithm:   "sha256",
			expectError: false,
			validate: func(t *testing.T, hash string, filePath string) {
				content, _ := os.ReadFile(filePath)
				h := sha256.Sum256(content)
				expected := base64.StdEncoding.EncodeToString(h[:])
				assert.Equal(t, expected, hash)
			},
		},
		{
			name: "Compute sha384 hash",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(filePath, []byte("hello world"), 0644)
				assert.NoError(t, err)
				return filePath
			},
			algorithm:   "sha384",
			expectError: false,
			validate: func(t *testing.T, hash string, filePath string) {
				content, _ := os.ReadFile(filePath)
				h := sha512.Sum384(content)
				expected := base64.StdEncoding.EncodeToString(h[:])
				assert.Equal(t, expected, hash)
			},
		},
		{
			name: "File not found",
			setupFunc: func(t *testing.T) string {
				return "/nonexistent/path/file.txt"
			},
			algorithm:   "sha512",
			expectError: true,
			validate:    func(t *testing.T, hash string, filePath string) {},
		},
		{
			name: "Unsupported algorithm",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(filePath, []byte("content"), 0644)
				assert.NoError(t, err)
				return filePath
			},
			algorithm:   "md5",
			expectError: true,
			validate:    func(t *testing.T, hash string, filePath string) {},
		},
		{
			name: "Empty file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "empty.txt")
				err := os.WriteFile(filePath, []byte{}, 0644)
				assert.NoError(t, err)
				return filePath
			},
			algorithm:   "sha512",
			expectError: false,
			validate: func(t *testing.T, hash string, filePath string) {
				// Hash of empty content
				h := sha512.Sum512([]byte{})
				expected := base64.StdEncoding.EncodeToString(h[:])
				assert.Equal(t, expected, hash)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath := tc.setupFunc(t)
			hash, err := ComputeHash(filePath, tc.algorithm)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tc.validate(t, hash, filePath)
			}
		})
	}
}

func TestValidatorValidateFile(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (filePath, integrity string)
		expectError error
		expectedAlg string
	}{
		{
			name: "Valid sha512 integrity",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.txt")
				content := []byte("test content for integrity check")
				err := os.WriteFile(filePath, content, 0644)
				assert.NoError(t, err)

				// Compute actual hash
				h := sha512.Sum512(content)
				hashStr := base64.StdEncoding.EncodeToString(h[:])
				return filePath, "sha512-" + hashStr
			},
			expectError: nil,
			expectedAlg: "sha512",
		},
		{
			name: "Valid sha256 integrity",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.txt")
				content := []byte("sha256 test content")
				err := os.WriteFile(filePath, content, 0644)
				assert.NoError(t, err)

				h := sha256.Sum256(content)
				hashStr := base64.StdEncoding.EncodeToString(h[:])
				return filePath, "sha256-" + hashStr
			},
			expectError: nil,
			expectedAlg: "sha256",
		},
		{
			name: "Hash mismatch",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(filePath, []byte("actual content"), 0644)
				assert.NoError(t, err)

				// Use hash of different content
				h := sha512.Sum512([]byte("different content"))
				hashStr := base64.StdEncoding.EncodeToString(h[:])
				return filePath, "sha512-" + hashStr
			},
			expectError: ErrIntegrityMismatch,
		},
		{
			name: "Empty integrity string",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(filePath, []byte("content"), 0644)
				assert.NoError(t, err)
				return filePath, ""
			},
			expectError: ErrNoIntegrity,
		},
		{
			name: "Multiple hashes - first matches",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.txt")
				content := []byte("multi-hash content")
				err := os.WriteFile(filePath, content, 0644)
				assert.NoError(t, err)

				h512 := sha512.Sum512(content)
				h256 := sha256.Sum256(content)
				hash512 := base64.StdEncoding.EncodeToString(h512[:])
				hash256 := base64.StdEncoding.EncodeToString(h256[:])

				// Both hashes are valid - should use sha512 (strongest)
				return filePath, "sha256-" + hash256 + " sha512-" + hash512
			},
			expectError: nil,
			expectedAlg: "sha512",
		},
		{
			name: "File not found",
			setupFunc: func(t *testing.T) (string, string) {
				return "/nonexistent/file.txt", "sha512-abc123=="
			},
			expectError: nil, // Will get a wrapped error, not ErrIntegrityMismatch
		},
	}

	validator := New()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath, integrity := tc.setupFunc(t)
			alg, err := validator.ValidateFile(filePath, integrity)

			if tc.expectError != nil {
				assert.ErrorIs(t, err, tc.expectError)
			} else if tc.name == "File not found" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "failed to open file")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedAlg, alg)
			}
		})
	}
}

func TestValidatorValidateFileStrict(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (filePath, integrity string)
		expectError bool
	}{
		{
			name: "Valid integrity - success",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.txt")
				content := []byte("strict mode test")
				err := os.WriteFile(filePath, content, 0644)
				assert.NoError(t, err)

				h := sha512.Sum512(content)
				hashStr := base64.StdEncoding.EncodeToString(h[:])
				return filePath, "sha512-" + hashStr
			},
			expectError: false,
		},
		{
			name: "Empty integrity - strict mode fails",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(filePath, []byte("content"), 0644)
				assert.NoError(t, err)
				return filePath, ""
			},
			expectError: true,
		},
		{
			name: "Hash mismatch - fails",
			setupFunc: func(t *testing.T) (string, string) {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.txt")
				err := os.WriteFile(filePath, []byte("actual"), 0644)
				assert.NoError(t, err)

				h := sha512.Sum512([]byte("expected"))
				hashStr := base64.StdEncoding.EncodeToString(h[:])
				return filePath, "sha512-" + hashStr
			},
			expectError: true,
		},
	}

	validator := New()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath, integrity := tc.setupFunc(t)
			err := validator.ValidateFileStrict(filePath, integrity)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
