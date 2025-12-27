package packagejson

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ernesto27/go-npm/config"
	"github.com/ernesto27/go-npm/yarnlock"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const (
	LOCK_FILE_NAME_GO_NPM = "go-npm-lock.json"
	LOCK_FILE_NAME_NPM    = "package-lock.json"
	LOCK_FILE_NAME_YARN   = "yarn.lock"
)

type Dependency struct {
	Name       string
	ActualName string
	Version    string
	Etag       string
	Nested     bool
}

type PackageJSON struct {
	Name                 string              `json:"name"`
	Description          string              `json:"description"`
	Version              any                 `json:"version"`
	Author               any                 `json:"author"`
	Contributors         any                 `json:"contributors"`
	License              any                 `json:"license"`
	Repository           any                 `json:"repository"`
	Homepage             any                 `json:"homepage"`
	Funding              any                 `json:"funding"`
	Keywords             any                 `json:"keywords"`
	Dependencies         any                 `json:"dependencies"`
	DevDependencies      any                 `json:"devDependencies"`
	OptionalDependencies any                 `json:"optionalDependencies"`
	PeerDependencies     any                 `json:"peerDependencies"`
	PeerDependenciesMeta map[string]PeerMeta `json:"peerDependenciesMeta"`
	Engines              any                 `json:"engines"`
	Files                any                 `json:"files"`
	Scripts              map[string]string   `json:"scripts"`
	Main                 any                 `json:"main"`
	Bin                  any                 `json:"bin"`
	Types                string              `json:"types"`
	Exports              any                 `json:"exports"`
	Private              bool                `json:"private"`
	Workspaces           any                 `json:"workspaces"`
	TrustedDependencies  []string            `json:"trustedDependencies"`
}

type Funding struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

type PeerMeta struct {
	Optional bool `json:"optional"`
}

func (p *PackageJSON) GetDependencies() map[string]string {
	return extractDependencyMap(p.Dependencies)
}

func (p *PackageJSON) GetDevDependencies() map[string]string {
	return extractDependencyMap(p.DevDependencies)
}

func (p *PackageJSON) GetOptionalDependencies() map[string]string {
	return extractDependencyMap(p.OptionalDependencies)
}

func (p *PackageJSON) GetPeerDependencies() map[string]string {
	return extractDependencyMap(p.PeerDependencies)
}

