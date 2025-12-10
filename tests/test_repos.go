package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// ANSI color codes
const (
	ColorRed    = "\033[0;31m"
	ColorGreen  = "\033[0;32m"
	ColorYellow = "\033[1;33m"
	ColorBlue   = "\033[0;34m"
	ColorReset  = "\033[0m"
)

// Repository represents a GitHub repository to test
type Repository struct {
	Name string
	URL  string
}

// TestSuite manages the test execution
type TestSuite struct {
	TestReposDir   string
	LogFile        string
	NPMPackagerBin string
	Repositories   []Repository
	WorkspaceRepos []string

	logWriter       *os.File
	totalTests      int
	successfulTests int
	failedTests     int
}

// NewTestSuite creates a new test suite with default configuration
func NewTestSuite(scriptDir string) (*TestSuite, error) {
	logPath := filepath.Join(scriptDir, "test-results.log")

	// Remove old log file
	os.Remove(logPath)

	// Create new log file
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Load repositories from JSON config
	repos, err := getDefaultRepositories(scriptDir)
	if err != nil {
		logFile.Close()
		return nil, err
	}

	return &TestSuite{
		TestReposDir:   filepath.Join(scriptDir, "repos"),
		LogFile:        logPath,
		NPMPackagerBin: filepath.Join(scriptDir, "..", "npm-packager"),
		Repositories:   repos,
		WorkspaceRepos: getDefaultWorkspaceRepos(),
		logWriter:      logFile,
	}, nil
}

// Close closes the log file
func (ts *TestSuite) Close() error {
	if ts.logWriter != nil {
		return ts.logWriter.Close()
	}
	return nil
}

// RepoConfig represents the JSON configuration structure
type RepoConfig struct {
	Repositories []Repository `json:"repositories"`
}

// loadRepositoriesFromJSON loads repositories from a JSON file
func loadRepositoriesFromJSON(configPath string) ([]Repository, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config RepoConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config.Repositories, nil
}

// getDefaultRepositories loads repositories from repos-config.json
func getDefaultRepositories(scriptDir string) ([]Repository, error) {
	configPath := filepath.Join(scriptDir, "repos-config.json")
	repos, err := loadRepositoriesFromJSON(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", configPath, err)
	}

	printStatus(ColorGreen, fmt.Sprintf("📋 Loaded %d repositories from %s", len(repos), configPath))
	return repos, nil
}

// getDefaultWorkspaceRepos returns the list of workspace repositories to test
func getDefaultWorkspaceRepos() []string {
	return []string{
		"workspaces/simple",
		"workspaces/complex",
	}
}

// printStatus prints a colored status message
func printStatus(color, message string) {
	fmt.Printf("%s%s%s\n", color, message, ColorReset)
}

// logMessage logs a message to both console and file
func (ts *TestSuite) logMessage(message string) {
	fmt.Println(message)
	if ts.logWriter != nil {
		ts.logWriter.WriteString(message + "\n")
	}
}

// cloneRepoIfNeeded clones a repository if it doesn't already exist
func (ts *TestSuite) cloneRepoIfNeeded(repo Repository) error {
	repoPath := filepath.Join(ts.TestReposDir, repo.Name)

	if _, err := os.Stat(repoPath); err == nil {
		printStatus(ColorYellow, fmt.Sprintf("  ⊙ Repository '%s' already exists, skipping clone", repo.Name))
		ts.logMessage(fmt.Sprintf("[INFO] Repository '%s' already exists at %s", repo.Name, repoPath))
		return nil
	}

	printStatus(ColorBlue, fmt.Sprintf("  ↓ Cloning repository '%s'...", repo.Name))
	ts.logMessage(fmt.Sprintf("[INFO] Cloning %s to %s", repo.URL, repoPath))

	cmd := exec.Command("git", "clone", "--depth", "1", repo.URL, repoPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		ts.logMessage(fmt.Sprintf("[ERROR] Failed to clone: %s", string(output)))
		return fmt.Errorf("failed to clone %s: %w", repo.Name, err)
	}

	printStatus(ColorGreen, fmt.Sprintf("  ✓ Successfully cloned '%s'", repo.Name))
	return nil
}

