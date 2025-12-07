# TUTORIAL  

![demo](https://raw.githubusercontent.com/ernesto27/go-npm/main/tutorial/go-npm.png)



# Intro

Hi, this tutorial is about creating an npm package manager clone using Go.

We start from scratch with a basic implementation in which we can run an install command like "go run . i" and start a simple express server.
This first version will be a starting point and functional for simple projects but does not have all the features that npm has (lock file, cache optimizations, global installations, etc), but besides that, it is a good starting point to understand how this works and to get a first glance of system programming.

## Demo

![demo](https://raw.githubusercontent.com/ernesto27/go-npm/main/tutorial/demo.gif)

Download code from tutorial here

https://drive.google.com/file/d/1r3DHq1a6poGcPl8BXwAyXIW8_0YoyNGT/view?usp=sharing



Before starting, it is necessary to know the current status of different JS/Node package managers.


### npm (Node Package Manager)
The default package manager bundled with Node.js. It manages dependencies in a flat node_modules structure. It is the industry standard for zero-configuration setups.

https://nodejs.org/en/learn/getting-started/an-introduction-to-the-npm-package-manager


### yarn (Yet Another Resource Negotiator)
Developed by Meta to improve upon npm's early performance. It utilizes parallel downloads for faster installation and is widely favored for its "Workspaces" feature, which simplifies managing multiple projects (monorepos).

https://yarnpkg.com/


### pnpm (Performant npm)
Designed for maximum disk efficiency. Instead of copying files for every project, it saves a single copy in a global store and links to them via symlinks. This drastically reduces disk usage and speeds up installation.

https://pnpm.io/es/


### bun
An ultra-fast, all-in-one JavaScript runtime and package manager written in Zig. It aims to replace the entire modern toolchain by acting as your runtime, bundler, test runner, and package manager simultaneously.

https://bun.com/



Although all of these projects use different languages and were created in different contexts and times, all share the same final goal, that is to download and install packages in a node_modules folder to be used in a front or backend project.


We will create a base and solid design structure to build upon it in future versions, also with testing to ensure that our code is working as expected. 


# Table of content
- How npm install works
- Setup project
- Config component
- Packagejson component
- Manifest component
- Version component
- Tarball Component
- Extractor component
- Manager component


# How npm install works 

Before starting the project we need to understand how the command npm install works in detail,  what components are involved and how they interact with each other.

This is a base diagram, we will start simple and not think at moment about cache and performance optimizations.

![npm install diagram](https://raw.githubusercontent.com/ernesto27/go-npm/main/tutorial/diagram.jpg)


1. **npm install**:
   call the command to install packages.

2. **Parse package.json file**:
   npm reads your project’s package.json to know which packages need to install.

3. **Download manifest**:
   npm go to the registry url of the packages and downloads the manifest file that contain all the versions and metadata of the package.

4. **Download tarball**:
   npm downloads the actual package file (a .tgz archive),  obtained from the manifest file.

5. **Extract tarball**:
   npm unpacks the .tgz file into user machine.

6. **Copy package to node_modules**:
   npm moves the unpacked package into your project's node_modules folder so it can be used.

This is a simple overview of the process,  like said before npm and other packages managers have optimizations and tricks even for specific OS to make things a lot faster.


# Setup project

We are going to use golang version 1.25 as language, so install  from here 

https://go.dev/doc/install

We also need nodejs in order to test the express backend.

https://nodejs.org/en/download


Create a new folder and initialize go module 

```bash
mkdir go-npm
cd go-npm 
go mod init go-npm 
```

We will use some external dependencies that help us with CLI commands and testing, so let's install them now.

```bash
go get github.com/spf13/cobra@v1.10.1
go get github.com/stretchr/testify@v1.11.1       
```

*main.go*
```go
package main

import (
	"go-npm/cmd"
)

func main() {
	cmd.Execute()
}
```

in main function we initialize the cmd package (create next) that will handle the CLI commands using cobra library.



Create a cmd folder and put a root.go file inside it 
```sh 
mkdir cmd
cd cmd
touch root.go
```

*cmd/root.go*

```go
package cmd

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "go-npm",
	Short: "A Go implementation of npm package manager",
	Long:  `go-npm is a Go implementation of an npm package manager that downloads and installs npm packages and their dependencies.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
}
```
Here we created the root command for our NPM clone and define some text to show in terminal, 
Execute is the function that is called in main.go and start the cobra init,  if error happens show message and exit the app.

init is a special function in go that is called when the package is used, here we disable the default completion command that cobra add by default.


Create a install file 

```sh 
cd cmd
touch install.go
```

*cmd/install.go*
```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install",
	Aliases: []string{"i"},
	Short:   "Install packages",
	Long:    `Install packages from package.json.`,
	RunE:    runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	fmt.Println("Starting installation process...")
	return nil
}
```
in this file we create a cobra command for install ,  this have and alias i that allow to call the command with "install" or "i".

RunE is the definition of the function to execute when this command is called,  for now just print a message,

in init function we add this command to the root command created previously in file root.go



We can test code with this command

```bash
go run . i
Starting installation process..
```

Some nice features that have cobra by default is the use of command -h (help) that show the availabe commands and descriptions


```bash
go run . -h
Usage:
  go-npm [command]

Available Commands:
  help        Help about any command
  install     Install packages

Flags:
  -h, --help   help for go-npm
```


# Config component

Create a config folder and a config.go file inside it

```sh
mkdir config
cd config
touch config.go
```

*config/config.go*

```go
package config

import (
	"go-npm/utils"
	"os"
	"path/filepath"
)

type Config struct {
	NpmRegistryURL string

	// Base directories
	BaseDir     string
	ManifestDir string
	TarballDir  string
	PackagesDir string

	// Local installation paths
	LocalNodeModules string
	LocalBinDir      string
}

func New() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Join(homeDir, ".config", "go-npm")

	if err := utils.CreateDir(baseDir); err != nil {
		return nil, err
	}

	manifestPath := filepath.Join(baseDir, "manifest")
	if err := utils.CreateDir(manifestPath); err != nil {
		return nil, err
	}

	tarballPath := filepath.Join(baseDir, "tarball")
	if err := utils.CreateDir(tarballPath); err != nil {
		return nil, err
	}

	packagesPath := filepath.Join(baseDir, "packages")
	if err := utils.CreateDir(packagesPath); err != nil {
		return nil, err
	}

	return &Config{
		NpmRegistryURL: "https://registry.npmjs.org/",
		BaseDir:        baseDir,
		ManifestDir:    filepath.Join(baseDir, "manifest"),
		TarballDir:     filepath.Join(baseDir, "tarball"),
		PackagesDir:    filepath.Join(baseDir, "packages"),

		LocalNodeModules: "./node_modules",
	}, nil
}

```

We define a `Config` struct to hold all the configuration paths and URLs needed for the app.

- **Config Struct**:
    - `NpmRegistryURL`: The base URL for the npm registry (e.g., `https://registry.npmjs.org/`).
    - `BaseDir`: The root directory for our application's configuration and cache (`~/.config/go-npm`).
    - `ManifestDir`: Directory to store downloaded package manifests.
    - `TarballDir`: Directory to store downloaded `.tgz` files.
    - `PackagesDir`: Directory for extracted packages (if we were using a global store).
    - `LocalNodeModules`: The path to the local `node_modules` directory in the user's project.

- **New()**:
    - **Returns**: A pointer to a `Config` instance and an error if initialization fails.
    - **Logic**:
        - Gets the user's home directory.
        - Constructs the base configuration path (`~/.config/go-npm`).
        - Creates all necessary subdirectories (`manifest`, `tarball`, `packages`) using `utils.CreateDir`.
        - Returns the populated `Config` struct.