// GetWorkspaces extracts workspace patterns from package.json
// Supports both array format: ["packages/*"] and object format: {"packages": ["packages/*"]}
func (p *PackageJSON) GetWorkspaces() []string {
	if p.Workspaces == nil {
		return []string{}
	}

	// Handle array format: ["packages/*"]
	if patterns, ok := p.Workspaces.([]any); ok {
		result := make([]string, 0, len(patterns))
		for _, pattern := range patterns {
			if str, ok := pattern.(string); ok {
				result = append(result, str)
			}
		}
		return result
	}

	// Handle object format: {"packages": ["packages/*"]}
	if obj, ok := p.Workspaces.(map[string]any); ok {
		if packages, ok := obj["packages"].([]any); ok {
			result := make([]string, 0, len(packages))
			for _, pattern := range packages {
				if str, ok := pattern.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}

	return []string{}
}

func (p *PackageJSON) GetTrustedDependencies() []string {
	if p.TrustedDependencies == nil {
		return []string{}
	}
	return p.TrustedDependencies
}

func extractDependencyMap(deps any) map[string]string {
	if deps == nil {
		return make(map[string]string)
	}

	if m, ok := deps.(map[string]interface{}); ok {
		result := make(map[string]string)
		for k, v := range m {
			if str, ok := v.(string); ok {
				result[k] = str
			}
		}
		return result
	}

	if m, ok := deps.(map[string]string); ok {
		return m
	}

	return make(map[string]string)
}

type PackageJSONParser struct {
	Config                *config.Config
	LockFileName          string
	PackageJSONRoot       *PackageJSON
	OriginalContentRoot   []byte
	PackageJSON           *PackageJSON
	PackageLock           *PackageLock
	FilePath              string
	OriginalContent       []byte
	LockFileContent       []byte
	LockFileContentGlobal []byte
	YarnLockParser        *yarnlock.YarnLockParser
}

type PackageLock struct {
	Name                 string                 `json:"name"`
	Version              string                 `json:"version"`
	LockfileVersion      int                    `json:"lockfileVersion"`
	Requires             bool                   `json:"requires"`
	Workspaces           map[string]string      `json:"workspaces,omitempty"`
	Dependencies         map[string]string      `json:"dependencies"`
	DevDependencies      map[string]string      `json:"devDependencies,omitempty"`
	OptionalDependencies map[string]string      `json:"optionalDependencies,omitempty"`
	PeerDependencies     map[string]string      `json:"peerDependencies,omitempty"`
	Packages             map[string]PackageItem `json:"packages"`
}

type PackageItem struct {
	Name                 string              `json:"name,omitempty"`
	Version              string              `json:"version,omitempty"`
	Resolved             string              `json:"resolved,omitempty"`
	Link                 bool                `json:"link,omitempty"`
	Integrity            string              `json:"integrity,omitempty"`
	License              any                 `json:"license,omitempty"`
	Etag                 string              `json:"etag,omitempty"`
	Dependencies         map[string]string   `json:"dependencies,omitempty"`
	DevDependencies      map[string]string   `json:"devDependencies,omitempty"`
	OptionalDependencies map[string]string   `json:"optionalDependencies,omitempty"`
	PeerDependencies     map[string]string   `json:"peerDependencies,omitempty"`
	PeerDependenciesMeta map[string]PeerMeta `json:"peerDependenciesMeta,omitempty"`
	Optional             bool                `json:"optional,omitempty"`
	Dev                  bool                `json:"dev,omitempty"`
	Bin                  any                 `json:"bin,omitempty"`
	Engines              any                 `json:"engines,omitempty"`
	OS                   []string            `json:"os,omitempty"`
	CPU                  []string            `json:"cpu,omitempty"`
	Scripts              map[string]string   `json:"scripts,omitempty"`
}

func NewPackageJSONParser(cfg *config.Config, yarnParser *yarnlock.YarnLockParser) *PackageJSONParser {
	return &PackageJSONParser{
		Config:         cfg,
		LockFileName:   LOCK_FILE_NAME_GO_NPM,
		YarnLockParser: yarnParser,
	}
}

func (p *PackageJSONParser) Parse(filePath string) (*PackageJSON, error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	var packageJSON PackageJSON
	if err := json.Unmarshal(fileContent, &packageJSON); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from file %s: %w", filePath, err)
	}

	if filePath == "package.json" {
		p.PackageJSONRoot = &packageJSON
		p.OriginalContentRoot = fileContent
	} else {
		p.PackageJSON = &packageJSON
		p.OriginalContent = fileContent

	}
	p.FilePath = filePath

	lockFileContent, err := os.ReadFile(p.LockFileName)
	if err == nil {
		var packageLock PackageLock
		if err := json.Unmarshal(lockFileContent, &packageLock); err == nil {
			p.PackageLock = &packageLock
			p.LockFileContent = lockFileContent
		}
	}

	return &packageJSON, nil
}

func (p *PackageJSONParser) ParseDefault() (*PackageJSON, error) {
	return p.Parse("package.json")
}

func (p *PackageJSONParser) ParseLockFile() (*PackageLock, error) {
	file, err := os.Open(p.LockFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", p.LockFileName, err)
	}
	defer file.Close()

	var packageLock PackageLock

	if err := json.NewDecoder(file).Decode(&packageLock); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from file %s: %w", p.LockFileName, err)
	}

	return &packageLock, nil
}

func (p *PackageJSONParser) CreateLockFile(data *PackageLock, isGlobal bool) error {
	lockFile := p.LockFileName
	if isGlobal {
		lockFile = p.Config.GlobalLockFile
	}

	file, err := os.Create(lockFile)

	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", lockFile, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to write JSON to file %s: %w", lockFile, err)
	}

	p.PackageLock = data

	return nil
}

