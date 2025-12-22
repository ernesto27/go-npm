package list

import (
	"fmt"
	"sort"
	"strings"

	"github.com/ernesto27/go-npm/packagejson"
)

type Lister struct {
	Lock        *packagejson.PackageLock
	ProjectName string
	Version     string
	ShowAll     bool
}

func New(lock *packagejson.PackageLock, projectName, version string) *Lister {
	return &Lister{
		Lock:        lock,
		ProjectName: projectName,
		Version:     version,
	}
}

func (l *Lister) Print() {
	l.printHeader()
	l.printDependencies()
	l.printSummary()
}

func (l *Lister) printHeader() {
	if l.Version != "" {
		fmt.Printf("%s@%s\n", l.ProjectName, l.Version)
	} else {
		fmt.Println(l.ProjectName)
	}
}

func (l *Lister) printDependencies() {
	// Collect all top-level dependencies
	allDeps := make(map[string]bool)
	for name := range l.Lock.Dependencies {
		allDeps[name] = false // false = production
	}
	for name := range l.Lock.DevDependencies {
		if _, exists := allDeps[name]; !exists {
			allDeps[name] = true // true = dev
		}
	}

	// Sort dependency names
	names := make([]string, 0, len(allDeps))
	for name := range allDeps {
		names = append(names, name)
	}
	sort.Strings(names)

	// Print each dependency
	for i, name := range names {
		isLast := i == len(names)-1
		pkgPath := "node_modules/" + name
		if item, exists := l.Lock.Packages[pkgPath]; exists {
			prefix := "├──"
			if isLast {
				prefix = "└──"
			}
			isDev := allDeps[name]
			l.printPackage(name, item.Version, pkgPath, prefix, "", isDev, 0)
		}
	}
}

func (l *Lister) printPackage(name, version, pkgPath, prefix, indent string, isDev bool, depth int) {
	devLabel := ""
	if isDev && depth == 0 {
		devLabel = " (dev)"
	}
	fmt.Printf("%s%s %s@%s%s\n", indent, prefix, name, version, devLabel)

	if !l.ShowAll {
		return
	}

	item, exists := l.Lock.Packages[pkgPath]
	if !exists || len(item.Dependencies) == 0 {
		return
	}

	// Sort sub-dependencies
	subDeps := make([]string, 0, len(item.Dependencies))
	for depName := range item.Dependencies {
		subDeps = append(subDeps, depName)
	}
	sort.Strings(subDeps)

	// Calculate new indent
	newIndent := indent
	if strings.HasPrefix(prefix, "├") {
		newIndent += "│   "
	} else if strings.HasPrefix(prefix, "└") {
		newIndent += "    "
	}

	for i, depName := range subDeps {
		isLast := i == len(subDeps)-1
		subPrefix := "├──"
		if isLast {
			subPrefix = "└──"
		}

		// Try nested path first, then top-level
		depPath := pkgPath + "/node_modules/" + depName
		depItem, depExists := l.Lock.Packages[depPath]
		if !depExists {
			depPath = "node_modules/" + depName
			depItem, depExists = l.Lock.Packages[depPath]
		}
		if depExists {
			l.printPackage(depName, depItem.Version, depPath, subPrefix, newIndent, false, depth+1)
		}
	}
}

func (l *Lister) printSummary() {
	fmt.Printf("\n%d packages\n", len(l.Lock.Packages))
}
