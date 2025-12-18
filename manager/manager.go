package manager

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ernesto27/go-npm/binlink"
	"github.com/ernesto27/go-npm/config"
	"github.com/ernesto27/go-npm/etag"
	"github.com/ernesto27/go-npm/extractor"
	manifestpkg "github.com/ernesto27/go-npm/manifest"
	"github.com/ernesto27/go-npm/packagecopy"
	"github.com/ernesto27/go-npm/packagejson"
	"github.com/ernesto27/go-npm/tarball"
	"github.com/ernesto27/go-npm/utils"
	"github.com/ernesto27/go-npm/version"
	"github.com/ernesto27/go-npm/workspace"
	"github.com/ernesto27/go-npm/yarnlock"
)

const npmRegistryURL = "https://registry.npmjs.org/"

type Job struct {
	Dependency packagejson.Dependency
	ParentName string
	ResultChan chan<- JobResult
}

type JobResult struct {
	Dependency      packagejson.Dependency
	ParentName      string
	NewDependencies map[string]string
	Error           error
}

type PackageManager struct {
	dependencies      map[string]string
	extractedPath     string
	processedPackages map[string]packagejson.Dependency
	configPath        string
	packagesPath      string
	Etag              etag.Etag
	isAdd             bool
	isGlobal          bool
	config            *config.Config
	packages          Packages
	packageLock       *packagejson.PackageLock
	manifest          *manifestpkg.Manifest
	tarball           *tarball.Tarball
	extractor         *extractor.TGZExtractor
	packageCopy       *packagecopy.PackageCopy
	parseJsonManifest *ParseJsonManifest
	versionInfo       *version.Info
	packageJsonParse  *packagejson.PackageJSONParser
	binLinker         *binlink.BinLinker
	workspaceRegistry *workspace.WorkspaceRegistry
	downloadMu        sync.Mutex
	downloadLocks     map[string]*sync.Mutex
}

type Package struct {
	Version            string `json:"version"`
	Nested             bool
	Dependencies       []packagejson.Dependency `json:"dependencies"`
	ParentDependencies []string
}

type Packages map[string]Package

type Dependencies struct {
	Config            *config.Config
	Manifest          *manifestpkg.Manifest
	Etag              *etag.Etag
	Tarball           *tarball.Tarball
	Extractor         *extractor.TGZExtractor
	PackageCopy       *packagecopy.PackageCopy
	ParseJsonManifest *ParseJsonManifest
	VersionInfo       *version.Info
	PackageJsonParse  *packagejson.PackageJSONParser
	BinLinker         *binlink.BinLinker
}

type QueueItem struct {
	Dep            packagejson.Dependency
	ParentName     string
	IsDev          bool
	IsOptional     bool
	IsPeer         bool
	IsPeerOptional bool
}

// generateUniqueTarballName creates a unique tarball filename to avoid collisions
// between scoped and non-scoped packages with the same base name.
// Example: @jest/expect and expect both produce expect-30.2.0.tgz without this
func generateUniqueTarballName(packageName, version string) string {
	safeName := strings.ReplaceAll(packageName, "/", "-")
	return safeName + "-" + version + ".tgz"
}

// parseAliasVersion detects npm package aliases in the format "npm:package@version"
// Returns: actualPackage, version, isAlias
func parseAliasVersion(version string) (string, string, bool) {
	if !strings.HasPrefix(version, "npm:") {
		return "", version, false
	}

	// Parse "npm:@babel/traverse@^7.25.3" or "npm:lodash@^4.17.21"
	spec := strings.TrimPrefix(version, "npm:")

	lastAt := strings.LastIndex(spec, "@")
	if lastAt <= 0 {
		return spec, "latest", true
	}

	if lastAt == 0 {
		return spec, "latest", true
	}

	actualPkg := spec[:lastAt]
	actualVersion := spec[lastAt+1:]

	return actualPkg, actualVersion, true
}

// extractPackageName extracts the package name from a packageResolved path
// Examples:
//
//	"node_modules/foo" → "foo"
//	"node_modules/foo/node_modules/bar" → "bar"
//	"node_modules/@scope/package" → "@scope/package"
func extractPackageName(packageResolved string) string {
	parts := strings.Split(packageResolved, "/node_modules/")
	return parts[len(parts)-1]
}

// GitHubDependency represents a parsed GitHub dependency
type GitHubDependency struct {
	Owner string
	Repo  string
	Ref   string // tag, branch, or commit SHA (empty for default branch)
}

func parseGitHubDependency(version string) (*GitHubDependency, bool) {
	if !strings.HasPrefix(version, "github:") {
		return nil, false
	}

	spec := strings.TrimPrefix(version, "github:")

	parts := strings.SplitN(spec, "#", 2)
	repoPath := parts[0]

	var ref string
	if len(parts) == 2 {
		ref = parts[1]
	}

	repoParts := strings.SplitN(repoPath, "/", 2)
	if len(repoParts) != 2 {
		return nil, false
	}

	return &GitHubDependency{
		Owner: repoParts[0],
		Repo:  repoParts[1],
		Ref:   ref,
	}, true
}

func BuildDependencies() (*Dependencies, error) {
	cfg, err := config.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	manifest, err := manifestpkg.NewManifest(cfg.BaseDir, npmRegistryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create manifest: %w", err)
	}

	etag, err := etag.NewEtag(cfg.BaseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create etag: %w", err)
	}

	return &Dependencies{
		Config:            cfg,
		Manifest:          manifest,
		Etag:              etag,
		Tarball:           tarball.NewTarball(cfg.TarballDir),
		Extractor:         extractor.NewTGZExtractor(),
		PackageCopy:       packagecopy.NewPackageCopy(),
		ParseJsonManifest: newParseJsonManifest(),
		VersionInfo:       version.New(),
		PackageJsonParse:  packagejson.NewPackageJSONParser(cfg, yarnlock.NewYarnLockParser()),
		BinLinker:         binlink.NewBinLinker(cfg.LocalNodeModules),
	}, nil
}

