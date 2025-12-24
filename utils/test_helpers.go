package utils

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// CaptureStdout captures the output written to os.Stdout during the execution of f.
func CaptureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// BuildTestBinary builds the go-npm binary for testing and returns the path to it.
// The binary is built in a temp directory to avoid polluting the project.
func BuildTestBinary(t *testing.T, projectRoot string) string {
	t.Helper()

	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "go-npm-test")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = projectRoot
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "failed to build binary: %s", string(output))

	return binaryPath
}

// RunWithIsolatedCache runs a command with GO_NPM_HOME and HOME set to temp directories
// to avoid polluting the user's real cache at ~/.config/go-npm
func RunWithIsolatedCache(t *testing.T, binaryPath string, workDir string, args ...string) ([]byte, error, string) {
	t.Helper()

	cacheDir := t.TempDir()

	cmd := exec.Command(binaryPath, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "GO_NPM_HOME="+cacheDir, "HOME="+cacheDir)

	output, err := cmd.CombinedOutput()
	return output, err, cacheDir
}