The structure created on your disk will look like this:

```
~/.config/go-npm
       │
       ├── manifest
       │
       └── packages
	   │
       └── tarball

```
> In this tutorial we are not going to use all folders created in .config folder, but is a good idea to have this ready for next tutorial.


# Packagejson component

First step after run the install command is to parse the package.json, this is necessary to obtaint the dependencies that need to be installed (for now we ignore devDependencies and peerDependencies, etc).

Let create a package.json file with this content in root of project.

*package.json*
```json
{
  "name": "go-npm-example",
  "version": "1.0.0",
  "description": "A simple Express server example",
  "main": "index.js",
  "scripts": {
    "start": "node index.js"
  },
  "dependencies": {
    "express": "^5.0.1"
  }
}
```

this is basic package.json for a node server that define a express dependency and will be use for test app.


For test server add this in root project

*index.js*

```js
const express = require('express');

const app = express();
const PORT = process.env.PORT || 3000;

app.use(express.json());

app.get('/', (req, res) => {
  res.json({
    message: 'Hello World!',
  });
});

app.get('/health', (req, res) => {
  res.json({ status: 'OK', timestamp: new Date().toISOString() });
});

app.listen(PORT, () => {
  console.log(`Server is running on port ${PORT}`);
});
```

This is nodejs code that setup and initialize a express server and two endpoints  / and /health 


Now if we run command `node index.js` we will get this error

![alt text](https://raw.githubusercontent.com/ernesto27/go-npm/main/tutorial/image.png)

This is expected because node_modeules folder does not exists 


Next step is to read the package.json file and parse the dependencies property, for that we will create a new packagejson package that will handle that.

Run this
```sh
mkdir packagejson
cd packagejson
touch packagejson.go
```

*packagejson.go*

```go
type PackageJSON struct {
	Name            string            `json:"name"`
	Description     string            `json:"description"`
	Version         any               `json:"version"`
	Author          any               `json:"author"`
	Contributors    any               `json:"contributors"`
	License         any               `json:"license"`
	Repository      any               `json:"repository"`
	Homepage        any               `json:"homepage"`
	Funding         any               `json:"funding"`
	Keywords        any               `json:"keywords"`
	Dependencies    map[string]string `json:"dependencies"`
	Engines         any               `json:"engines"`
	Files           any               `json:"files"`
	Scripts         map[string]string `json:"scripts"`
	Main            any               `json:"main"`
	Bin             any               `json:"bin"`
	Types           string            `json:"types"`
	Exports         any               `json:"exports"`
	Private         bool              `json:"private"`
	Workspaces      any               `json:"workspaces"`
}

type PackageJSONParser struct {
	Config          *config.Config
	PackageJSON     *PackageJSON
	FilePath        string
	OriginalContent []byte
}

func NewPackageJSONParser(cfg *config.Config) *PackageJSONParser {
	return &PackageJSONParser{
		Config: cfg,
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

	p.PackageJSON = &packageJSON
	p.FilePath = filePath
	p.OriginalContent = fileContent

	return &packageJSON, nil
}
```

We define the `PackageJSON` struct to map the standard `package.json` fields. While we include many fields for completeness, our primary focus is on `Dependencies`.

- **NewPackageJSONParser(cfg *config.Config)**:
    - **Parameters**: `cfg` - A pointer to the `Config` struct.
    - **Returns**: A pointer to a new `PackageJSONParser` instance.
    - **Logic**: Initializes the parser with the provided configuration.

- **Parse(filePath string)**:
    - **Parameters**: `filePath` - The path to the `package.json` file to parse.
    - **Returns**: A pointer to the parsed `PackageJSON` struct and an error if parsing fails.
    - **Logic**:
        - Reads the file content using `os.ReadFile`.
        - Unmarshals the JSON content into the `PackageJSON` struct.
        - Stores the parsed data and original content in the parser instance for potential future use (e.g., writing back changes).


### Testing the parser
One of the most important things in this kind of project is create tests at beginnig of the project, this is critical when add new features or refactor to check that 
all is working as expected.


run this
```sh
cd packagejson
touch packagejson_test.go
```

*packagejson/packagejson_test.go*

```go
package packagejson

import (
	"encoding/json"
	"go-npm/config"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPackageJSONParser_Parse(t *testing.T) {
	testCases := []struct {
		name        string
		setupFile   func(t *testing.T) string
		expectError bool
		validate    func(t *testing.T, result *PackageJSON)
	}{
		{
			name: "Valid basic package.json",
			setupFile: func(t *testing.T) string {
				tmpDir := t.TempDir()
				tmpFile := filepath.Join(tmpDir, "package.json")

				packageData := PackageJSON{
					Name:        "test-package",
					Description: "A test package",
					Version:     "1.2.3",
					Author:      "Test Author",
					License:     "MIT",
					Homepage:    "https://example.com",
					Keywords:    []string{"test", "example"},
					Dependencies: map[string]string{
						"express": "^4.18.0",
						"lodash":  "^4.17.21",
					},
					Scripts: map[string]string{
						"start": "node index.js",
						"test":  "jest",
					},
					Main:    "index.js",
					Types:   "index.d.ts",
					Private: false,
				}

				data, _ := json.MarshalIndent(packageData, "", "  ")
				os.WriteFile(tmpFile, data, 0644)
				return tmpDir
			},
			expectError: false,
			validate: func(t *testing.T, result *PackageJSON) {
				assert.Equal(t, "test-package", result.Name)
				assert.Equal(t, "1.2.3", result.Version)
				assert.Equal(t, "A test package", result.Description)
				assert.Equal(t, "MIT", result.License)
				assert.Equal(t, map[string]string{
					"express": "^4.18.0",
					"lodash":  "^4.17.21",
				}, result.Dependencies)
				assert.Equal(t, map[string]string{
					"start": "node index.js",
					"test":  "jest",
				}, result.Scripts)
			},
		},
		{
			name: "Non-existent file",
			setupFile: func(t *testing.T) string {
				return t.TempDir()
			},
			expectError: true,
			validate: func(t *testing.T, result *PackageJSON) {
				assert.Nil(t, result)
			},
		},
		{
			name: "Invalid JSON",
			setupFile: func(t *testing.T) string {
				tmpDir := t.TempDir()
				tmpFile := filepath.Join(tmpDir, "package.json")

				invalidJSON := []byte(`{
					"name": "test",
					"version": "1.0.0",
					"invalid":
				}`)

				os.WriteFile(tmpFile, invalidJSON, 0644)
				return tmpDir
			},
			expectError: true,
			validate: func(t *testing.T, result *PackageJSON) {
				assert.Nil(t, result)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := tc.setupFile(t)

			// Save current directory
			originalDir, err := os.Getwd()
			assert.NoError(t, err)
			defer os.Chdir(originalDir)

			// Change to temp directory
			err = os.Chdir(tmpDir)
			assert.NoError(t, err)

			cfg, err := config.New()
			assert.NoError(t, err)

			parser := NewPackageJSONParser(cfg)
			result, err := parser.Parse("package.json")

			if tc.expectError {
				assert.Error(t, err, "Expected an error")
			} else {
				assert.NoError(t, err, "Expected no error")
			}

			tc.validate(t, result)
		})
	}
}

```

We use TableDrivenTest pattern to define multiple test cases for the Parse method,
we have a setupFile file function that create the context and files for each test case,  and a validate function to check expected results.

- Valid basic package.json: We create a valid file. We expect no error, and we check that the parser correctly read the name, version, and dependencies.
- Non-existent file: We don't create a file at all. We expect an error.
- Invalid JSON: We create a file with broken JSON (like a missing bracket). We expect an error. 

After we loop through each test case,  change to the /tmp dir created for the test case, this is important for not create files in our current working dir, 
and run tests in and isolated folder that does not affect our current directory,  also we use the testify library to make assertions easier.

Run test  

```bash
go test ./...
?       go-npm  [no test files]
?       go-npm/cmd      [no test files]
?       go-npm/config   [no test files]
ok      go-npm/packagejson      0.003s
```

# Manifest component

After parsing the package.json and get the dependencies to install, we need to obtain the manifest file from npm registry, 
this is necessary to download the correspoding tarball associated to the package.

For example if we have this express dependency in package.json

```json
"dependencies": {
    "express": "^5.0.1"
}
```
And go to this url https://registry.npmjs.org/express and obtain the manifest file.

This return a json file with a lot of information about the package, versions, dist-tags, time, maintainers, etc.
we will focus in versions for now, for example this is a object of all available entries, wit this structure

```json
"versions": {
    "5.0.1": {
      "name": "express",
      "version": "5.0.1",
      "dist": { 
        "tarball": "https://registry.npmjs.org/express/-/express-5.0.1.tgz",
        "shasum": "somehashvalue"
      }
    },
   // more items
}
```

So for that we will create a new manifest package that will handle the manifest fetching and parsing.


```bash
mkdir manifest
cd manifest
touch manifest.go
```

*manifest/manifest.go*

```go
package manifest

import (
	"go-npm/utils"
	"path/filepath"
)

type Manifest struct {
	npmResgistryURL string
	Path            string
}


type NPMPackage struct {
	ID       string             `json:"_id"`
	Rev      string             `json:"_rev"`
	Name     string             `json:"name"`
	DistTags DistTags           `json:"dist-tags"`
	Versions map[string]Version `json:"versions"`
	Time     map[string]string  `json:"time"`
	Bugs     any                `json:"bugs"`
	License  any                `json:"license"`
	Homepage any                `json:"homepage"`
	Keywords any                `json:"keywords"`

	Repository     any             `json:"repository"`
	Description    string          `json:"description"`
	Contributors   any             `json:"contributors"`
	Maintainers    any             `json:"maintainers"`
	Readme         string          `json:"readme"`
	ReadmeFilename string          `json:"readmeFilename"`
	Users          map[string]bool `json:"users"`
}

type DistTags struct {
	Latest string `json:"latest"`
	Next   string `json:"next"`
}

type Version struct {
	Name                   string                 `json:"name"`
	Version                string                 `json:"version"`
	Author                 any                    `json:"author"`
	License                any                    `json:"license"`
	ID                     string                 `json:"_id"`
	Maintainers            any                    `json:"maintainers"`
	Homepage               any                    `json:"homepage"`
	Bugs                   any                    `json:"bugs"`
	Dist                   Dist                   `json:"dist"`
	From                   string                 `json:"_from"`
	Shasum                 string                 `json:"_shasum"`
	Engines                any                    `json:"engines"`
	GitHead                string                 `json:"gitHead"`
	Scripts                any                    `json:"scripts"`
	NPMUser                NPMUser                `json:"_npmUser"`
	Repository             any                    `json:"repository"`
	NPMVersion             string                 `json:"_npmVersion"`
	Description            string                 `json:"description"`
	Directories            map[string]interface{} `json:"directories"`
	NodeVersion            string                 `json:"_nodeVersion"`
	Dependencies           map[string]string      `json:"dependencies"`
	DevDependencies        map[string]string      `json:"devDependencies"`
	OptionalDependencies   map[string]string      `json:"optionalDependencies"`
	PeerDependencies       map[string]string      `json:"peerDependencies"`
	PeerDependenciesMeta   map[string]PeerMeta    `json:"peerDependenciesMeta"`
	OS                     []string               `json:"os"`
	CPU                    []string               `json:"cpu"`
	HasShrinkwrap          bool                   `json:"_hasShrinkwrap"`
	Keywords               any                    `json:"keywords"`
	Contributors           any                    `json:"contributors"`
	Files                  any                    `json:"files"`
	NPMOperationalInternal NPMOperationalInternal `json:"_npmOperationalInternal"`
	NPMSignature           string                 `json:"npm-signature"`
}

type PeerMeta struct {
	Optional bool `json:"optional"`
}

type Author struct {
	URL   string `json:"url"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Maintainer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Contributor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	URL   string `json:"url"`
}