func New(deps *Dependencies) (*PackageManager, error) {
	return &PackageManager{
		dependencies:      make(map[string]string),
		extractedPath:     deps.Config.LocalNodeModules,
		processedPackages: make(map[string]packagejson.Dependency),
		configPath:        deps.Config.BaseDir,
		packagesPath:      deps.Config.PackagesDir,
		Etag:              *deps.Etag,
		isAdd:             false,
		isGlobal:          false,
		config:            deps.Config,
		packages:          make(Packages),
		tarball:           deps.Tarball,
		extractor:         deps.Extractor,
		packageCopy:       deps.PackageCopy,
		manifest:          deps.Manifest,
		parseJsonManifest: deps.ParseJsonManifest,
		versionInfo:       deps.VersionInfo,
		packageJsonParse:  deps.PackageJsonParse,
		binLinker:         deps.BinLinker,
		downloadLocks:     make(map[string]*sync.Mutex),
	}, nil
}

func (pm *PackageManager) SetupGlobal() error {
	// Create global directory first
	if err := utils.CreateDir(pm.config.GlobalDir); err != nil {
		return err
	}

	if err := utils.CreateDir(pm.config.GlobalNodeModules); err != nil {
		return err
	}
	if err := utils.CreateDir(pm.config.GlobalBinDir); err != nil {
		return err
	}

	pm.isGlobal = true
	pm.extractedPath = pm.config.GlobalNodeModules

	pm.binLinker.SetGlobalMode(pm.config.GlobalNodeModules, pm.config.GlobalBinDir)

	// Load existing global lock file if it exists
	if _, err := os.Stat(pm.config.GlobalLockFile); err == nil {
		lockFileContent, err := os.ReadFile(pm.config.GlobalLockFile)
		if err != nil {
			return fmt.Errorf("failed to read global lock file: %w", err)
		}

		var lockFile packagejson.PackageLock
		if err := json.Unmarshal(lockFileContent, &lockFile); err != nil {
			return fmt.Errorf("failed to parse global lock file: %w", err)
		}

		pm.packageJsonParse.LockFileContentGlobal = lockFileContent
		pm.packageJsonParse.PackageLock = &lockFile
		pm.packageLock = &lockFile
	} else {
		// Initialize empty lock file structure for new global installs
		lockFile := &packagejson.PackageLock{
			Name:            "global",
			Version:         "1.0.0",
			LockfileVersion: 3,
			Requires:        true,
			Dependencies:    make(map[string]string),
			DevDependencies: make(map[string]string),
			Packages:        make(map[string]packagejson.PackageItem),
		}
		pm.packageLock = lockFile
		pm.packageJsonParse.PackageLock = lockFile
	}

	return nil
}

func (pm *PackageManager) ParsePackageJSON(isProduction bool) error {
	data, err := pm.packageJsonParse.ParseDefault()
	if err != nil {
		return err
	}

	// Discover workspaces first (needed for both fresh and incremental installs)
	if len(data.GetWorkspaces()) > 0 {
		rootDir, _ := filepath.Abs(".")
		registry := workspace.NewWorkspaceRegistry(rootDir, pm.packageJsonParse)

		if err := registry.Discover(data); err != nil {
			return fmt.Errorf("failed to discover workspaces: %w", err)
		}

		pm.workspaceRegistry = registry

		if errors := registry.Validate(); len(errors) > 0 {
			for _, e := range errors {
				fmt.Printf("Warning: %v\n", e)
			}
		}
	}

	lockFileExists := false

	if pm.packageJsonParse.PackageLock != nil {
		packagesToAdd, packagesToRemove := pm.packageJsonParse.ResolveDependencies()

		for _, pkg := range packagesToAdd {
			err = pm.Add(pkg.Name, pkg.Version, true)
			if err != nil {
				return err
			}
		}

		for _, pkg := range packagesToRemove {
			err = pm.Remove(pkg.Name, false)
			if err != nil {
				return err
			}
		}

		if isProduction && len(pm.packageJsonParse.PackageLock.DevDependencies) > 0 {
			pm.removeDevOnlyPackages()
		}

		pm.packageLock = pm.packageJsonParse.PackageLock

		lockFileExists = true
	} else {
		// Priority 1: Try npm lock file (package-lock.json)
		err := pm.packageJsonParse.MigrateFromPackageLock()
		if err == nil {
			fmt.Println("Migrating from package-lock.json")
			pm.packageLock = pm.packageJsonParse.PackageLock
			lockFileExists = true
		} else {
			// Priority 2: Try yarn.lock (v1 only)
			err = pm.packageJsonParse.MigrateFromYarnLock()
			if err == nil {
				fmt.Println("Migrating from yarn.lock")
				pm.packageLock = pm.packageJsonParse.PackageLock
				lockFileExists = true
			}
		}
	}

	// Create workspace symlinks even when lock file exists
	err = pm.CreateWorkspaceSymlinks()
	if err != nil {
		return err
	}

	if !lockFileExists {
		err = pm.fetchToCache(*data, isProduction)
		if err != nil {
			return err
		}

		err = pm.packageJsonParse.CreateLockFile(pm.packageLock, false)
		if err != nil {
			return err
		}
	}

	return nil
}

