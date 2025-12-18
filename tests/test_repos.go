package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	YarnTestDir    string
	LogFile        string
	NPMPackagerBin string
	Repositories   []Repository
	WorkspaceRepos []string

	logWriter       *os.File
	totalTests      int
	successfulTests int
	failedTests     int
	failedRepos     map[string]bool
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
		YarnTestDir:    filepath.Join(scriptDir, "yarn"),
		LogFile:        logPath,
		NPMPackagerBin: filepath.Join(scriptDir, "..", "npm-packager"),
		Repositories:   repos,
		WorkspaceRepos: getDefaultWorkspaceRepos(),
		logWriter:      logFile,
		failedRepos:    make(map[string]bool),
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
	lockFilePath := filepath.Join(repoPath, "go-npm-lock.json")
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
func (ts *TestSuite) testRepo(repoPath, repoName string, withLockFile bool) error {
	testPhase := "without lock file"
	if withLockFile {
		testPhase = "with lock file"
	}

	fmt.Println()
	printStatus(ColorBlue, fmt.Sprintf("│ Testing '%s' %s", repoName, testPhase))
	printStatus(ColorBlue, "└─────────────────────────────────────────")

	ts.logMessage("")
	ts.logMessage("==========================================")
	ts.logMessage(fmt.Sprintf("Testing: %s (%s)", repoName, testPhase))
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

	// Verify lock file status matches expected test phase
	lockFilePath := filepath.Join(repoPath, "go-npm-lock.json")
	_, lockFileExists := os.Stat(lockFilePath)
	if withLockFile && lockFileExists != nil {
		printStatus(ColorYellow, "  ⚠  Expected lock file but none found, continuing anyway...")
		ts.logMessage("[WARNING] Expected lock file but none found")
	} else if !withLockFile && lockFileExists == nil {
		printStatus(ColorYellow, "  ⚠  Lock file exists but testing without lock file phase, this shouldn't happen")
		ts.logMessage("[WARNING] Lock file exists during 'without lock file' test phase")
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

	printStatus(ColorGreen, fmt.Sprintf("✓ Successfully installed dependencies for '%s' %s (%.1fs)", repoName, testPhase, duration))
	ts.logMessage(fmt.Sprintf("[SUCCESS] npm-packager completed successfully in %.1fs", duration))

	// Count installed packages
	nodeModulesPath := filepath.Join(repoPath, "node_modules")
	if entries, err := os.ReadDir(nodeModulesPath); err == nil {
		pkgCount := len(entries)
		printStatus(ColorGreen, fmt.Sprintf("  📦 Installed %d packages", pkgCount))
		ts.logMessage(fmt.Sprintf("[INFO] Installed %d packages in node_modules", pkgCount))
	}

	// Verify lock file was created in first phase
	if !withLockFile {
		if _, err := os.Stat(lockFilePath); err == nil {
			printStatus(ColorGreen, "  ✓ Lock file created successfully")
			ts.logMessage("[INFO] go-npm-lock.json was created")
		} else {
			printStatus(ColorYellow, "  ⚠  Lock file was not created")
			ts.logMessage("[WARNING] go-npm-lock.json was not created")
		}
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

	// Process each repository (2 tests per repo: without lock file, then with lock file)
	for i, repo := range ts.Repositories {
		fmt.Println()
		printStatus(ColorBlue, "┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		printStatus(ColorBlue, fmt.Sprintf("┃ [%d/%d] Repository: %s", i+1, len(ts.Repositories), repo.Name))
		printStatus(ColorBlue, "┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// Clone if needed
		if err := ts.cloneRepoIfNeeded(repo); err != nil {
			printStatus(ColorRed, fmt.Sprintf("  ✗ Failed to clone %s: %v", repo.Name, err))
			ts.failedTests += 2 // Both tests will fail
			ts.totalTests += 2
			if repo.Name != "" {
				ts.failedRepos[repo.Name] = true
			}
			continue
		}

		repoPath := filepath.Join(ts.TestReposDir, repo.Name)

		// Clean up before first test
		ts.cleanupRepo(repoPath, repo.Name)

		// Test 1: Without lock file (fresh install)
		ts.totalTests++
		printStatus(ColorBlue, "\n  ═══ Phase 1: Testing without lock file ═══")
		if err := ts.testRepo(repoPath, repo.Name, false); err == nil {
			ts.successfulTests++
		} else {
			ts.failedTests++
			if repo.Name != "" {
				ts.failedRepos[repo.Name] = true
			}
			// If first test fails, skip second test
			printStatus(ColorYellow, "  ⊙ Skipping second test due to first test failure")
			ts.totalTests++
			ts.failedTests++
			continue
		}

		// Test 2: With lock file (using existing lock file from first test)
		// Don't clean up - keep the lock file and node_modules
		ts.totalTests++
		printStatus(ColorBlue, "\n  ═══ Phase 2: Testing with lock file ═══")
		if err := ts.testRepo(repoPath, repo.Name, true); err == nil {
			ts.successfulTests++
		} else {
			ts.failedTests++
			if repo.Name != "" {
				ts.failedRepos[repo.Name] = true
			}
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
	ts.logMessage(fmt.Sprintf("Repositories: %d", len(ts.Repositories)))
	ts.logMessage(fmt.Sprintf("Total tests (2 per repo): %d", ts.totalTests))
	ts.logMessage(fmt.Sprintf("Successful: %d", ts.successfulTests))
	ts.logMessage(fmt.Sprintf("Failed: %d", ts.failedTests))
	if ts.failedTests > 0 && len(ts.failedRepos) > 0 {
		ts.logMessage("Failed repositories:")
		// Convert map to sorted slice for consistent output
		failedRepoNames := make([]string, 0, len(ts.failedRepos))
		for repoName := range ts.failedRepos {
			failedRepoNames = append(failedRepoNames, repoName)
		}
		sort.Strings(failedRepoNames)
		for _, repoName := range failedRepoNames {
			ts.logMessage(fmt.Sprintf("  • %s", repoName))
		}
	}
	ts.logMessage(fmt.Sprintf("Completed: %s", time.Now().Format("2006-01-02 15:04:05")))
	ts.logMessage("==========================================")

	// Calculate success rate
	successRate := float64(ts.successfulTests) / float64(ts.totalTests) * 100

	printStatus(ColorBlue, fmt.Sprintf("  Repositories: %d", len(ts.Repositories)))
	printStatus(ColorBlue, fmt.Sprintf("  Total tests (2 phases per repo): %d", ts.totalTests))
	printStatus(ColorGreen, fmt.Sprintf("  ✓ Successful: %d", ts.successfulTests))
	if ts.failedTests > 0 {
		printStatus(ColorRed, fmt.Sprintf("  ✗ Failed: %d", ts.failedTests))
		if len(ts.failedRepos) > 0 {
			fmt.Println()
			printStatus(ColorRed, "  Failed repositories:")
			// Convert map to sorted slice for consistent output
			failedRepoNames := make([]string, 0, len(ts.failedRepos))
			for repoName := range ts.failedRepos {
				failedRepoNames = append(failedRepoNames, repoName)
			}
			sort.Strings(failedRepoNames)
			for _, repoName := range failedRepoNames {
				printStatus(ColorRed, fmt.Sprintf("    • %s", repoName))
			}
		}
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

// RunYarnTests runs tests on projects in the tests/yarn directory
func (ts *TestSuite) RunYarnTests() error {
	printStatus(ColorBlue, "==========================================")
	printStatus(ColorBlue, "  NPM-PACKAGER YARN TEST SUITE")
	printStatus(ColorBlue, "==========================================")
	fmt.Println()

	ts.logMessage("NPM-Packager Yarn Test Suite")
	ts.logMessage(fmt.Sprintf("Started: %s", time.Now().Format("2006-01-02 15:04:05")))
	ts.logMessage("")

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

	// Find all subdirectories in the yarn test directory
	entries, err := os.ReadDir(ts.YarnTestDir)
	if err != nil {
		return fmt.Errorf("failed to read yarn test directory: %w", err)
	}

	var testProjects []string
	for _, entry := range entries {
		if entry.IsDir() {
			projectPath := filepath.Join(ts.YarnTestDir, entry.Name())
			packageJsonPath := filepath.Join(projectPath, "package.json")
			if _, err := os.Stat(packageJsonPath); err == nil {
				testProjects = append(testProjects, entry.Name())
			}
		}
	}

	if len(testProjects) == 0 {
		printStatus(ColorYellow, "⚠  No test projects found in yarn directory")
		printStatus(ColorYellow, fmt.Sprintf("  Add directories with package.json files to: %s", ts.YarnTestDir))
		return nil
	}

	printStatus(ColorGreen, fmt.Sprintf("📋 Found %d yarn test projects", len(testProjects)))
	fmt.Println()

	// Process each test project (2 tests per project: without lock file, then with lock file)
	for i, projectName := range testProjects {
		fmt.Println()
		printStatus(ColorBlue, "┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		printStatus(ColorBlue, fmt.Sprintf("┃ [%d/%d] Yarn Project: %s", i+1, len(testProjects), projectName))
		printStatus(ColorBlue, "┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		projectPath := filepath.Join(ts.YarnTestDir, projectName)

		// Clean up before first test
		ts.cleanupRepo(projectPath, projectName)

		// Test 1: Without lock file (fresh install)
		ts.totalTests++
		printStatus(ColorBlue, "\n  ═══ Phase 1: Testing without lock file ═══")
		if err := ts.testRepo(projectPath, projectName, false); err == nil {
			ts.successfulTests++
		} else {
			ts.failedTests++
			ts.failedRepos[projectName] = true
			// If first test fails, skip second test
			printStatus(ColorYellow, "  ⊙ Skipping second test due to first test failure")
			ts.totalTests++
			ts.failedTests++
			continue
		}

		// Test 2: With lock file (using existing lock file from first test)
		ts.totalTests++
		printStatus(ColorBlue, "\n  ═══ Phase 2: Testing with lock file ═══")
		if err := ts.testRepo(projectPath, projectName, true); err == nil {
			ts.successfulTests++
		} else {
			ts.failedTests++
			ts.failedRepos[projectName] = true
		}
	}

	// Print summary
	ts.printSummary(len(testProjects), "Yarn Projects")

	if ts.failedTests > 0 {
		return fmt.Errorf("yarn test suite failed with %d failures", ts.failedTests)
	}

	return nil
}

// printSummary prints the test summary
func (ts *TestSuite) printSummary(totalItems int, itemType string) {
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
	ts.logMessage(fmt.Sprintf("%s: %d", itemType, totalItems))
	ts.logMessage(fmt.Sprintf("Total tests (2 per item): %d", ts.totalTests))
	ts.logMessage(fmt.Sprintf("Successful: %d", ts.successfulTests))
	ts.logMessage(fmt.Sprintf("Failed: %d", ts.failedTests))
	if ts.failedTests > 0 && len(ts.failedRepos) > 0 {
		ts.logMessage("Failed items:")
		failedNames := make([]string, 0, len(ts.failedRepos))
		for name := range ts.failedRepos {
			failedNames = append(failedNames, name)
		}
		sort.Strings(failedNames)
		for _, name := range failedNames {
			ts.logMessage(fmt.Sprintf("  • %s", name))
		}
	}
	ts.logMessage(fmt.Sprintf("Completed: %s", time.Now().Format("2006-01-02 15:04:05")))
	ts.logMessage("==========================================")

	// Calculate success rate
	successRate := float64(0)
	if ts.totalTests > 0 {
		successRate = float64(ts.successfulTests) / float64(ts.totalTests) * 100
	}

	printStatus(ColorBlue, fmt.Sprintf("  %s: %d", itemType, totalItems))
	printStatus(ColorBlue, fmt.Sprintf("  Total tests (2 phases per item): %d", ts.totalTests))
	printStatus(ColorGreen, fmt.Sprintf("  ✓ Successful: %d", ts.successfulTests))
	if ts.failedTests > 0 {
		printStatus(ColorRed, fmt.Sprintf("  ✗ Failed: %d", ts.failedTests))
		if len(ts.failedRepos) > 0 {
			fmt.Println()
			printStatus(ColorRed, "  Failed items:")
			failedNames := make([]string, 0, len(ts.failedRepos))
			for name := range ts.failedRepos {
				failedNames = append(failedNames, name)
			}
			sort.Strings(failedNames)
			for _, name := range failedNames {
				printStatus(ColorRed, fmt.Sprintf("    • %s", name))
			}
		}
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
}

func main() {
	// Define command-line flags
	yarnOnly := flag.Bool("yarn", false, "Run only yarn tests from tests/yarn directory")
	reposOnly := flag.Bool("repos", false, "Run only repository tests")
	flag.Parse()

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

	var runErr error

	// Determine which tests to run based on flags
	if *yarnOnly {
		// Run only yarn tests
		runErr = suite.RunYarnTests()
	} else if *reposOnly {
		// Run only repository tests
		runErr = suite.Run()
	} else {
		// Run both (default behavior - repos first, then yarn)
		runErr = suite.Run()
		if runErr == nil {
			// Reset counters for yarn tests
			suite.totalTests = 0
			suite.successfulTests = 0
			suite.failedTests = 0
			suite.failedRepos = make(map[string]bool)
			runErr = suite.RunYarnTests()
		}
	}

	if runErr != nil {
		os.Exit(1)
	}
}