func (p *PackageJSONParser) UpdateLockFile(data *PackageLock, isGlobal bool) error {
	lockFileContent := p.LockFileContent
	lockFileName := p.LockFileName

	if isGlobal {
		lockFileContent = p.LockFileContentGlobal
		lockFileName = p.Config.GlobalLockFile
	}

	if lockFileContent == nil {
		return fmt.Errorf("lock file content not cached, call Parse() first")
	}

	var existingLock PackageLock
	if err := json.Unmarshal(lockFileContent, &existingLock); err != nil {
		return fmt.Errorf("failed to parse existing lock file: %w", err)
	}

	for key, version := range data.Dependencies {
		existingLock.Dependencies[key] = version
	}

	for key, version := range data.OptionalDependencies {
		if existingLock.OptionalDependencies == nil {
			existingLock.OptionalDependencies = make(map[string]string)
		}
		existingLock.OptionalDependencies[key] = version
	}

	if existingLock.Packages == nil {
		existingLock.Packages = make(map[string]PackageItem)
	}

	for key, packageItem := range data.Packages {
		_, ok := existingLock.Packages[key]
		if ok {
			p.resolveVersionMismatch(&existingLock, key, packageItem)
		}
		existingLock.Packages[key] = packageItem
	}

	updatedContent, err := json.MarshalIndent(existingLock, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated lock file: %w", err)
	}

	if err := os.WriteFile(lockFileName, updatedContent, 0644); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	p.PackageLock = &existingLock
	if isGlobal {
		p.LockFileContentGlobal = updatedContent
	} else {
		p.LockFileContent = updatedContent
	}

	return nil
}

func (p *PackageJSONParser) resolveVersionMismatch(existingLock *PackageLock, key string, packageItem PackageItem) {
	for keyp, p := range existingLock.Packages {
		if p.Dependencies != nil {
			for depName := range p.Dependencies {
				if depName == packageItem.Name {
					nestedKey := keyp + "/node_modules/" + packageItem.Name
					existingLock.Packages[nestedKey] = existingLock.Packages[key]
					delete(existingLock.Packages, key)
				}
			}
		}
	}
}

func (p *PackageJSONParser) AddOrUpdateDependency(name string, version string) error {
	if p.PackageJSONRoot == nil {
		return fmt.Errorf("package.json not loaded, call Parse() first")
	}

	if p.OriginalContentRoot == nil {
		return fmt.Errorf("original content not cached, call Parse() first")
	}

	deps := p.PackageJSONRoot.GetDependencies()

	if version == "" || version == "latest" {
		if existingVersion, exists := p.PackageLock.Packages[name]; exists {
			version = existingVersion.Resolved
		}
	}

	deps[name] = version
	p.PackageJSONRoot.Dependencies = deps

	// Check if dependency already exists (using cached content)
	jsonStr := string(p.OriginalContentRoot)
	existingValue := gjson.Get(jsonStr, "dependencies."+name)
	isNewDependency := !existingValue.Exists()

	// Use sjson to update the dependency
	var err error
	jsonStr, err = sjson.SetRaw(jsonStr, "dependencies."+name, fmt.Sprintf(`"%s"`, version))
	if err != nil {
		return fmt.Errorf("failed to update dependency: %w", err)
	}

	// Fix formatting if it's a new dependency (sjson adds it incorrectly)
	if isNewDependency {
		malformed := "\n  ,\"" + name + `":"` + version + `"}`
		wellFormed := `,` + "\n" + `    "` + name + `": "` + version + `"` + "\n  }"
		jsonStr = strings.Replace(jsonStr, malformed, wellFormed, 1)
	}

	// Write back to file
	if err := os.WriteFile("package.json", []byte(jsonStr), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", p.FilePath, err)
	}

	// Update cached content for subsequent calls
	p.OriginalContentRoot = []byte(jsonStr)

	return nil
}