func (pm *PackageManager) CreateWorkspaceSymlinks() error {
	if pm.workspaceRegistry == nil {
		return nil
	}

	fmt.Println("Creating workspace symlinks...")

	for _, wsPkg := range pm.workspaceRegistry.Packages {
		err := pm.workspaceRegistry.CreateSymlink(pm.extractedPath, wsPkg.Name, wsPkg.Path)
		if err != nil {
			return fmt.Errorf("failed to create symlink for %s: %w", wsPkg.Name, err)
		}

		fmt.Printf("  ✓ Linked %s\n", wsPkg.Name)
	}

	return nil
}

func (pm *PackageManager) removeDevOnlyPackages() {
	pkgsToRemoveMap := make(map[string]bool)

	for name := range pm.packageJsonParse.PackageLock.DevDependencies {
		pkgToRemove := pm.packageJsonParse.ResolveDependenciesToRemove(name)

		for _, pkg := range pkgToRemove {
			pkgsToRemoveMap[pkg] = true
			delete(pm.packageJsonParse.PackageLock.Dependencies, pkg)
		}
	}

	pathsToDelete := []string{}
	for pkgPath := range pm.packageJsonParse.PackageLock.Packages {
		shouldDelete := false

		pkgName := strings.TrimPrefix(pkgPath, "node_modules/")
		if strings.Contains(pkgName, "/node_modules/") {
			parts := strings.Split(pkgName, "/node_modules/")
			pkgName = parts[len(parts)-1]
		}

		if pkgsToRemoveMap[pkgName] {
			shouldDelete = true
		}

		for pkg := range pkgsToRemoveMap {
			prefix := "node_modules/" + pkg + "/node_modules/"
			if strings.HasPrefix(pkgPath, prefix) {
				shouldDelete = true
				break
			}
		}

		if shouldDelete {
			pathsToDelete = append(pathsToDelete, pkgPath)
		}
	}

	for _, pkgPath := range pathsToDelete {
		delete(pm.packageJsonParse.PackageLock.Packages, pkgPath)
	}
}

func (pm *PackageManager) InstallFromCache() error {
	packagesToInstall := make(map[string]packagejson.PackageItem)
	for pkgPath := range pm.packageLock.Packages {
		item := pm.packageLock.Packages[pkgPath]

		if item.Link {
			continue
		}

		namePkg := strings.TrimPrefix(pkgPath, "node_modules/")
		if strings.Contains(namePkg, "/node_modules/") {
			parts := strings.Split(namePkg, "/node_modules/")
			namePkg = parts[len(parts)-1]
		}

		targetPath := path.Join(pm.extractedPath, namePkg)
		exists := utils.FolderExists(targetPath)
		if !exists {
			packagesToInstall[pkgPath] = item
		}
	}

	var wg sync.WaitGroup
	errChan := make(chan error, len(packagesToInstall))
	for name, item := range packagesToInstall {
		if name == "" {
			continue
		}

		wg.Add(1)
		go func(name string, item packagejson.PackageItem) {
			defer wg.Done()

			namePkg := strings.TrimPrefix(name, "node_modules/")
			pkgName := namePkg
			if strings.Contains(namePkg, "/node_modules/") {
				parts := strings.Split(namePkg, "/node_modules/")
				pkgName = parts[len(parts)-1]
			}

			pathPkg := path.Join(pm.packagesPath, pkgName+"@"+item.Version)

			exists := utils.FolderExists(pathPkg)
			if !exists {
				if item.Resolved == "" {
					fmt.Printf("Skipping package %s - empty resolved URL in lock file\n", item.Name)
					return
				}

				// Check if this is a git URL and convert to tarball URL if needed
				downloadURL := item.Resolved
				tarballFilename := generateUniqueTarballName(pkgName, item.Version)

				if tarballURL, filename, isGit := convertGitURLToTarball(item.Resolved); isGit {
					downloadURL = tarballURL
					tarballFilename = filename
					fmt.Printf("Converting git URL to tarball for %s\n", pkgName)
				}

				// Lock based on package@version to prevent concurrent extractions to the same directory
				// Use the same locking key as fetchToCache to prevent race conditions
				packageKey := pkgName + "@" + item.Version
				pm.downloadMu.Lock()
				packageLock_, exists := pm.downloadLocks[packageKey]
				if !exists {
					packageLock_ = &sync.Mutex{}
					pm.downloadLocks[packageKey] = packageLock_
				}
				pm.downloadMu.Unlock()

				packageLock_.Lock()

				// Double-check folder existence after acquiring lock
				if !utils.FolderExists(pathPkg) {
					tarballPath := filepath.Join(pm.tarball.TarballPath, tarballFilename)

					// Validate tarball (checks existence and integrity)
					shouldDownload := true
					if utils.ValidateTarball(tarballPath) {
						shouldDownload = false
					} else {
						os.Remove(tarballPath)
					}

					if shouldDownload {
						err := pm.tarball.DownloadAs(downloadURL, tarballFilename)
						if err != nil {
							packageLock_.Unlock()
							errChan <- err
							return
						}
					}

					err := pm.extractor.Extract(tarballPath, pathPkg)
					if err != nil {
						packageLock_.Unlock()
						errChan <- err
						return
					}
				}
				packageLock_.Unlock()
			}

			targetPath := path.Join(pm.extractedPath, namePkg)
			err := pm.packageCopy.CopyDirectory(pathPkg, targetPath)
			if err != nil {
				errChan <- err
				return
			}
		}(name, item)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		return err
	}

	if err := pm.binLinker.LinkAllPackages(); err != nil {
		return fmt.Errorf("failed to link bin executables: %w", err)
	}

	return nil
}

