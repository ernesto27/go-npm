package progress

import (
	"fmt"
	"sync"
	"time"

	"github.com/briandowns/spinner"
)

type PackageInfo struct {
	Name    string
	Version string
}

type Progress struct {
	spinner    *spinner.Spinner
	startTime  time.Time
	topLevel   []PackageInfo
	totalCount int
	mu         sync.Mutex
	version    string
	verbose    bool
}

// New creates a new Progress instance with the given version
func New(version string, verbose bool) *Progress {
	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond)
	s.Color("cyan")

	return &Progress{
		spinner:  s,
		topLevel: make([]PackageInfo, 0),
		version:  version,
		verbose:  verbose,
	}
}

// Start prints the header and starts the spinner
func (p *Progress) Start() {
	p.startTime = time.Now()
	fmt.Printf("go-npm install %s\n\n", p.version)
	p.spinner.Suffix = " Resolving dependencies..."
	p.spinner.Start()
}

// SetStatus updates the spinner status message
// When verbose mode is enabled, it also prints the message to stdout
func (p *Progress) SetStatus(msg string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.spinner.Suffix = " " + msg

	if p.verbose {
		p.spinner.Stop()
		fmt.Printf("  %s\n", msg)
		p.spinner.Start()
	}
}

// AddTopLevel adds a top-level package to be shown in the summary
func (p *Progress) AddTopLevel(name, version string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.topLevel = append(p.topLevel, PackageInfo{Name: name, Version: version})
}

// IncrementCount increments the total package count
func (p *Progress) IncrementCount() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.totalCount++
}

// Finish stops the spinner and prints the final summary
func (p *Progress) Finish() {
	p.spinner.Stop()

	// Print top-level packages with + prefix
	for _, pkg := range p.topLevel {
		fmt.Printf("+ %s@%s\n", pkg.Name, pkg.Version)
	}

	if len(p.topLevel) > 0 {
		fmt.Println()
	}

	// Print summary
	duration := time.Since(p.startTime)
	fmt.Printf("%d packages installed [%.2fs]\n", p.totalCount, duration.Seconds())
}

// Warn prints a warning message (doesn't interrupt spinner)
func (p *Progress) Warn(format string, args ...interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Temporarily stop spinner to print warning cleanly
	p.spinner.Stop()
	fmt.Printf("warning: "+format+"\n", args...)
	p.spinner.Start()
}