type Bugs struct {
	URL string `json:"url"`
}

type Dist struct {
	Shasum       string      `json:"shasum"`
	Tarball      string      `json:"tarball"`
	Integrity    string      `json:"integrity"`
	Signatures   []Signature `json:"signatures"`
	FileCount    int         `json:"fileCount"`
	UnpackedSize int         `json:"unpackedSize"`
}

type Signature struct {
	Sig   string `json:"sig"`
	KeyID string `json:"keyid"`
}

type NPMUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Repository struct {
	URL  string `json:"url"`
	Type string `json:"type"`
}

type NPMOperationalInternal struct {
	Tmp  string `json:"tmp"`
	Host string `json:"host"`
}

func NewManifest(manifestPath string, npmRegistryURL string) (*Manifest, error) {
	return &Manifest{
		Path:            manifestPath,
		npmResgistryURL: npmRegistryURL,
	}, nil
}


func (m *Manifest) Download(pkg string) (NPMPackage, error) {
	url := m.npmResgistryURL + pkg
	filename := filepath.Join(m.Path, pkg+".json")
	npmPackage := NPMPackage{}

	statusCode, err := utils.DownloadFile(url, filename)
	if err != nil {
		return npmPackage, fmt.Errorf("failed to download manifest for package %s: %w", pkg, err)
	}

	if statusCode != http.StatusOK {
		return npmPackage, fmt.Errorf("failed to download manifest for package %s: status code %d", pkg, statusCode)
	}

	file, err := os.Open(filename)
	if err != nil {
		return npmPackage, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&npmPackage); err != nil {
		return npmPackage, fmt.Errorf("failed to parse JSON from file %s: %w", filename, err)
	}

	return npmPackage, err
}


```

- **NewManifest(manifestPath string, npmRegistryURL string)**:
    - **Parameters**:
        - `manifestPath`: The local directory path where manifest files will be cached.
        - `npmRegistryURL`: The base URL of the NPM registry (e.g., `https://registry.npmjs.org/`).
    - **Returns**: A pointer to an initialized `Manifest` struct or an error.

- **Download(pkg string)**:
    - **Parameters**:
        - `pkg`: The name of the package to download (e.g., `express`).
    - **Returns**: An `NPMPackage` struct containing the parsed manifest data, or an error if the download fails.
    - **Logic**:
        1.  **Construct URL**: Appends the package name to the registry URL (e.g., `https://registry.npmjs.org/express`).
        2.  **Download File**: Calls `utils.DownloadFile` to fetch the JSON manifest and save it to the local cache directory.
        3.  **Validate Response**: Checks if the HTTP status code is 200 OK.
        4.  **Parse JSON**: Opens the downloaded file and unmarshals the JSON content into the `NPMPackage` struct.




Create utils file

```sh
mkdir utils
cd utils
touch utils.go
```

*utils/utils.go*

```go
package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func DownloadFile(url, filename string) (int, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		return resp.StatusCode, nil
	}

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("HTTP error: %s, %d %s", url, resp.StatusCode, resp.Status)
	}

	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return resp.StatusCode, fmt.Errorf("failed to create directory structure: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("failed to create file: %w", err)
	}

	_, err = io.Copy(file, resp.Body)
	file.Close()

	if err != nil {
		os.Remove(filename)
		return resp.StatusCode, fmt.Errorf("failed to write file: %w", err)
	}

	return resp.StatusCode, nil
}

func CreateDir(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		if err := os.Mkdir(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}
		fmt.Printf("Created directory: %s\n", dirPath)
	}
	return nil
}
```