func (pm *PackageManager) removePackagesFromNodeModules(pkgList []string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(pkgList))

	for _, pkg := range pkgList {
		wg.Add(1)
		go func(pkgName string) {
			defer wg.Done()

			pkgPath := filepath.Join(pm.extractedPath, pkgName)

			if err := os.RemoveAll(pkgPath); err != nil {
				errChan <- fmt.Errorf("failed to remove package %s: %w", pkgName, err)
			}
		}(pkg)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		return err
	}

	return nil
}

func (pm *PackageManager) Add(pkgName string, version string, isInstall bool) error {
	packageJson, err := pm.packageJsonParse.ParseDefault()
	if err != nil {
		return err
	}

	if !isInstall {
		deps := packageJson.GetDependencies()
		if _, exists := deps[pkgName]; exists {
			if version != "" && deps[pkgName] == version {
				fmt.Println("Package", pkgName, "already exists in dependencies with the same version", version)
				return nil
			}
		}
	}

	packageJsonAdd := packagejson.PackageJSON{
		Dependencies: map[string]string{
			pkgName: version,
		},
	}
	err = pm.fetchToCache(packageJsonAdd, false)
	if err != nil {
		return err
	}

	err = pm.packageJsonParse.AddOrUpdateDependency(pkgName, version)
	if err != nil {
		return err
	}

	err = pm.packageJsonParse.UpdateLockFile(pm.packageLock, false)
	if err != nil {
		return err
	}

	pm.packageLock = pm.packageJsonParse.PackageLock

	return nil
}

func (pm *PackageManager) Remove(pkg string, removeFromPackageJson bool) error {

	pkgToRemove := pm.packageJsonParse.ResolveDependenciesToRemove(pkg)
	fmt.Println(pkgToRemove)

	err := pm.binLinker.UnlinkPackage(pkg)
	if err != nil {
		return err
	}

	err = pm.removePackagesFromNodeModules(pkgToRemove)
	if err != nil {
		return err
	}

	if removeFromPackageJson {
		err = pm.packageJsonParse.RemoveDependencies(pkg)
		if err != nil {
			return err
		}
	}

	err = pm.packageJsonParse.RemoveFromLockFile(pkg, pkgToRemove, true)
	if err != nil {
		return err
	}

	return nil
}