// cleanupRepo removes node_modules and go-package-lock from a repository
func (ts *TestSuite) cleanupRepo(repoPath, repoName string) {
	printStatus(ColorYellow, fmt.Sprintf("  🧹 Cleaning '%s'...", repoName))

	cleaned := false

	// Remove node_modules
	nodeModulesPath := filepath.Join(repoPath, "node_modules")
	if _, err := os.Stat(nodeModulesPath); err == nil {
		ts.logMessage(fmt.Sprintf("[INFO] Removing node_modules from %s", repoName))
		os.RemoveAll(nodeModulesPath)
		cleaned = true
	}

	// Remove go-package-lock
	lockFilePath := filepath.Join(repoPath, "go-package-lock")
	if _, err := os.Stat(lockFilePath); err == nil {
		ts.logMessage(fmt.Sprintf("[INFO] Removing go-package-lock from %s", repoName))
		os.Remove(lockFilePath)
		cleaned = true
	}

	if cleaned {
		printStatus(ColorGreen, fmt.Sprintf("  ✓ Cleaned '%s'", repoName))
	} else {
		printStatus(ColorGreen, fmt.Sprintf("  ✓ '%s' already clean", repoName))
	}
}

// testRepo tests a repository with npm-packager
func (ts *TestSuite) testRepo(repoPath, repoName string) error {
	fmt.Println()
	printStatus(ColorBlue, fmt.Sprintf("│ Testing '%s' with npm-packager", repoName))
	printStatus(ColorBlue, "└─────────────────────────────────────────")

	ts.logMessage("")
	ts.logMessage("==========================================")
	ts.logMessage(fmt.Sprintf("Testing: %s", repoName))
	ts.logMessage(fmt.Sprintf("Path: %s", repoPath))
	ts.logMessage(fmt.Sprintf("Timestamp: %s", time.Now().Format("2006-01-02 15:04:05")))
	ts.logMessage("==========================================")

	// Check if package.json exists
	packageJsonPath := filepath.Join(repoPath, "package.json")
	if _, err := os.Stat(packageJsonPath); os.IsNotExist(err) {
		printStatus(ColorRed, fmt.Sprintf("✗ No package.json found in '%s', skipping", repoName))
		ts.logMessage(fmt.Sprintf("[ERROR] No package.json found in %s", repoPath))
		return fmt.Errorf("no package.json found")
	}

	// Run npm-packager
	ts.logMessage(fmt.Sprintf("[INFO] Running npm-packager for repository: %s", repoName))
	printStatus(ColorYellow, fmt.Sprintf("⚙  Running: %s i", filepath.Base(ts.NPMPackagerBin)))
	fmt.Println()

	startTime := time.Now()
	cmd := exec.Command(ts.NPMPackagerBin, "i")
	cmd.Dir = repoPath

	// Create pipes for stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Create a multi-writer to write to both console and log file
	consoleAndLog := io.MultiWriter(os.Stdout, ts.logWriter)

	// Use WaitGroup to ensure all output is captured
	var wg sync.WaitGroup
	wg.Add(2)

	// Stream stdout and stderr in real-time
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintln(consoleAndLog, "  "+line)
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintf(consoleAndLog, "%s  %s%s\n", ColorRed, line, ColorReset)
		}
	}()

	// Wait for command to complete
	cmdErr := cmd.Wait()

	// Wait for all output to be processed
	wg.Wait()

	duration := time.Since(startTime).Seconds()

	fmt.Println()

	if cmdErr != nil {
		printStatus(ColorRed, fmt.Sprintf("✗ Failed to install dependencies for '%s' (%.1fs)", repoName, duration))
		ts.logMessage(fmt.Sprintf("[ERROR] Repository: %s - npm-packager failed after %.1fs", repoName, duration))
		return cmdErr
	}

	printStatus(ColorGreen, fmt.Sprintf("✓ Successfully installed dependencies for '%s' (%.1fs)", repoName, duration))
	ts.logMessage(fmt.Sprintf("[SUCCESS] npm-packager completed successfully in %.1fs", duration))

	// Count installed packages
	nodeModulesPath := filepath.Join(repoPath, "node_modules")
	if entries, err := os.ReadDir(nodeModulesPath); err == nil {
		pkgCount := len(entries)
		printStatus(ColorGreen, fmt.Sprintf("  📦 Installed %d packages", pkgCount))
		ts.logMessage(fmt.Sprintf("[INFO] Installed %d packages in node_modules", pkgCount))
	}

	return nil
}