- **DownloadFile(url, filename string)**:
    - **Parameters**:
        - `url`: The URL of the file to download.
        - `filename`: The local path where the file should be saved.
    - **Returns**: The HTTP status code and an error if the operation fails.
    - **Logic**:
        1.  **Create Request**: Creates a new HTTP GET request.
        2.  **Execute Request**: Sends the request using an HTTP client.
        3.  **Check Status**:
            - Returns `StatusNotModified` if applicable.
            - Returns an error if the status code is not 200 OK.
        4.  **Create Directory**: Ensures the parent directory exists using `os.MkdirAll`.
        5.  **Create File**: Creates the destination file.
        6.  **Write Content**: Streams the response body to the file using `io.Copy`.
        7.  **Cleanup**: Removes the file if writing fails.

- **CreateDir(dirPath string)**:
    - **Parameters**:
        - `dirPath`: The path of the directory to create.
    - **Returns**: An error if creation fails.
    - **Logic**:
        - Checks if the directory already exists.
        - If not, creates it with `0755` permissions (rwxr-xr-x).

We have the base code to check if we can download a manifest file from npm,  to do that update the install command

*cmd/install.go*
```go
func runInstall(cmd *cobra.Command, args []string) error {
	fmt.Println("Starting installation process...")

	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	manifest, err := manifest.NewManifest(cfg.ManifestDir, cfg.NpmRegistryURL)
	if err != nil {
		panic(err)
	}

	npmPackage, err := manifest.Download("express")
	if err != nil {
		panic(err)
	}

	fmt.Println(npmPackage.Name)

	return nil
}

```

here we call config.New method to initialize config properties and folders needed for this project,  then call manifest Download method to save manifest, for now we hardcode "express" package to test.


if everything work we should see the file create in our machine.

```bash
ls ~/.config/go-npm/manifest
express.json
```

Like we did for packagejsongo package we are going to create a new test file manifest_test.go

```sh
cd manifest
touch manifest_test.go
```

*manifest/manifest_test.go*
```go
package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setupTestDirs(t *testing.T) string {
	tmpDir := t.TempDir()
	return tmpDir
}

func TestDownloadManifest_Download(t *testing.T) {
	packageName := "express"

	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (string, string)
		expectError bool
		validate    func(t *testing.T, m *Manifest, packageName string)
	}{
		{
			name: "Download express manifest",
			setupFunc: func(t *testing.T) (string, string) {
				configDir := setupTestDirs(t)
				return configDir, packageName
			},
			expectError: false,
			validate: func(t *testing.T, m *Manifest, packageName string) {
				expectedFile := filepath.Join(m.Path, packageName+".json")
				_, err := os.Stat(expectedFile)
				assert.NoError(t, err, "Manifest file should exist")

				info, err := os.Stat(expectedFile)
				assert.NoError(t, err)
				assert.Greater(t, info.Size(), int64(0), "File should not be empty")
			},
		},
		{
			name: "Error with invalid package name",
			setupFunc: func(t *testing.T) (string, string) {
				configDir := setupTestDirs(t)
				return configDir, "this-package-does-not-exist-12345678"
			},
			expectError: true,
			validate: func(t *testing.T, m *Manifest, packageName string) {
				expectedFile := filepath.Join(m.Path, packageName+".json")
				info, err := os.Stat(expectedFile)
				if err == nil {
					assert.Equal(t, int64(0), info.Size(), "File should be empty or not exist")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			configDir, packageName := tc.setupFunc(t)
			manifest, err := NewManifest(configDir, "https://registry.npmjs.org/")
			assert.NoError(t, err)
			_, err = manifest.Download(packageName)

			if tc.expectError {
				assert.Error(t, err, "Expected an error")
			} else {
				assert.NoError(t, err, "Expected no error")
			}

			tc.validate(t, manifest, packageName)
		})
	}
}

```

In this test we add two test cases
- Download express manifest: we expect to download the manifest file correctly and check that the file exist
- Error with invalid package name: we expect an error when try to download a manifest for a non existent package

Like in previous tests we run that in /temp directory to prevent a conflict  
with path ~/.config/go-npm/manifest, that is used by when run install command.

Also note that use the real npm registry url here,  another option is to use a mock library to prevent to make a network call, but for simplicity and also to test real world context, we prefer to go in this way.

Run with 

```bash
go test ./... 
```

You should expect to not have any errors here.



# Version component

Next step is to get the correct version from package json dependencies,  
there is a lot of options in version definitions.

*Examples:*

- Exact version: "express": "5.0.1"
- Caret range: "express": "^5.0.1"
- Tilde range: "express": "~5.0.1"
- Greater than or equal to: "express": ">=5.0.1"
- Less than: "express": "<6.0.0"
- etc


To handle this we will use a external library that help us with semver parsing and comparison.

> Semantic Versioning (SemVer) is a versioning system that uses the format MAJOR.MINOR.PATCH, where:
>
> MAJOR changes break compatibility,
>
> MINOR adds features without breaking existing ones,
>
> PATCH fixes bugs without changing behavior.
>
> It defines clear, predictable rules so tools can automatically resolve and compare versions safely.


Install wit this command

```bash
go get github.com/Masterminds/semver/v3
```


Create a new folder version and a file version.go inside it

```sh
mkdir version
cd version
touch version.go
```

*version/version.go*

```go
package version

import (
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

type VersionInfo struct {
}

func newVersionInfo() *VersionInfo {
	return &VersionInfo{}
}

type NPMPackage struct {
	ID       string             `json:"_id"`
	Rev      string             `json:"_rev"`
	Name     string             `json:"name"`
	DistTags DistTags           `json:"dist-tags"`
	Versions map[string]Version `json:"versions"`
	Time     map[string]string  `json:"time"`
	Bugs     any                `json:"bugs"`
	License  any                `json:"license"`
	Homepage any                `json:"homepage"`
	Keywords any                `json:"keywords"`

	Repository     any             `json:"repository"`
	Description    string          `json:"description"`
	Contributors   any             `json:"contributors"`
	Maintainers    any             `json:"maintainers"`
	Readme         string          `json:"readme"`
	ReadmeFilename string          `json:"readmeFilename"`
	Users          map[string]bool `json:"users"`
}

type DistTags struct {
	Latest string `json:"latest"`
	Next   string `json:"next"`
}

type Version struct {
	Name                   string                 `json:"name"`
	Version                string                 `json:"version"`
	Author                 any                    `json:"author"`
	License                any                    `json:"license"`
	ID                     string                 `json:"_id"`
	Maintainers            any                    `json:"maintainers"`
	Homepage               any                    `json:"homepage"`
	Bugs                   any                    `json:"bugs"`
	Dist                   Dist                   `json:"dist"`
	From                   string                 `json:"_from"`
	Shasum                 string                 `json:"_shasum"`
	Engines                any                    `json:"engines"`
	GitHead                string                 `json:"gitHead"`
	Scripts                any                    `json:"scripts"`
	NPMUser                NPMUser                `json:"_npmUser"`
	Repository             any                    `json:"repository"`
	NPMVersion             string                 `json:"_npmVersion"`
	Description            string                 `json:"description"`
	Directories            map[string]interface{} `json:"directories"`
	NodeVersion            string                 `json:"_nodeVersion"`
	Dependencies           map[string]string      `json:"dependencies"`
	DevDependencies        map[string]string      `json:"devDependencies"`
	OptionalDependencies   map[string]string      `json:"optionalDependencies"`
	PeerDependencies       map[string]string      `json:"peerDependencies"`
	PeerDependenciesMeta   map[string]PeerMeta    `json:"peerDependenciesMeta"`
	OS                     []string               `json:"os"`
	CPU                    []string               `json:"cpu"`
	HasShrinkwrap          bool                   `json:"_hasShrinkwrap"`
	Keywords               any                    `json:"keywords"`
	Contributors           any                    `json:"contributors"`
	Files                  any                    `json:"files"`
	NPMOperationalInternal NPMOperationalInternal `json:"_npmOperationalInternal"`
	NPMSignature           string                 `json:"npm-signature"`
}

type PeerMeta struct {
	Optional bool `json:"optional"`
}

type Author struct {
	URL   string `json:"url"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Maintainer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Contributor struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	URL   string `json:"url"`
}