func (pm *PackageManager) fetchToCache(packageJson packagejson.PackageJSON, isProduction bool) error {
	queue := make([]QueueItem, 0)

	// Print workspace info if already discovered
	if pm.workspaceRegistry != nil {
		fmt.Printf("Discovered %d workspace packages\n", len(pm.workspaceRegistry.Packages))
		for _, ws := range pm.workspaceRegistry.Packages {
			fmt.Printf("  - %s@%s\n", ws.Name, ws.Version)
		}
	}

	for name, version := range packageJson.GetDependencies() {
		dep := packagejson.Dependency{Name: name, Version: version}

		// Check for GitHub dependency format: "github:user/repo#ref"
		if _, isGitHub := parseGitHubDependency(version); isGitHub {
			// Store the GitHub dependency info in the version field temporarily
			// The actual name will be determined after extracting the package
			dep.ActualName = name
			dep.Version = version // Keep the full GitHub spec
		} else if actualPkg, actualVersion, isAlias := parseAliasVersion(version); isAlias {
			// Check for npm alias format: "npm:actual-package@version"
			dep.ActualName = actualPkg
			dep.Version = actualVersion
		} else {
			dep.ActualName = name
		}

		queue = append(queue, QueueItem{
			Dep:        dep,
			ParentName: "package.json",
			IsDev:      false,
		})
	}

	if !isProduction {
		for name, version := range packageJson.GetDevDependencies() {
			dep := packagejson.Dependency{Name: name, Version: version}

			// Check for GitHub dependency format: "github:user/repo#ref"
			if _, isGitHub := parseGitHubDependency(version); isGitHub {
				dep.ActualName = name
				dep.Version = version
			} else if actualPkg, actualVersion, isAlias := parseAliasVersion(version); isAlias {
				// Check for npm alias format: "npm:actual-package@version"
				dep.ActualName = actualPkg
				dep.Version = actualVersion
			} else {
				dep.ActualName = name
			}

			queue = append(queue, QueueItem{
				Dep:        dep,
				ParentName: "package.json",
				IsDev:      true,
			})
		}
	}

	for name, version := range packageJson.GetOptionalDependencies() {
		dep := packagejson.Dependency{Name: name, Version: version}

		// Check for GitHub dependency format: "github:user/repo#ref"
		if _, isGitHub := parseGitHubDependency(version); isGitHub {
			dep.ActualName = name
			dep.Version = version
		} else if actualPkg, actualVersion, isAlias := parseAliasVersion(version); isAlias {
			// Check for npm alias format: "npm:actual-package@version"
			dep.ActualName = actualPkg
			dep.Version = actualVersion
		} else {
			dep.ActualName = name
		}

		queue = append(queue, QueueItem{
			Dep:        dep,
			ParentName: "package.json",
			IsDev:      false,
			IsOptional: true,
		})
	}

	packageLock := packagejson.PackageLock{}
	packageLock.Packages = make(map[string]packagejson.PackageItem)
	packageLock.Dependencies = make(map[string]string)
	packageLock.DevDependencies = make(map[string]string)
	packageLock.OptionalDependencies = make(map[string]string)
	packageLock.PeerDependencies = make(map[string]string)
	packagesVersion := make(map[string]QueueItem)

	var (
		wg             sync.WaitGroup
		mapMutex       sync.Mutex
		activeWorkers  int
		workerMutex    sync.Mutex
		processingPkgs = make(map[string]bool)
	)

	errChan := make(chan error, 1)
	done := make(chan struct{})

	workChan := make(chan QueueItem, len(queue))
	for _, item := range queue {
		if item.IsDev {
			packageLock.DevDependencies[item.Dep.Name] = item.Dep.Version
		} else {
			packageLock.Dependencies[item.Dep.Name] = item.Dep.Version
		}
		workChan <- item
	}

	for {
		workerMutex.Lock()
		workers := activeWorkers
		workerMutex.Unlock()

		if len(workChan) == 0 && workers == 0 {
			break
		}

		select {
		case item := <-workChan:
			workerMutex.Lock()
			activeWorkers++
			workerMutex.Unlock()

			wg.Add(1)

			go func(item QueueItem) {
				defer func() {
					wg.Done()
					workerMutex.Lock()
					activeWorkers--
					workerMutex.Unlock()
				}()

				if item.Dep.Name == "" {
					return
				}

				select {
				case <-done:
					return
				default:
				}

				// Use ActualName for downloading (handles aliases)
				actualName := item.Dep.ActualName
				if actualName == "" {
					actualName = item.Dep.Name
				}

				if pm.workspaceRegistry != nil {
					if wsPkg, isWorkspace := pm.workspaceRegistry.GetWorkspacePackage(actualName); isWorkspace {
						mapMutex.Lock()
						packageResolved := "node_modules/" + item.Dep.Name

						pckItem := packagejson.PackageItem{
							Name:     item.Dep.Name,
							Version:  wsPkg.Version,
							Resolved: "file:" + wsPkg.Path,
							Link:     true,
						}
						packageLock.Packages[packageResolved] = pckItem

						if packageLock.Workspaces == nil {
							packageLock.Workspaces = make(map[string]string)
						}
						packageLock.Workspaces[item.Dep.Name] = wsPkg.Version

						if item.ParentName == "package.json" {
							if item.IsDev {
								packageLock.DevDependencies[item.Dep.Name] = wsPkg.Version
							} else {
								packageLock.Dependencies[item.Dep.Name] = wsPkg.Version
							}
						}

						for depName, depVersion := range wsPkg.PackageJSON.GetDependencies() {
							pkgItem := packageLock.Packages[packageResolved]
							if pkgItem.Dependencies == nil {
								pkgItem.Dependencies = make(map[string]string)
							}
							pkgItem.Dependencies[depName] = depVersion
							packageLock.Packages[packageResolved] = pkgItem

							subDep := packagejson.Dependency{Name: depName, Version: depVersion, ActualName: depName}
							workChan <- QueueItem{
								Dep:        subDep,
								ParentName: packageResolved,
								IsDev:      item.IsDev,
							}
						}

						mapMutex.Unlock()

						return
					}
				}

				var version string
				var tarballURL string
				var resolvedURL string
				var currentEtag string
				var isGitHubDep bool
				var commitSHA string
				var npmPackage *manifestpkg.NPMPackage
				var err error

				// Check if this is a GitHub dependency
				if ghDep, isGitHub := parseGitHubDependency(item.Dep.Version); isGitHub {
					isGitHubDep = true

					// Resolve GitHub ref to commit SHA
					commitSHA, err = resolveGitHubRef(ghDep.Owner, ghDep.Repo, ghDep.Ref)
					if err != nil {
						if item.IsOptional || item.IsPeerOptional {
							fmt.Printf("Warning: Optional GitHub dependency %s failed to resolve: %v\n", item.Dep.Name, err)
							return
						}
						select {
						case errChan <- fmt.Errorf("failed to resolve GitHub dependency %s: %w", item.Dep.Name, err):
							close(done)
						default:
						}
						return
					}

					// Use full commit SHA as version (needed for lock file and sub-dependency resolution)
					version = commitSHA
					tarballURL = buildGitHubTarballURL(ghDep.Owner, ghDep.Repo, commitSHA)
					resolvedURL = buildGitHubResolvedURL(ghDep.Owner, ghDep.Repo, commitSHA)

					fmt.Printf("Resolved GitHub %s/%s#%s to commit %s\n", ghDep.Owner, ghDep.Repo, ghDep.Ref, commitSHA[:8])
				} else {
					// NPM package - download manifest and resolve version
					pm.downloadMu.Lock()
					pkgLock, exists := pm.downloadLocks[actualName]
					if !exists {
						pkgLock = &sync.Mutex{}
						pm.downloadLocks[actualName] = pkgLock
					}
					pm.downloadMu.Unlock()

					pkgLock.Lock()

					manifestPath := filepath.Join(pm.manifest.Path, actualName+".json")

					if _, err := os.Stat(manifestPath); err == nil {
						currentEtag = pm.Etag.Get(actualName)
					} else {
						etag := pm.Etag.Get(actualName)
						var downloadErr error
						currentEtag, _, downloadErr = pm.manifest.Download(actualName, etag)
						if downloadErr != nil {
							pkgLock.Unlock()
							if item.IsOptional || item.IsPeerOptional {
								fmt.Printf("Warning: Optional dependency %s failed to download manifest: %v\n", item.Dep.Name, downloadErr)
								return
							}
							select {
							case errChan <- downloadErr:
								close(done)
							default:
							}
							return
						}
					}

					npmPackage, err = pm.parseJsonManifest.parse(manifestPath)
					pkgLock.Unlock()

					if err != nil {
						if item.IsOptional || item.IsPeerOptional {
							fmt.Printf("Warning: Optional dependency %s failed to parse manifest: %v\n", item.Dep.Name, err)
							return
						}
						select {
						case errChan <- err:
							close(done)
						default:
						}
						return
					}

					version = pm.versionInfo.GetVersion(item.Dep.Version, npmPackage)
				}

				packageKey := actualName + "@" + version

				if version == "" {
					fmt.Println("Version not found for package:", item.Dep.Name, "with constraint:", item.Dep.Version)
				}

				// Check platform compatibility for optional dependencies
				if item.IsOptional {
					if versionData, ok := npmPackage.Versions[version]; ok {
						if !utils.IsCompatiblePlatform(versionData.OS, versionData.CPU) {
							fmt.Printf("Skipping optional dependency %s@%s (incompatible platform)\n", item.Dep.Name, version)
							// Still add to lock file but skip download
							mapMutex.Lock()
							packageResolved := "node_modules/" + item.Dep.Name
							pckItem := packagejson.PackageItem{
								Name:     item.Dep.Name,
								Version:  version,
								Resolved: "",
								Optional: true,
								OS:       versionData.OS,
								CPU:      versionData.CPU,
							}
							packageLock.Packages[packageResolved] = pckItem
							if item.ParentName == "package.json" {
								packageLock.OptionalDependencies[item.Dep.Name] = version
							}
							mapMutex.Unlock()
							return
						}
					}
				}

				var packageResolved string
				var processingKey string

				mapMutex.Lock()
				// Check if this exact package@version has already been processed or is being processed
				if processingPkgs[packageKey] {
					mapMutex.Unlock()
					return
				}
				if existingPkg, ok := packagesVersion[item.Dep.Name]; ok {
					// Check if the existing hoisted version satisfies the current constraint
					// existingPkg.Dep.Version is the resolved version (e.g., "0.1.0")
					// item.Dep.Version is the version constraint (e.g., "^0.3.0")
					existingSatisfiesConstraint := pm.versionInfo.SatisfiesConstraint(existingPkg.Dep.Version, item.Dep.Version)

					if !existingSatisfiesConstraint {
						// ParentName is now the full resolved path (e.g., "node_modules/wrap-ansi")
						// or "package.json" for top-level dependencies
						if item.ParentName == "package.json" {
							packageResolved = "node_modules/" + item.Dep.Name
							processingKey = packageKey
						} else {
							packageResolved = item.ParentName + "/node_modules/" + item.Dep.Name
							// Use a nested-specific key that includes the parent path
							// This allows the same version to be nested under multiple parents
							processingKey = packageResolved + "@" + version
						}

						// Check if this specific nested location has already been processed
						if processingPkgs[processingKey] {
							mapMutex.Unlock()
							return
						}

						fmt.Println("Package Repeated:", item.Dep.Name)
						fmt.Println("Resolved version:", version)
						fmt.Println("Existing version:", existingPkg.Dep.Version, "does not satisfy constraint:", item.Dep.Version)
						fmt.Println("Installing nested at:", packageResolved)

						processingPkgs[processingKey] = true
					} else {
						mapMutex.Unlock()
						return
					}
				} else {
					packageResolved = "node_modules/" + item.Dep.Name
					processingKey = packageKey
					packagesVersion[item.Dep.Name] = QueueItem{
						Dep:        packagejson.Dependency{Name: item.Dep.Name, Version: version},
						ParentName: item.ParentName,
					}

					processingPkgs[processingKey] = true
				}
				mapMutex.Unlock()

				configPackageVersion := filepath.Join(pm.packagesPath, actualName+"@"+version)

				// Build tarball URL if not already set (for npm packages)
				if !isGitHubDep {
					tarballName := actualName
					if strings.HasPrefix(actualName, "@") && strings.Contains(actualName, "/") {
						parts := strings.Split(actualName, "/")
						tarballName = parts[1]
					}
					tarballURL = fmt.Sprintf("%s%s/-/%s-%s.tgz", npmRegistryURL, actualName, tarballName, version)
					resolvedURL = tarballURL
				}

				uniqueTarballName := generateUniqueTarballName(actualName, version)

				// Lock based on package@version to prevent concurrent processing of the same package
				pm.downloadMu.Lock()
				packageLock_, exists := pm.downloadLocks[packageKey]
				if !exists {
					packageLock_ = &sync.Mutex{}
					pm.downloadLocks[packageKey] = packageLock_
				}
				pm.downloadMu.Unlock()

				packageLock_.Lock()
				defer packageLock_.Unlock()

				// Check again if folder exists after acquiring lock
				if !utils.FolderExists(configPackageVersion) {
					if tarballURL == "" || version == "" {
						fmt.Printf("Skipping download for %s - invalid URL or empty version\n", item.Dep.Name)
						return
					}

					tarballPath := filepath.Join(pm.tarball.TarballPath, uniqueTarballName)

					// Validate tarball (checks existence and integrity)
					shouldDownloadTarball := true
					if utils.ValidateTarball(tarballPath) {
						shouldDownloadTarball = false
					} else {
						os.Remove(tarballPath)
					}

					if shouldDownloadTarball {
						err = pm.tarball.DownloadAs(tarballURL, uniqueTarballName)
						if err != nil {
							if item.IsOptional || item.IsPeerOptional {
								fmt.Printf("Warning: Optional dependency %s failed to download tarball: %v\n", item.Dep.Name, err)
								return
							}
							select {
							case errChan <- err:
								close(done)
							default:
							}
							return
						}
					}

					// Extract tarball (extractor strips first dir component for both npm and GitHub)
					err = pm.extractor.Extract(tarballPath, configPackageVersion)
					if err != nil {
						if item.IsOptional || item.IsPeerOptional {
							fmt.Printf("Warning: Optional dependency %s failed to extract: %v\n", item.Dep.Name, err)
							return
						}
						select {
						case errChan <- err:
							close(done)
						default:
						}
						return
					}
				}

				mapMutex.Lock()
				pckItem := packagejson.PackageItem{
					Name:     item.Dep.Name,
					Version:  version,
					Resolved: resolvedURL,
					Etag:     currentEtag,
					Optional: item.IsOptional,
				}
				// Add OS and CPU fields if available (npm packages only)
				if !isGitHubDep {
					if versionData, ok := npmPackage.Versions[version]; ok {
						if len(versionData.OS) > 0 {
							pckItem.OS = versionData.OS
						}
						if len(versionData.CPU) > 0 {
							pckItem.CPU = versionData.CPU
						}
					}
				}
				packageLock.Packages[packageResolved] = pckItem

				// Add to OptionalDependencies in lock if this is a top-level optional dependency
				if item.IsOptional && item.ParentName == "package.json" {
					packageLock.OptionalDependencies[item.Dep.Name] = version
				}

				// Track peer dependencies that were auto-installed
				if item.IsPeer {
					packageLock.PeerDependencies[item.Dep.Name] = version
				}
				mapMutex.Unlock()

				packageDir := filepath.Join(pm.packagesPath, actualName+"@"+version)
				packageJsonPath := filepath.Join(packageDir, "package.json")

				// Validate package.json exists and is not corrupted (non-zero size)
				fileInfo, statErr := os.Stat(packageJsonPath)
				if statErr != nil || fileInfo.Size() == 0 {
					// Package.json is missing or empty - remove corrupted package directory
					err = os.RemoveAll(packageDir)
					if err != nil {
						errChan <- fmt.Errorf("failed to remove corrupted package %s: %w", actualName, err)
						close(done)
						return
					}

					// Re-extract from tarball
					uniqueTarballName := generateUniqueTarballName(actualName, version)
					tarballPath := filepath.Join(pm.tarball.TarballPath, uniqueTarballName)

					if extractErr := pm.extractor.Extract(tarballPath, packageDir); extractErr != nil {
						select {
						case errChan <- fmt.Errorf("failed to re-extract corrupted package %s: %w", actualName, extractErr):
							close(done)
						default:
						}
						return
					}
				}

				data, err := pm.packageJsonParse.Parse(packageJsonPath)
				if err != nil {
					select {
					case errChan <- err:
						close(done)
					default:
					}
					return
				}

				mapMutex.Lock()
				currentPkgName := extractPackageName(packageResolved)
				for name, depVersion := range data.GetDependencies() {
					pkgItem := packageLock.Packages[packageResolved]
					if pkgItem.Dependencies == nil {
						pkgItem.Dependencies = make(map[string]string)
					}
					pkgItem.Dependencies[name] = depVersion
					packageLock.Packages[packageResolved] = pkgItem

					// Skip if package is trying to install itself as nested dependency
					if name == currentPkgName {
						continue
					}

					// Check if sub-dependency is also an alias
					subDep := packagejson.Dependency{Name: name, Version: depVersion}
					if actualPkg, actualVersion, isAlias := parseAliasVersion(depVersion); isAlias {
						subDep.ActualName = actualPkg
						subDep.Version = actualVersion
					} else {
						subDep.ActualName = name
					}

					workChan <- QueueItem{
						Dep:        subDep,
						ParentName: packageResolved,
						IsDev:      item.IsDev,
					}
				}

				// Process optional dependencies from sub-packages
				for name, depVersion := range data.GetOptionalDependencies() {
					pkgItem := packageLock.Packages[packageResolved]
					if pkgItem.OptionalDependencies == nil {
						pkgItem.OptionalDependencies = make(map[string]string)
					}
					pkgItem.OptionalDependencies[name] = depVersion
					packageLock.Packages[packageResolved] = pkgItem

					// Skip if package is trying to install itself as nested dependency
					if name == currentPkgName {
						continue
					}

					// Check if sub-dependency is also an alias
					subDep := packagejson.Dependency{Name: name, Version: depVersion}
					if actualPkg, actualVersion, isAlias := parseAliasVersion(depVersion); isAlias {
						subDep.ActualName = actualPkg
						subDep.Version = actualVersion
					} else {
						subDep.ActualName = name
					}

					workChan <- QueueItem{
						Dep:        subDep,
						ParentName: packageResolved,
						IsDev:      false,
						IsOptional: true,
					}
				}

				// Process peer dependencies from sub-packages (auto-install per npm 7+ behavior)
				for name, depVersion := range data.GetPeerDependencies() {
					pkgItem := packageLock.Packages[packageResolved]
					if pkgItem.PeerDependencies == nil {
						pkgItem.PeerDependencies = make(map[string]string)
					}
					pkgItem.PeerDependencies[name] = depVersion
					packageLock.Packages[packageResolved] = pkgItem

					// Skip if package is trying to install itself as nested dependency
					if name == currentPkgName {
						continue
					}

					// Check if this peer dependency is optional
					isPeerOptional := false
					if data.PeerDependenciesMeta != nil {
						if meta, exists := data.PeerDependenciesMeta[name]; exists {
							isPeerOptional = meta.Optional
						}
					}

					// Check if sub-dependency is also an alias
					subDep := packagejson.Dependency{Name: name, Version: depVersion}
					if actualPkg, actualVersion, isAlias := parseAliasVersion(depVersion); isAlias {
						subDep.ActualName = actualPkg
						subDep.Version = actualVersion
					} else {
						subDep.ActualName = name
					}

					workChan <- QueueItem{
						Dep:            subDep,
						ParentName:     packageResolved,
						IsDev:          false,
						IsOptional:     false,
						IsPeer:         true,
						IsPeerOptional: isPeerOptional,
					}
				}
				mapMutex.Unlock()
			}(item)
		default:
			workerMutex.Lock()
			if activeWorkers == 0 {
				workerMutex.Unlock()
				break
			}
			workerMutex.Unlock()
		}
	}

	wg.Wait()
	close(errChan)

	if err := <-errChan; err != nil {
		return err
	}
	pm.packageLock = &packageLock

	// Validate peer dependencies and print warnings
	warnings := pm.validatePeerDependencies(&packageLock)
	if len(warnings) > 0 {
		fmt.Fprintln(os.Stderr, "\n⚠️  Peer dependency warnings:")
		for _, warning := range warnings {
			fmt.Fprintln(os.Stderr, "  ", warning)
		}
		fmt.Fprintln(os.Stderr)
	}

	return nil
}