func (p *PackageJSONParser) ResolveDependencies() (toInstall []Dependency, toRemove []Dependency) {
	toInstall = []Dependency{}
	toRemove = []Dependency{}

	for name, versionInJSON := range p.PackageJSON.GetDependencies() {
		versionInLock, exists := p.PackageLock.Dependencies[name]
		if !exists || versionInJSON != versionInLock {
			toInstall = append(toInstall, Dependency{
				Name:    name,
				Version: versionInJSON,
			})
		}
	}

	for name, versionInJSON := range p.PackageJSON.GetDevDependencies() {
		versionInLock, exists := p.PackageLock.DevDependencies[name]
		if !exists || versionInJSON != versionInLock {
			toInstall = append(toInstall, Dependency{
				Name:    name,
				Version: versionInJSON,
			})
		}
	}

	for name, versionInJSON := range p.PackageJSON.GetOptionalDependencies() {
		versionInLock, exists := p.PackageLock.OptionalDependencies[name]
		if !exists || versionInJSON != versionInLock {
			toInstall = append(toInstall, Dependency{
				Name:    name,
				Version: versionInJSON,
			})
		}
	}

	for name, versionInLock := range p.PackageLock.Dependencies {
		deps := p.PackageJSON.GetDependencies()
		devDeps := p.PackageJSON.GetDevDependencies()
		optionalDeps := p.PackageJSON.GetOptionalDependencies()

		_, existsInDeps := deps[name]
		_, existsInDevDeps := devDeps[name]
		_, existsInOptionalDeps := optionalDeps[name]

		if !existsInDeps && !existsInDevDeps && !existsInOptionalDeps {
			toRemove = append(toRemove, Dependency{
				Name:    name,
				Version: versionInLock,
			})
		}
	}

	return toInstall, toRemove
}

func (p *PackageJSONParser) ResolveDependenciesToRemove(pkg string) []string {
	pkgToKeep := make(map[string]bool)

	for directDep := range p.PackageLock.Dependencies {
		if directDep == pkg {
			continue
		}

		visited := make(map[string]bool)
		queue := []string{directDep}

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]

			if visited[current] {
				continue
			}
			visited[current] = true
			pkgToKeep[current] = true

			pkgPath := "node_modules/" + current
			pkgItem := p.PackageLock.Packages[pkgPath]

			for childDep := range pkgItem.Dependencies {
				if !visited[childDep] {
					queue = append(queue, childDep)
				}
			}
		}
	}

	pkgToRemove := []string{}
	visited := make(map[string]bool)
	queue := []string{pkg}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		pkgPath := "node_modules/" + current
		pkgItem := p.PackageLock.Packages[pkgPath]

		if !pkgToKeep[current] {
			pkgToRemove = append(pkgToRemove, current)
		}

		for childDep := range pkgItem.Dependencies {
			if !visited[childDep] {
				queue = append(queue, childDep)
			}
		}
	}

	return pkgToRemove
}

func (p *PackageJSONParser) RemoveDependencies(pkg string) error {
	if p.PackageJSON == nil {
		return fmt.Errorf("package.json not loaded, call Parse() first")
	}

	deps := p.PackageJSON.GetDependencies()
	if len(deps) == 0 {
		return fmt.Errorf("no dependencies found in package.json")
	}

	_, exists := deps[pkg]
	if !exists {
		return fmt.Errorf("dependency '%s' not found in package.json", pkg)
	}

	jsonStr := string(p.OriginalContent)
	var err error
	jsonStr, err = sjson.Delete(jsonStr, "dependencies."+pkg)
	if err != nil {
		return fmt.Errorf("failed to remove dependency from package.json: %w", err)
	}

	if err := os.WriteFile(p.FilePath, []byte(jsonStr), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", p.FilePath, err)
	}

	delete(deps, pkg)
	p.PackageJSON.Dependencies = deps
	p.OriginalContent = []byte(jsonStr)

	return nil
}

func (p *PackageJSONParser) RemoveFromLockFile(pkg string, pkgToRemove []string, isGlobal bool) error {
	if p.PackageLock == nil {
		return fmt.Errorf("package lock not loaded")
	}

	delete(p.PackageLock.Dependencies, pkg)

	for _, pkgName := range pkgToRemove {
		delete(p.PackageLock.Packages, "node_modules/"+pkgName)
	}

	packagesToDelete := []string{}
	for key := range p.PackageLock.Packages {
		for _, pkgName := range pkgToRemove {
			if strings.Contains(key, "/node_modules/"+pkgName) {
				packagesToDelete = append(packagesToDelete, key)
			}
		}
	}
	for _, key := range packagesToDelete {
		delete(p.PackageLock.Packages, key)
	}

	err := p.CreateLockFile(p.PackageLock, isGlobal)
	if err != nil {
		return err
	}

	return nil
}