type Bugs struct {
	URL string `json:"url"`
}

type Dist struct {
	Shasum       string      `json:"shasum"`
	Tarball      string      `json:"tarball"`
	Integrity    string      `json:"integrity"`
	Signatures   []Signature `json:"signatures"`
	FileCount    int         `json:"fileCount"`
	UnpackedSize int         `json:"unpackedSize"`
}

type Signature struct {
	Sig   string `json:"sig"`
	KeyID string `json:"keyid"`
}

type NPMUser struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Repository struct {
	URL  string `json:"url"`
	Type string `json:"type"`
}

type NPMOperationalInternal struct {
	Tmp  string `json:"tmp"`
	Host string `json:"host"`
}

type ParseJsonManifest struct {
}

// getVersion resolves a version constraint to a specific version string
// It supports all npm semver ranges: ^, ~, >=, <=, >, <, ||, hyphen ranges, wildcards, and exact versions
func (v *VersionInfo) GetVersion(version string, npmPackage *NPMPackage) string {
	if version == "" || version == "latest" || version == "*" {
		return npmPackage.DistTags.Latest
	}

	if version == "next" && npmPackage.DistTags.Next != "" {
		return npmPackage.DistTags.Next
	}

	constraint, err := semver.NewConstraint(version)
	if err != nil {
		if versionObj, exists := npmPackage.Versions[version]; exists {
			return versionObj.Version
		}
		return npmPackage.DistTags.Latest
	}

	var matchingVersions []*semver.Version
	for vStr := range npmPackage.Versions {
		semverVersion, err := semver.NewVersion(vStr)
		if err != nil {
			continue 
		}
		if constraint.Check(semverVersion) {
			matchingVersions = append(matchingVersions, semverVersion)
		}
	}

	if len(matchingVersions) == 0 {
		return npmPackage.DistTags.Latest
	}

	sort.Sort(semver.Collection(matchingVersions))
	bestVersion := matchingVersions[len(matchingVersions)-1]

	originalVersion := bestVersion.Original()

	if _, exists := npmPackage.Versions[originalVersion]; exists {
		return originalVersion
	}

	stringVersion := bestVersion.String()
	if _, exists := npmPackage.Versions[stringVersion]; exists {
		return stringVersion
	}

	trimmedOriginal := strings.TrimPrefix(originalVersion, "v")
	if _, exists := npmPackage.Versions[trimmedOriginal]; exists {
		return trimmedOriginal
	}

	trimmedString := strings.TrimPrefix(stringVersion, "v")
	if _, exists := npmPackage.Versions[trimmedString]; exists {
		return trimmedString
	}

	return trimmedOriginal
}

```

- **GetVersion(version string, npmPackage *NPMPackage)**:
    - **Parameters**:
        - `version`: The version constraint string (e.g., `^1.0.0`, `~2.1.0`, `latest`).
        - `npmPackage`: The manifest data containing all available versions and distribution tags.
    - **Returns**: The specific version string that best satisfies the constraint (e.g., `1.2.3`).
    - **Logic**:
        1.  **Handle Special Cases**:
            - If `version` is empty, `latest`, or `*`, returns the version tagged as `latest` in `dist-tags`.
            - If `version` is `next` and a `next` tag exists, returns that version.
        2.  **Parse Constraint**: Uses the `semver` library to parse the version constraint string. If parsing fails, it checks if the version string exists directly in the package versions (exact match fallback).
        3.  **Filter Versions**: Iterates through all available versions in the manifest and collects those that satisfy the constraint.
        4.  **Sort and Select**: Sorts the matching versions in ascending order and selects the highest one.
        5.  **Format Return Value**: Ensures the returned version string matches the format used in the manifest (handling potential `v` prefixes).



### Create test file


*version/version_test.go*

```go
package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestPackage(versions []string, latest string) *NPMPackage {
	pkg := &NPMPackage{
		DistTags: DistTags{
			Latest: latest,
		},
		Versions: make(map[string]Version),
	}

	for _, v := range versions {
		pkg.Versions[v] = Version{
			Version: v,
		}
	}

	return pkg
}