// Run executes the test suite
func (ts *TestSuite) Run() error {
	printStatus(ColorBlue, "==========================================")
	printStatus(ColorBlue, "  NPM-PACKAGER REPOSITORY TEST SUITE")
	printStatus(ColorBlue, "==========================================")
	fmt.Println()

	ts.logMessage("NPM-Packager Repository Test Suite")
	ts.logMessage(fmt.Sprintf("Started: %s", time.Now().Format("2006-01-02 15:04:05")))
	ts.logMessage("")

	// Create repos directory
	if err := os.MkdirAll(ts.TestReposDir, 0755); err != nil {
		return fmt.Errorf("failed to create repos directory: %w", err)
	}

	// Always rebuild npm-packager to ensure we're testing the latest code
	printStatus(ColorYellow, "🔨 Building npm-packager...")

	cmd := exec.Command("go", "build", "-o", "npm-packager")
	cmd.Dir = filepath.Dir(ts.NPMPackagerBin)

	buildOutput, err := cmd.CombinedOutput()
	if err != nil {
		printStatus(ColorRed, fmt.Sprintf("Failed to build: %s", string(buildOutput)))
		return fmt.Errorf("failed to build npm-packager: %w", err)
	}

	printStatus(ColorGreen, "✓ Built npm-packager successfully")
	fmt.Println()

	// Process each repository
	for i, repo := range ts.Repositories {
		ts.totalTests++

		fmt.Println()
		printStatus(ColorBlue, "┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		printStatus(ColorBlue, fmt.Sprintf("┃ [%d/%d] Repository: %s", i+1, len(ts.Repositories), repo.Name))
		printStatus(ColorBlue, "┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// Clone if needed
		if err := ts.cloneRepoIfNeeded(repo); err != nil {
			printStatus(ColorRed, fmt.Sprintf("  ✗ Failed to clone %s: %v", repo.Name, err))
			ts.failedTests++
			continue
		}

		repoPath := filepath.Join(ts.TestReposDir, repo.Name)

		// Clean up
		ts.cleanupRepo(repoPath, repo.Name)

		// Test
		if err := ts.testRepo(repoPath, repo.Name); err == nil {
			ts.successfulTests++
		} else {
			ts.failedTests++
		}
	}

	// Run tests on workspace repos (commented out by default, like in bash version)
	// ts.testWorkspaceRepos()

	// Print summary
	fmt.Println()
	fmt.Println()
	printStatus(ColorBlue, "╔══════════════════════════════════════════╗")
	printStatus(ColorBlue, "║           TEST SUMMARY                   ║")
	printStatus(ColorBlue, "╚══════════════════════════════════════════╝")
	fmt.Println()

	ts.logMessage("")
	ts.logMessage("==========================================")
	ts.logMessage("TEST SUMMARY")
	ts.logMessage("==========================================")
	ts.logMessage(fmt.Sprintf("Total repositories tested: %d", ts.totalTests))
	ts.logMessage(fmt.Sprintf("Successful: %d", ts.successfulTests))
	ts.logMessage(fmt.Sprintf("Failed: %d", ts.failedTests))
	ts.logMessage(fmt.Sprintf("Completed: %s", time.Now().Format("2006-01-02 15:04:05")))
	ts.logMessage("==========================================")

	// Calculate success rate
	successRate := float64(ts.successfulTests) / float64(ts.totalTests) * 100

	printStatus(ColorBlue, fmt.Sprintf("  Total repositories tested: %d", ts.totalTests))
	printStatus(ColorGreen, fmt.Sprintf("  ✓ Successful: %d", ts.successfulTests))
	if ts.failedTests > 0 {
		printStatus(ColorRed, fmt.Sprintf("  ✗ Failed: %d", ts.failedTests))
	} else {
		printStatus(ColorGreen, "  ✗ Failed: 0")
	}
	fmt.Println()

	if ts.failedTests == 0 {
		printStatus(ColorGreen, fmt.Sprintf("  🎉 Success rate: %.1f%% - All tests passed!", successRate))
	} else {
		printStatus(ColorYellow, fmt.Sprintf("  📊 Success rate: %.1f%%", successRate))
	}

	printStatus(ColorBlue, fmt.Sprintf("  📝 Full logs saved to: %s", ts.LogFile))
	fmt.Println()

	if ts.failedTests > 0 {
		return fmt.Errorf("test suite failed with %d failures", ts.failedTests)
	}

	return nil
}

func main() {
	// Detect project root by looking for go.mod or main.go
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get current directory: %v\n", err)
		os.Exit(1)
	}

	// Check if we're in the tests directory
	testsDir := workDir
	if filepath.Base(workDir) != "tests" {
		// We're likely in project root, use tests subdirectory
		testsDir = filepath.Join(workDir, "tests")
	}

	// Create and run test suite
	suite, err := NewTestSuite(testsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create test suite: %v\n", err)
		os.Exit(1)
	}
	defer suite.Close()

	if err := suite.Run(); err != nil {
		os.Exit(1)
	}
}