// validatePeerDependencies checks if peer dependency requirements are satisfied
func (pm *PackageManager) validatePeerDependencies(packageLock *packagejson.PackageLock) []string {
	warnings := []string{}

	// Iterate through all packages and check their peer dependencies
	for pkgPath, pkgItem := range packageLock.Packages {
		if len(pkgItem.PeerDependencies) == 0 {
			continue
		}

		for peerName, peerVersionConstraint := range pkgItem.PeerDependencies {
			// Check if peer dependency is installed
			installedVersion := ""
			peerPath := "node_modules/" + peerName

			if peerPkg, exists := packageLock.Packages[peerPath]; exists {
				installedVersion = peerPkg.Version
			}

			if installedVersion == "" {
				warnings = append(warnings, fmt.Sprintf(
					"%s requires peer %s@%s but it is not installed",
					pkgPath, peerName, peerVersionConstraint,
				))
				continue
			}

			// Check if installed version satisfies the peer requirement
			npmPackage := &manifestpkg.NPMPackage{
				Versions: map[string]manifestpkg.Version{
					installedVersion: {Version: installedVersion},
				},
				DistTags: manifestpkg.DistTags{Latest: installedVersion},
			}

			resolvedVersion := pm.versionInfo.GetVersion(peerVersionConstraint, npmPackage)
			if resolvedVersion != installedVersion {
				warnings = append(warnings, fmt.Sprintf(
					"%s requires peer %s@%s but version %s is installed",
					pkgPath, peerName, peerVersionConstraint, installedVersion,
				))
			}
		}
	}

	return warnings
}