func TestVersionInfo_getVersion(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		versions []string
		latest   string
		expected string
	}{
		// Empty version and latest keyword
		{
			name:     "Empty version should return latest",
			version:  "",
			versions: []string{"1.0.0", "1.1.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "2.0.0",
		},
		{
			name:     "Asterisk wildcard",
			version:  "*",
			versions: []string{"1.0.0", "1.5.0", "2.3.1"},
			latest:   "2.3.1",
			expected: "2.3.1",
		},
		/// more...
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vi := newVersionInfo()
			pkg := createTestPackage(tc.versions, tc.latest)
			result := vi.GetVersion(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}
```
In this file we define a set of unit tests for the GetVersion method, covering a wide range of scenarios and edge cases.

> Full test are found in zip download filepackage manager 

Run test 
```bash
go test -v ./...
```

And verify all tests pass successfully.

We could check this in install command with this code 

*cmd/install.go*

```go

func runInstall(cmd *cobra.Command, args []string) error {
	fmt.Println("Starting installation process...")

	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	manifest, err := manifest.NewManifest(cfg.ManifestDir, cfg.NpmRegistryURL)
	if err != nil {
		panic(err)
	}

	npmPackage, err := manifest.Download("express")
	if err != nil {
		panic(err)
	}

	fmt.Println(npmPackage.Name)

	v := version.NewVersionInfo()
	resolvedVersion := v.GetVersion("^4.0.0", npmPackage)
	fmt.Println("Resolved version:", resolvedVersion)

	return nil
}
```

We harcoded the version in call to v.GetVersion to ^4.0.0, we should get this response

```
Starting installation process...
express
Resolved version: 4.21.2
```
In this case version 4.21.2 is the highest version that satisfies the ^4.0.0 constraint.


# Tarball Component

After obtain the version,  we are prepared to download the tarball file from the npm registry,  we can find the tarball URL 
from the manifest file, in the "dist.tarball" property.


example express  
https://registry.npmjs.org/express

```
dist: {
	shasum: "9e0364d1c74e076d7409d302429a384b10dfbd42",
	tarball: "https://registry.npmjs.org/express/-/express-4.4.1.tgz",
	integrity: "sha512-qIrOJ2/9+0i50PwvWZvrCaram8HPxLsUuc+k/SsWnEV6B3ZbjPdPQ+KNpifLRPD1Kym0z8X4GPPGfCGcyC0O0w==",
	signatures: [
		{
			sig: "MEYCIQDSLSlprZmz+ohIbUtuoCZPuJ3UsfcV7vO1V46ypPZkTQIhALIjRW1XR5XD1F7gGr9CLS3v3Ff1MxYwCJnGB7I0dqpc",
			keyid: "SHA256:jl3bwswu80PjjokCgh0o2w5c2U4LhQAE57gj9cz1kzA"
		}
	]
},
```

Create a tarball package

```bash
mkdir tarball
cd tarball
touch tarball.go
```

add this content

*tarball/tarball.go*

```go
package tarball

import (
	"fmt"
	"go-npm/manifest"
	"go-npm/utils"
	"os"
	"path"
	"path/filepath"
)

type Tarball struct {
	TarballPath string
}

func NewTarball() *Tarball {
	tarballPath := os.TempDir()
	return &Tarball{TarballPath: tarballPath}
}


func (d *Tarball) Download(version string, npmPackage manifest.NPMPackage) (string, error) {
	versionData, ok := npmPackage.Versions[version]
	if !ok {
		return "", fmt.Errorf("version %s not found in package %s", version, npmPackage.Name)
	}

	url := versionData.Dist.Tarball
	filename := path.Base(url)
	filePath := filepath.Join(d.TarballPath, filename)

	_, err := utils.DownloadFile(url, filePath)
	return filePath, err
}
```

- **NewTarball()**:
    - **Returns**: A pointer to a `Tarball` instance initialized with the system's temporary directory path.

- **Download(version string, npmPackage manifest.NPMPackage)**:
    - **Parameters**:
        - `version`: The specific version string to download (e.g., `4.21.2`).
        - `npmPackage`: The manifest data containing the tarball URL.
    - **Returns**: The absolute file path of the downloaded `.tgz` file and an error if the download fails.
    - **Logic**:
        1.  **Lookup Version**: Retrieves the version metadata from `npmPackage.Versions`. Returns an error if the version is not found.
        2.  **Get URL**: Extracts the tarball URL from the `Dist` property (e.g., `https://registry.npmjs.org/express/-/express-4.21.2.tgz`).
        3.  **Determine Filename**: Uses `path.Base` to extract the filename from the URL (e.g., `express-4.21.2.tgz`).
        4.  **Construct Path**: Joins the temporary directory path with the filename.
        5.  **Download**: Calls `utils.DownloadFile` to stream the file content from the URL to the local path.



we can test this in command install

*cmd/install.go*

```go	
func runInstall(cmd *cobra.Command, args []string) error {
	fmt.Println("Starting installation process...")

	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	manifest, err := manifest.NewManifest(cfg.ManifestDir, cfg.NpmRegistryURL)
	if err != nil {
		panic(err)
	}

	npmPackage, err := manifest.Download("express")
	if err != nil {
		panic(err)
	}

	fmt.Println(npmPackage.Name)

	v := version.NewVersionInfo()
	resolvedVersion := v.GetVersion("^4.0.0", npmPackage)
	fmt.Println("Resolved version:", resolvedVersion)

	tarball := tarball.NewTarball()
	if _, err := tarball.Download(resolvedVersion, npmPackage); err != nil {
		panic(err)
	}

	return nil
}
```

if everything works fine we should see the tarball file in /tmp directory

Run 

```bash
ls /tmp | grep 4.21.2
```

express-4.21.2.tgz


Like always add test for tarball.go 

*tarball/tarball_test.go*

```go
package tarball

import (
	"go-npm/manifest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownloadTarball_Download(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (string, manifest.NPMPackage)
		expectError bool
		validate    func(t *testing.T, tb *Tarball, version string, npmPackage manifest.NPMPackage, err error)
	}{
		{
			name: "Download express tarball successfully",
			setupFunc: func(t *testing.T) (string, manifest.NPMPackage) {
				version := "4.18.2"
				url := "https://registry.npmjs.org/express/-/express-4.18.2.tgz"
				pkg := manifest.NPMPackage{
					Name: "express",
					Versions: map[string]manifest.Version{
						version: {
							Dist: manifest.Dist{
								Tarball: url,
							},
						},
					},
				}
				return version, pkg
			},
			expectError: false,
			validate: func(t *testing.T, tb *Tarball, version string, npmPackage manifest.NPMPackage, err error) {
				assert.NoError(t, err, "Download should succeed")

				expectedFile := filepath.Join(tb.TarballPath, "express-4.18.2.tgz")
				info, statErr := os.Stat(expectedFile)
				assert.NoError(t, statErr, "Tarball file should exist")
				assert.Greater(t, info.Size(), int64(0), "File should not be empty")
			},
		},
		{
			name: "Error with invalid tarball URL",
			setupFunc: func(t *testing.T) (string, manifest.NPMPackage) {
				version := "1.0.0"
				url := "https://registry.npmjs.org/invalid-package-12345678/-/invalid-package-12345678-1.0.0.tgz"
				pkg := manifest.NPMPackage{
					Name: "invalid-package-12345678",
					Versions: map[string]manifest.Version{
						version: {
							Dist: manifest.Dist{
								Tarball: url,
							},
						},
					},
				}
				return version, pkg
			},
			expectError: true,
			validate: func(t *testing.T, tb *Tarball, version string, npmPackage manifest.NPMPackage, err error) {
				assert.Error(t, err, "Should return error for non-existent package")
				assert.Contains(t, err.Error(), "HTTP error", "Error should indicate HTTP status problem")

				expectedFile := filepath.Join(tb.TarballPath, "invalid-package-12345678-1.0.0.tgz")
				info, statErr := os.Stat(expectedFile)
				if statErr == nil {
					assert.Equal(t, int64(0), info.Size(), "File should be empty or not exist")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			version, pkg := tc.setupFunc(t)
			tarball := NewTarball()
			_, err := tarball.Download(version, pkg)

			if tc.expectError {
				assert.Error(t, err, "Expected an error")
			} else {
				assert.NoError(t, err, "Expected no error")
			}

			tc.validate(t, tarball, version, pkg, err)
		})
	}
}
```

We create two test cases,  one for successful download of express tarball,  and another for invalid package that should return and error. 
in Validate function we check the following.
- if we expect error or not
- use os.Stat to check if the file exists in the temp directory



# Extractor component

Now that we have the tarball file in /tmp, we are preprared to extract the content and copy in the node_modules directory.
for that we create a new package

```bash
mkdir extractor
cd extractor
touch extractor.go
```

*extractor/extractor.go*
```go
package extractor

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type TGZExtractor struct {
	bufferSize int
}

func NewTGZExtractor() *TGZExtractor {
	return &TGZExtractor{
		bufferSize: 32 * 1024,
	}
}

func (e *TGZExtractor) Extract(srcPath, destPath string) error {
	file, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", srcPath, err)
	}
	defer file.Close()

	bufReader := bufio.NewReaderSize(file, e.bufferSize)

	gzr, err := gzip.NewReader(bufReader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	copyBuffer := make([]byte, e.bufferSize)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		relativePath := e.stripPackagePrefix(header.Name)
		if relativePath == "" {
			continue
		}
		target := filepath.Join(destPath, relativePath)

		if !e.isValidPath(target, destPath) {
			fmt.Printf("Skipping unsafe path: %s\n", header.Name)
			continue
		}

		switch header.Typeflag {
		case tar.TypeReg:
			if err := e.extractFile(tr, target, header, copyBuffer); err != nil {
				return err
			}
		default:
			fmt.Printf("Skipping unsupported file type: %c for %s\n", header.Typeflag, header.Name)
		}
	}

	return nil
}

func (e *TGZExtractor) isValidPath(target string, destPath string) bool {
	cleanDest := filepath.Clean(destPath) + string(os.PathSeparator)
	cleanTarget := filepath.Clean(target)
	return strings.HasPrefix(cleanTarget, cleanDest)
}

func (e *TGZExtractor) extractFile(tr *tar.Reader, target string, header *tar.Header, copyBuffer []byte) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", target, err)
	}

	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", target, err)
	}
	defer f.Close()

	_, err = io.CopyBuffer(f, tr, copyBuffer)
	if err != nil {
		return fmt.Errorf("failed to write file %s: %w", target, err)
	}

	return nil
}

func (e *TGZExtractor) stripPackagePrefix(path string) string {
	if idx := strings.Index(path, "/"); idx != -1 {
		return path[idx+1:]
	}
	return ""
}

```

- **NewTGZExtractor()**:
    - **Returns**: A pointer to a `TGZExtractor` initialized with a 32KB buffer size for efficient I/O operations.

- **Extract(srcPath, destPath string)**:
    - **Parameters**:
        - `srcPath`: The absolute path to the source `.tgz` file.
        - `destPath`: The target directory where contents should be extracted (e.g., `node_modules/package-name`).
    - **Returns**: An error if the extraction fails.
    - **Logic**:
        1.  **Open File**: Opens the source tarball file.
        2.  **Initialize Readers**: Sets up a buffered reader, a Gzip reader for decompression, and a Tar reader for archive access.
        3.  **Iterate Entries**: Loops through each file header in the tar archive.
        4.  **Sanitize Path**: Calls `stripPackagePrefix` to remove the top-level `package/` directory common in NPM tarballs.
        5.  **Security Check**: Verifies that the target path is within the destination directory to prevent path traversal attacks.
        6.  **Extract Entry**:
            - If it's a directory, it's created implicitly by `os.MkdirAll` when extracting files.
            - If it's a regular file, calls `extractFile` to write the content.

- **extractFile(tr *tar.Reader, target string, header *tar.Header, copyBuffer []byte)**:
    - **Logic**:
        - Creates the parent directory for the target file.
        - Creates the file with the permissions specified in the tar header.
        - Copies the file content from the tar reader to the new file using a shared buffer.


Update install command to test this 

*cmd/install.go*
```go
func runInstall(cmd *cobra.Command, args []string) error {
	fmt.Println("Starting installation process...")

	cfg, err := config.New()
	if err != nil {
		panic(err)
	}

	manifest, err := manifest.NewManifest(cfg.ManifestDir, cfg.NpmRegistryURL)
	if err != nil {
		panic(err)
	}

	npmPackage, err := manifest.Download("express")
	if err != nil {
		panic(err)
	}

	fmt.Println(npmPackage.Name)

	v := version.NewVersionInfo()
	resolvedVersion := v.GetVersion("^4.0.0", npmPackage)
	fmt.Println("Resolved version:", resolvedVersion)

	tarball := tarball.NewTarball()
	downloadedPath, err := tarball.Download(resolvedVersion, npmPackage)
	if err != nil {
		panic(err)
	}

	extractor := extractor.NewTGZExtractor()
	destPath := filepath.Join("node_modules", npmPackage.Name)
	if err := extractor.Extract(downloadedPath, destPath); err != nil {
		panic(err)
	}

	fmt.Printf("Package installed to %s\n", destPath)

	return nil
}
```

Run 

```bash
go run . i
```

You should see a node_modules folder created with express package inside.

We can check the express server with this.

```js
node index.js
```

Error 

node:internal/modules/cjs/loader:1386
  throw err;
  ^

Error: Cannot find module 'array-flatten'
Require stack:
- /home/ernesto/code/go-npm/tutorial/code/node_modules/express/lib/router/route.js
- /home/ernesto/code/go-npm/tutorial/code/node_modules/express/lib/router/index.js
- /home/ernesto/code/go-npm/tutorial/code/node_modules/express/lib/application.js
- /home/ernesto/code/go-npm/tutorial/code/node_modules/express/lib/express.js
- /home/ernesto/code/go-npm/tutorial/code/node_modules/express/index.js
- /home/ernesto/code/go-npm/tutorial/code/index.js

This is expected because we only installed the parent dependency express,  but not its child dependencies ,  we can fix this later.



```bash
cat node_modules/express/package.json | grep version
```
Output: 
"version": "4.21.2",


Like always add test for extractor.go

*extractor/extractor_test.go*

```go
package extractor

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// setupTestExtractorDirs creates temporary directories for testing
func setupTestExtractorDirs(t *testing.T) (string, string) {
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "src")
	destDir := filepath.Join(tmpDir, "dest")
	os.MkdirAll(srcDir, 0755)
	os.MkdirAll(destDir, 0755)
	return srcDir, destDir
}

// createTestTarball creates a test .tgz file with specified entries
func createTestTarball(t *testing.T, path string, entries map[string]string) {
	file, err := os.Create(path)
	assert.NoError(t, err)
	defer file.Close()

	gzw := gzip.NewWriter(file)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	for name, content := range entries {
		header := &tar.Header{
			Name:     name,
			Mode:     0644,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		err := tw.WriteHeader(header)
		assert.NoError(t, err)

		_, err = tw.Write([]byte(content))
		assert.NoError(t, err)
	}
}

func TestTGZExtractorStripPackagePrefix(t *testing.T) {
	testCases := []struct {
		name        string
		inputPath   string
		expectedVal string
	}{
		{
			name:        "Strip package prefix successfully",
			inputPath:   "package/index.js",
			expectedVal: "index.js",
		},
		{
			name:        "Strip package prefix from nested path",
			inputPath:   "package/lib/utils.js",
			expectedVal: "lib/utils.js",
		},
		{
			name:        "No package prefix - return empty string",
			inputPath:   "index.js",
			expectedVal: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			extractor := NewTGZExtractor()
			result := extractor.stripPackagePrefix(tc.inputPath)
			assert.Equal(t, tc.expectedVal, result)
		})
	}
}

func TestTGZExtractorExtract(t *testing.T) {
	testCases := []struct {
		name        string
		setupFunc   func(t *testing.T) (string, string)
		expectError bool
		validate    func(t *testing.T, destDir string, err error)
	}{
		{
			name: "Extract tarball with package prefix successfully",
			setupFunc: func(t *testing.T) (string, string) {
				srcDir, destDir := setupTestExtractorDirs(t)
				tarballPath := filepath.Join(srcDir, "test.tgz")

				entries := map[string]string{
					"package/index.js":     "console.log('hello');",
					"package/package.json": "{\"name\":\"test\"}",
					"package/lib/utils.js": "module.exports = {};",
				}
				createTestTarball(t, tarballPath, entries)

				return tarballPath, destDir
			},
			expectError: false,
			validate: func(t *testing.T, destDir string, err error) {
				assert.NoError(t, err, "Extract should succeed")

				indexPath := filepath.Join(destDir, "index.js")
				assert.FileExists(t, indexPath)

				packageJsonPath := filepath.Join(destDir, "package.json")
				assert.FileExists(t, packageJsonPath)

				utilsPath := filepath.Join(destDir, "lib", "utils.js")
				assert.FileExists(t, utilsPath)

				content, readErr := os.ReadFile(indexPath)
				assert.NoError(t, readErr)
				assert.Equal(t, "console.log('hello');", string(content))
			},
		},
		{
			name: "Skip files without directory prefix",
			setupFunc: func(t *testing.T) (string, string) {
				srcDir, destDir := setupTestExtractorDirs(t)
				tarballPath := filepath.Join(srcDir, "test.tgz")

				entries := map[string]string{
					"index.js":  "console.log('no prefix');",
					"README.md": "# Test Package",
				}
				createTestTarball(t, tarballPath, entries)

				return tarballPath, destDir
			},
			expectError: false,
			validate: func(t *testing.T, destDir string, err error) {
				assert.NoError(t, err)

				indexPath := filepath.Join(destDir, "index.js")
				assert.NoFileExists(t, indexPath, "Files without directory prefix should be skipped")

				readmePath := filepath.Join(destDir, "README.md")
				assert.NoFileExists(t, readmePath, "Files without directory prefix should be skipped")
			},
		},
		{
			name: "Error with non-existent tarball file",
			setupFunc: func(t *testing.T) (string, string) {
				srcDir, destDir := setupTestExtractorDirs(t)
				tarballPath := filepath.Join(srcDir, "nonexistent.tgz")
				return tarballPath, destDir
			},
			expectError: true,
			validate: func(t *testing.T, destDir string, err error) {
				assert.Error(t, err, "Should return error for non-existent file")
				assert.Contains(t, err.Error(), "failed to open file")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tarballPath, destDir := tc.setupFunc(t)
			extractor := NewTGZExtractor()
			err := extractor.Extract(tarballPath, destDir)

			if tc.expectError {
				assert.Error(t, err, "Expected an error")
			} else {
				assert.NoError(t, err, "Expected no error")
			}

			tc.validate(t, destDir, err)
		})
	}
}
```

We add two helper function in this test.

- setupTestExtractorDirs to create temporary source and destination directories for testing,  in /temp directory of our machine.
- createTestTarball create a tgz file with specified files for testing various cases.


We add test for stripPackagePrefix method to ensure it correctly removes the leading package directory from file paths.

TestTGZExtractorExtract configure various scenarios for extracting tarball 

We need to setup testing context in order to make sure that all is working as expected, 
for that in setupFunc we call functions setupTestExtractorDirs and createTestTarball to prepare the source tarball and destination directory.

- Expected error or not
- Check if file exists in destination directory 
- Check content of files extracted.


# Manager component 

At moment we have a solid list of components that have a specific task to perform.  

- Manifest:  download and parse package manifest from npm registry
- Version:  resolve version constraints to specific version
- Tarball:  download the tarball file for specific version
- Extractor:  extract the tarball contents into node_modules

We are going to create a new package call manager that abstract the initialization and interaction with all these components.

```bash
mkdir manager
cd manager
touch manager.go
```

*manager/manager.go*

```go
package manager

import (
	"fmt"
	"go-npm/config"
	"go-npm/extractor"
	"go-npm/manifest"
	"go-npm/packagejson"
	"go-npm/tarball"
	"go-npm/version"
	"path/filepath"
)

type Manager struct {
	Config      *config.Config
	Manifest    *manifest.Manifest
	Version     *version.VersionInfo
	Tarball     *tarball.Tarball
	Extractor   *extractor.TGZExtractor
	PackageJSON *packagejson.PackageJSON
}

type job struct {
	Name    string
	Version string
}

func New() (*Manager, error) {
	cfg, err := config.New()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	m, err := manifest.NewManifest(cfg.ManifestDir, cfg.NpmRegistryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to init manifest: %w", err)
	}

	parser := packagejson.NewPackageJSONParser(cfg)
	pkgJSON, err := parser.Parse("package.json")
	if err != nil {
		return nil, fmt.Errorf("failed to parse package.json: %w", err)
	}

	return &Manager{
		Config:      cfg,
		Manifest:    m,
		Version:     version.NewVersionInfo(),
		Tarball:     tarball.NewTarball(),
		Extractor:   extractor.NewTGZExtractor(),
		PackageJSON: pkgJSON,
	}, nil
}

func (m *Manager) Install() error {
	var queue []job
	for name, version := range m.PackageJSON.Dependencies {
		queue = append(queue, job{Name: name, Version: version})
	}

	installed := make(map[string]bool)
	parser := packagejson.NewPackageJSONParser(m.Config)

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if installed[current.Name] {
			continue
		}

		npmPackage, err := m.Manifest.Download(current.Name)
		if err != nil {
			return err
		}

		fmt.Println("Installing:", npmPackage.Name)

		resolvedVersion := m.Version.GetVersion(current.Version, npmPackage)
		fmt.Println("Resolved version:", resolvedVersion)

		downloadedPath, err := m.Tarball.Download(resolvedVersion, npmPackage)
		if err != nil {
			return err
		}

		destPath := filepath.Join("node_modules", npmPackage.Name)
		if err := m.Extractor.Extract(downloadedPath, destPath); err != nil {
			return err
		}

		installedPkgJSONPath := filepath.Join(destPath, "package.json")
		installedPkgJSON, err := parser.Parse(installedPkgJSONPath)
		if err == nil && installedPkgJSON.Dependencies != nil {
			for name, version := range installedPkgJSON.Dependencies {
				if !installed[name] {
					queue = append(queue, job{Name: name, Version: version})
				}
			}
		}

		installed[current.Name] = true
	}

	return nil
}
```

- **New()**:
    - **Returns**: A pointer to a fully initialized `Manager` struct or an error.
    - **Logic**:
        - Loads configuration.
        - Initializes the Manifest, Version, Tarball, and Extractor components.
        - Parses the root `package.json` to get the initial list of dependencies.

- **Install()**:
    - **Returns**: An error if the installation process fails.
    - **Logic**:
        1.  **Initialize Queue**: Creates a job queue and populates it with the direct dependencies from the root `package.json`.
        2.  **Process Queue**: Enters a loop that continues as long as there are jobs in the queue.
        3.  **Check Cache**: Checks the `installed` map to see if the current package has already been processed. If so, it skips to the next job to prevent infinite loops and redundant work.
        4.  **Download Manifest**: Fetches the package metadata from the registry.
        5.  **Resolve Version**: Determines the exact version to install based on the semantic versioning constraint (e.g., `^1.0.0` -> `1.2.3`).
        6.  **Download Tarball**: Downloads the `.tgz` file for the resolved version.
        7.  **Extract**: Unpacks the tarball into the `node_modules` directory.
        8.  **Process Child Dependencies**:
            - Parses the `package.json` of the *newly installed* package.
            - Adds its dependencies to the queue if they haven't been installed yet.
        9.  **Mark Complete**: Adds the package name to the `installed` map.


![demo](https://raw.githubusercontent.com/ernesto27/go-npm/main/tutorial/manager.png)

This will install the root dependencies and all child dependencies recursively,  so we can update install command to test.

*cmd/install.go*
```go
func runInstall(cmd *cobra.Command, args []string) error {
	fmt.Println("Starting installation process...")

	mgr, err := manager.New()
	if err != nil {
		return err
	}

	if err := mgr.Install(); err != nil {
		return err
	}

	fmt.Printf("Package installed to %s\n", filepath.Join("node_modules", "express"))

	return nil
}
```

We remove all the logic and initialization to the New manager method and call the Install method.

Run 

```bash 
go run . i 
```

You should see all dependencies installed in node_modules folder.

```bash
ls node_modules
```

![alt text](https://raw.githubusercontent.com/ernesto27/go-npm/main/tutorial/image-1.png)

And express server 

```bash
node index.js
```
Response 

> Server is running on port 3000

Check with curl 

```bash
curl http://localhost:3000
```

We should see this response

> {"message":"Hello World!"}

## Conclusion

In this tutorial, we have successfully built a functional package manager in Go that mimics the core behavior of `npm`.

We covered:
- **Project Structure**: Setting up a Go project with a clear architecture.
- **Registry Interaction**: Fetching package metadata (manifests) from the npm registry.
- **Version Resolution**: Implementing Semantic Versioning (SemVer) to resolve version constraints like `^1.0.0`.
- **Tarball Handling**: Downloading and extracting `.tgz` package files securely.
- **Dependency Management**: Recursively installing dependencies using a queue-based approach.
- **CLI Integration**: Creating a command-line interface using `cobra`.

You now have a working tool that can install  packages like `express` and its dependencies and run a basic server,
In later tutorials we can explore more features like handling devDependencies, peerDependencies, caching, and publishing packages.