func (p *PackageJSONParser) MigrateFromPackageLock() error {
	lockData, err := os.ReadFile(LOCK_FILE_NAME_NPM)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", LOCK_FILE_NAME_NPM, err)
	}

	var packageLock PackageLock
	if err = json.Unmarshal(lockData, &packageLock); err != nil {
		return fmt.Errorf("failed to parse %s: %w", LOCK_FILE_NAME_NPM, err)
	}

	for key, item := range packageLock.Packages {
		if key == "" {
			packageLock.Dependencies = item.Dependencies
			packageLock.DevDependencies = item.DevDependencies
			delete(packageLock.Packages, key)
		}
	}

	err = p.CreateLockFile(&packageLock, false)
	if err != nil {
		return fmt.Errorf("failed to create go-npm lock file: %w", err)
	}

	p.PackageLock = &packageLock
	p.LockFileContent = lockData

	return nil
}

// MigrateFromYarnLock converts yarn.lock (v1) to go-npm-lock.json
func (p *PackageJSONParser) MigrateFromYarnLock() error {
	if p.YarnLockParser == nil {
		return fmt.Errorf("yarn lock parser not initialized")
	}

	lockData, err := os.ReadFile(LOCK_FILE_NAME_YARN)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", LOCK_FILE_NAME_YARN, err)
	}

	if !p.YarnLockParser.IsYarnV1(lockData) {
		return fmt.Errorf("unsupported yarn.lock format: only v1 is supported")
	}

	yarnLock, err := p.YarnLockParser.ParseContent(lockData)
	if err != nil {
		return fmt.Errorf("failed to parse yarn.lock: %w", err)
	}

	packageLock := p.convertYarnToPackageLock(yarnLock)

	err = p.CreateLockFile(packageLock, false)
	if err != nil {
		return fmt.Errorf("failed to create go-npm lock file: %w", err)
	}

	p.PackageLock = packageLock
	p.LockFileContent = lockData

	return nil
}

// convertYarnToPackageLock converts YarnLock to PackageLock format
func (p *PackageJSONParser) convertYarnToPackageLock(yarnLock *yarnlock.YarnLock) *PackageLock {
	packageLock := &PackageLock{
		Name:            p.PackageJSON.Name,
		LockfileVersion: 3,
		Requires:        true,
		Packages:        make(map[string]PackageItem),
		Dependencies:    make(map[string]string),
	}

	// Get top-level dependencies from package.json
	if p.PackageJSON != nil {
		deps := p.PackageJSON.GetDependencies()
		for name, version := range deps {
			packageLock.Dependencies[name] = version
		}

		devDeps := p.PackageJSON.GetDevDependencies()
		if len(devDeps) > 0 {
			packageLock.DevDependencies = make(map[string]string)
			for name, version := range devDeps {
				packageLock.DevDependencies[name] = version
			}
		}

		optDeps := p.PackageJSON.GetOptionalDependencies()
		if len(optDeps) > 0 {
			packageLock.OptionalDependencies = make(map[string]string)
			for name, version := range optDeps {
				packageLock.OptionalDependencies[name] = version
			}
		}
	}

	// Convert each yarn entry to a PackageItem
	for _, entry := range yarnLock.Entries {
		pkgPath := "node_modules/" + entry.Name

		packageItem := PackageItem{
			Name:      entry.Name,
			Version:   entry.Version,
			Resolved:  entry.Resolved,
			Integrity: entry.Integrity,
		}

		// Copy dependencies
		if len(entry.Dependencies) > 0 {
			packageItem.Dependencies = make(map[string]string)
			for depName, depVersion := range entry.Dependencies {
				packageItem.Dependencies[depName] = depVersion
			}
		}

		// Copy optional dependencies
		if len(entry.OptionalDependencies) > 0 {
			packageItem.OptionalDependencies = make(map[string]string)
			for depName, depVersion := range entry.OptionalDependencies {
				packageItem.OptionalDependencies[depName] = depVersion
			}
		}

		packageLock.Packages[pkgPath] = packageItem
	}

	return packageLock
}