func (pm *PackageManager) addBinToPath() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	bashrcPath := filepath.Join(homeDir, ".bashrc")
	exportLine := fmt.Sprintf("export PATH=\"%s:$PATH\"", pm.config.GlobalBinDir)

	content, err := os.ReadFile(bashrcPath)
	if err != nil {
		if os.IsNotExist(err) {
			content = []byte{}
		} else {
			return fmt.Errorf("failed to read .bashrc: %w", err)
		}
	}

	if strings.Contains(string(content), exportLine) {
		return nil
	}

	newContent := string(content)
	if len(content) > 0 && !strings.HasSuffix(newContent, "\n") {
		newContent += "\n"
	}
	newContent += fmt.Sprintf("\n# Added by go-npm\n%s\n", exportLine)

	if err := os.WriteFile(bashrcPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write .bashrc: %w", err)
	}

	return nil
}

func (pm *PackageManager) InstallGlobal(pkgName, version string) error {
	if !pm.isGlobal {
		return fmt.Errorf("package manager is not in global mode")
	}

	fmt.Printf("Installing %s globally...\n", pkgName)

	if version == "" {
		version = "latest"
	}

	packageJsonToInstall := packagejson.PackageJSON{
		Dependencies: map[string]string{
			pkgName: version,
		},
	}

	if err := pm.fetchToCache(packageJsonToInstall, false); err != nil {
		return fmt.Errorf("failed to fetch package to cache: %w", err)
	}

	if err := pm.InstallFromCache(); err != nil {
		return fmt.Errorf("failed to install package: %w", err)
	}

	if _, err := os.Stat(pm.config.GlobalLockFile); err == nil {
		if err := pm.packageJsonParse.UpdateLockFile(pm.packageLock, true); err != nil {
			return fmt.Errorf("failed to update global lock file: %w", err)
		}
	} else {
		if err := pm.packageJsonParse.CreateLockFile(pm.packageLock, true); err != nil {
			return fmt.Errorf("failed to create global lock file: %w", err)
		}
	}
	// Add bin directory to PATH in .bashrc
	if err := pm.addBinToPath(); err != nil {
		fmt.Printf("Warning: Failed to add bin directory to PATH: %v\n", err)
		fmt.Printf("Please manually add to PATH: export PATH=\"%s:$PATH\"\n", pm.config.GlobalBinDir)
	} else {
		fmt.Printf("\n✓ Successfully installed %s globally\n", pkgName)
		fmt.Printf("✓ Added bin directory to PATH in ~/.bashrc\n")
		fmt.Printf("  Run 'source ~/.bashrc' to apply changes in current terminal\n")
		return nil
	}

	fmt.Printf("\n✓ Successfully installed %s globally\n", pkgName)
	fmt.Printf("Binaries available in: %s\n", pm.config.GlobalBinDir)

	return nil
}
