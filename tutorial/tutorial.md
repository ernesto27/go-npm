# TUTORIAL  

Part 1 

- Explain what this tutorial is about.
- show how npm i works, architecture diagram of similar
- Show example of the end result. 
- dependencies required 
- setup project, folders,  hello world  


folders.
- cmd cobra 
- config 
- extractor
- manager 
- manifest 
- packagecopy 
- packagejson 
- tarball


# Intro

This tutorial is about to create npm package manager version using golang.
We start from scratch with a basic implementation in which we can run this command and run a simple express server, 
this first version does not have all the cache and performance optimizations of npm or other packages have, but it is a good starting point to understand how this works and to get a first glance of system programming.

```bash
go-npm i 
```

We will create a base and solid desing structure to build upon it in future versions, also with testing to ensure that our code is working as expected.


# How npm install works 

Before start coding we need to understand how the command npm install works,  what components are involved and how they interact with each other.
This is a base diagram, we will start simple and not think at moment about cache and performance optimizations.

![npm install diagram](go-npm-i.png)


1. **npm install**
   You run the command to install packages.

2. **Parse package.json**
   npm reads your project’s package.json to know which packages it needs.

3. **Download manifest**
   npm contacts the registry and downloads the package information (metadata), like available versions.

4. **Download tarball**
   npm downloads the actual package file (a .tgz archive).

5. **Extract tarball**
   npm unpacks the .tgz file into normal files and folders.

6. **Copy package to node_modules**
   npm moves the unpacked package into your project's node_modules folder so it can be used.


# Setup project

We are going to use golang as language 1.25, so intall from here 

https://go.dev/doc/install

We also need nodejs installed, check this.
https://nodejs.org/en/download

Create a new folder and initialize go module 

```bash
mkdir go-npm
cd go-npm 
go mod init go-npm 
```

We will use some external dependencies to help us with CLI commands and testing, so let's install them now.

```bash
go get -u github.com/spf13/cobra@v1.10.1
go get -u github.com/stretchr/testify@v1.11.1       
```

Create a main.go 
```go
package main

import (
	"go-npm/cmd"
)

func main() {
	cmd.Execute()
}
```

Create a cmd folder and a root.go file inside it 
```sh 
mkdir cmd
cd cmd
touch root.go
```
In root.go we will setup the base for our CLI commands using Cobra 

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

For check command, create a file in cmd/install.go

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install [package[@version]]",
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
in this file we intialize the install command with alias i, and a simple runInstall function that for now just print a message.

We can run this command with this 

```bash
go run . i
Starting installation process..
```

Some nice features that have cobra by default is the use of command -h that show the availabe commands and descriptions

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

We have created a good starting point for our project, next we will start to download a real package.

# Config component

Like we said we need to download files from npm registry in order to create the correct node_modules,  for that we create a folder for this project in the .config folder of the user home directory.

So, create a config folder and a config.go file inside it

```sh
mkdir config
cd config
touch config.go
```

config.go

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


in this file we define all the config directories that the we need for our package manager, 
some are created in .config/go-npm folder and others are local to the project like node_modules, 
New method is used to initialize all config dirs with the correct paths and create folders in path  .config/go-npm


```
~/.config/go-npm
       │
       ├── manifest
       │
       └── packages
```
in later sections we will go in detail about each folder purpose.


# Parse package json

First step after run the install command is to parse the package.json to obtain the list of dependencies to install
(for now we ignore devDependencies and peerDependencies).

The idea is to install dependencies for a node express server, so create a package.json file with this content

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

Add and index.js to test the server

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

If we run command `node index.js` we will get an error because express is not installed yet.

We need to read the package.json file and parse the dependencies field, for that we will create a new packagejson package that will handle that.

Create a new folder packagejson and a file packagejson.go inside it

```sh
mkdir packagejson
cd packagejson
touch packagejson.go
```

packagejson.go

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

This file create a struct PackageJSON that represent the fields of package.json file, there is a lot of fields that we are not going to use for now, but we define them for future use,  the important field for now is Dependencies that is a map of package name to version string.
like this 

```json
"dependencies": {
    "express": "^5.0.1"
}
```

NewPackageJSONParser method is used to initialize the parser, this receives a config instance struct previously created.
Parse method should read file pass as argument and unmarshal the json content into PackageJSON struct, also intialize fields of the parse for later uses.

### Testing the parser
One of the most important things is this kind of project is test since beginning, even in this infancy stage, critical to add new features and ensure that existing functionality is not broken.

Create a packagejson_test.go file in the same folder 

```sh
touch packagejson_test.go
```

packagejson_test.go

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

We use TableDriveTest pattern to define multiple test cases for the Parse method of PackageJSONParser.

- Valid basic package.json: We create a valid file. We expect no error, and we check that the parser - correctly read the name, version, and dependencies.
- Non-existent file: We don't create a file at all. We expect an error.
- Invalid JSON: We create a file with broken JSON (like a missing bracket). We expect an error.

After we loop through each test case,  we change to the temp dir created for the test case, this is important for not create files in our current working dir,  and run the Parse method.

Run test  

```bash
go test ./...
?       go-npm  [no test files]
?       go-npm/cmd      [no test files]
?       go-npm/config   [no test files]
ok      go-npm/packagejson      0.003s
```

# Manifest component

Ok, after parsing the package.json and get the dependencies to install, we need to obtain the manifest file from npm registry, 
this is necessary to download the correspoding tarball for the package.

For example if we have this express dependency in package.json

```json
"dependencies": {
    "express": "^5.0.1"
}
```
If we go to this url https://registry.npmjs.org/express and obtain the manifest file in json format.

This return a json file with a lot of information about the package, versions, dist-tags, time, maintainers, etc.
we will focus in versions for now, this is a object of all available entries, wit this structure

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

So for that we will create a new manifest package that will handle the manifest fetching and parsing.

Create a new folder manifest and a file manifest.go inside it

```bash
mkdir manifest
cd manifest
touch manifest.go
```

manifest.go

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

We add a NewManifest method to expect as parameter two paramenter
- manifestPath: path where to save the manifest file
- npmRegistryURL: base url of npm registry

various structs are defined to represent the manifest file structure, the parent of all are NPMPackage.

in Download method we get as parameter the name of the package "express" for example, 
check statusCode response from call to DownloadFile function,  
after we read file and unmarshal the json content into NPMPackage struct that represent the manifest file structure and return it.





Create utils package and a utils.go file inside it

```sh
mkdir utils
cd utils
touch utils.go
```

utils.go

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

Here we define two function that will be useful in multiple components
- DownloadFile: download a file from url and save it to filename path
- CreateDir: create a directory if not exist in especified path

Ok, we have the base to check if we can download a manifest file from npm,  to do that update the install file

cmd/install.go
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

here we call config.New method to initialize config properties and folders needed for this project,  after call manifest Download to save manifest file in this path ~/.config/go-npm/manifest/express.json

if everything work we should see the file create in our machine.

```bash
ls ~/.config/go-npm/manifest
express.json
```

Like we did for packagejsongo package we need to create a new test file manifest_test.go

```sh
cd manifest
touch manifest_test.go
```

add this code 
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

in this test we add two test cases
- Download express manifest: we expect to download the manifest file correctly and check that the file exist
- Error with invalid package name: we expect an error when try to download a manifest for a non existent package

We also add a function call setupTestDirs, this is very importatn because set configure the test to run in /temp directory and not make a conflict 
with path ~/.config/go-npm/manifest.

Also note that use the real npm registry url to download, another option is to use a mock libraty to prevent go to internet, but for simplicity and expect real world behavior we go this way.

Run with 

```bash
go test ./... 
```

You should expect to not have any errors here.





# Version component

Next step after manifiest is to get the correct version from package json dependencies,  
there is a lot of formats that we can use

examples:

- Exact version: "express": "5.0.1"
- Caret range: "express": "^5.0.1"
- Tilde range: "express": "~5.0.1"
- Greater than or equal to: "express": ">=5.0.1"
- Less than: "express": "<6.0.0"

Etc. 

to handle this we will create a version component that will parse the version string and resolve the version to download.

First install require lib to parse semver

> SemVer is a versioning system that uses MAJOR.MINOR.PATCH to indicate breaking changes, new features, and bug fixes.

```bash
go get golang.org/x/mod/semver
```




Create a new folder version and a file version.go inside it

```sh
mkdir version
cd version
touch version.go
```

version.go

```go
package manager

import (
	"go-npm/manifest"
	"strings"

	"golang.org/x/mod/semver"
)

type VersionInfo struct {
}

func newVersionInfo() *VersionInfo {
	return &VersionInfo{}
}

func (v *VersionInfo) getVersion(version string, npmPackage *manifest.NPMPackage) string {

	if version == "" {
		return npmPackage.DistTags.Latest
	}

	switch {
	case strings.Contains(version, "||"):
		orVersion := v.getVersionOr(version, npmPackage)
		return orVersion
	case strings.HasPrefix(version, "^"):
		caretVersion := v.getVersionCaret(version, npmPackage)
		return caretVersion
	case strings.HasPrefix(version, "~"):
		tildeVersion := v.getVersionTilde(version, npmPackage)
		return tildeVersion
	case strings.Contains(version, ">=") && (strings.Contains(version, "<") || strings.Contains(version, "<=")):
		complexVersion := v.getVersionComplexRange(version, npmPackage)
		return complexVersion
	case strings.HasPrefix(version, ">="):
		return v.getVersionGreaterOrEqual(version, npmPackage)
	case strings.HasPrefix(version, "<="):
		return v.getVersionLessOrEqual(version, npmPackage)
	case strings.HasPrefix(version, ">"):
		return v.getVersionGreater(version, npmPackage)
	case strings.HasPrefix(version, "<"):
		return v.getVersionLess(version, npmPackage)
	case strings.Contains(version, " - "):
		return v.getVersionHyphenRange(version, npmPackage)
	case version == "*" || version == "latest":
		return npmPackage.DistTags.Latest
	case strings.Contains(version, "x") || strings.Contains(version, "X"):
		wildcardVersion := v.getVersionWildcard(version, npmPackage)
		return wildcardVersion
	default:
		parts := strings.Split(version, ".")
		if len(parts) == 3 {
			npmVersion, exists := npmPackage.Versions[version]
			if exists && npmVersion.Version == version {
				return npmVersion.Version
			}

		}
		return npmPackage.DistTags.Latest
	}
}

func (v *VersionInfo) getVersionCaret(version string, npmPackage *manifest.NPMPackage) string {
	baseVersion := strings.Replace(version, "^", "", 1)
	v1 := "v" + baseVersion

	var bestVersion string
	var bestSemver string

	for k := range npmPackage.Versions {
		v2 := "v" + k
		if semver.Compare(v2, v1) >= 0 {
			majorBase := semver.Major(v1)
			majorCandidate := semver.Major(v2)

			if majorBase == majorCandidate {
				// For major version 0, caret behaves like tilde (also match minor)
				// ^0.2.3 means >=0.2.3 <0.3.0
				if majorBase == "v0" {
					minorBase := semver.MajorMinor(v1)
					minorCandidate := semver.MajorMinor(v2)
					if minorBase != minorCandidate {
						continue
					}
				}

				if bestSemver == "" || semver.Compare(v2, bestSemver) > 0 {
					bestVersion = k
					bestSemver = v2
				}
			}
		}
	}

	return bestVersion
}

func (v *VersionInfo) getVersionTilde(version string, npmPackage *manifest.NPMPackage) string {
	baseVersion := strings.Replace(version, "~", "", 1)
	v1 := "v" + baseVersion

	var bestVersion string
	var bestSemver string

	for k := range npmPackage.Versions {
		v2 := "v" + k
		if semver.Compare(v2, v1) >= 0 {
			// For tilde, we need to match the major and minor versions exactly
			majorBase := semver.Major(v1)
			minorBase := semver.MajorMinor(v1)
			majorCandidate := semver.Major(v2)
			minorCandidate := semver.MajorMinor(v2)

			// Tilde allows patch-level changes if minor version is specified
			// ~1.2.3 := >=1.2.3 <1.(2+1).0 := >=1.2.3 <1.3.0
			if majorBase == majorCandidate && minorBase == minorCandidate {
				if bestSemver == "" || semver.Compare(v2, bestSemver) > 0 {
					bestVersion = k
					bestSemver = v2
				}
			}
		}
	}

	return bestVersion
}

func (v *VersionInfo) getVersionComplexRange(version string, npmPackage *manifest.NPMPackage) string {

	var lowerBound, upperBound string
	var lowerInclusive, upperInclusive bool

	// Parse the complex range (e.g., ">= 2.1.2 < 3.0.0")
	parts := strings.Fields(version)

	for i := 0; i < len(parts)-1; i += 2 {
		operator := parts[i]
		versionStr := parts[i+1]

		switch operator {
		case ">=":
			lowerBound = versionStr
			lowerInclusive = true
		case ">":
			lowerBound = versionStr
			lowerInclusive = false
		case "<=":
			upperBound = versionStr
			upperInclusive = true
		case "<":
			upperBound = versionStr
			upperInclusive = false
		}
	}

	var bestVersion string
	var bestSemver string

	for k := range npmPackage.Versions {
		vCandidate := "v" + k

		// Check lower bound
		if lowerBound != "" {
			vLower := "v" + lowerBound
			comparison := semver.Compare(vCandidate, vLower)
			if lowerInclusive && comparison < 0 {
				continue
			}
			if !lowerInclusive && comparison <= 0 {
				continue
			}
		}

		// Check upper bound
		if upperBound != "" {
			vUpper := "v" + upperBound
			comparison := semver.Compare(vCandidate, vUpper)
			if upperInclusive && comparison > 0 {
				continue
			}
			if !upperInclusive && comparison >= 0 {
				continue
			}
		}

		// This version satisfies both bounds, check if it's the best one
		if bestSemver == "" || semver.Compare(vCandidate, bestSemver) > 0 {
			bestVersion = k
			bestSemver = vCandidate
		}
	}

	return bestVersion
}

func (v *VersionInfo) getVersionWildcard(version string, npmPackage *manifest.NPMPackage) string {
	normalized := strings.ToLower(version)
	parts := strings.Split(normalized, ".")

	// Handle different wildcard patterns:
	// "x" or "x.x.x" -> any version (use latest)
	// "1.x" or "1.x.x" -> any minor/patch in major 1
	// "1.2.x" -> any patch in 1.2

	if len(parts) == 1 && parts[0] == "x" {
		// "x" means any version
		return npmPackage.DistTags.Latest
	}

	var major, minor string
	var matchMinor bool

	if len(parts) >= 1 && parts[0] != "x" {
		major = parts[0]
	}
	if len(parts) >= 2 && parts[1] != "x" {
		minor = parts[1]
		matchMinor = true
	}

	var bestVersion string
	var bestSemver string

	for k := range npmPackage.Versions {
		vCandidate := "v" + k
		candidateParts := strings.Split(k, ".")

		if len(candidateParts) < 2 {
			continue
		}

		// Match major version if specified
		if major != "" && candidateParts[0] != major {
			continue
		}

		// Match minor version if specified
		if matchMinor && len(candidateParts) >= 2 && candidateParts[1] != minor {
			continue
		}

		// This version matches the pattern, check if it's the best one
		if bestSemver == "" || semver.Compare(vCandidate, bestSemver) > 0 {
			bestVersion = k
			bestSemver = vCandidate
		}
	}

	return bestVersion
}

func (v *VersionInfo) getVersionOr(version string, npmPackage *manifest.NPMPackage) string {
	// Split by || to get alternative constraints
	constraints := strings.Split(version, "||")

	var bestVersion string
	var bestSemver string

	// Try each constraint and find the highest version that satisfies any of them
	for _, constraint := range constraints {
		constraint = strings.TrimSpace(constraint)

		// Recursively resolve each constraint
		resolvedVersion := v.getVersion(constraint, npmPackage)

		if resolvedVersion != "" {
			vCandidate := "v" + resolvedVersion

			// Keep track of the highest version found
			if bestSemver == "" || semver.Compare(vCandidate, bestSemver) > 0 {
				bestVersion = resolvedVersion
				bestSemver = vCandidate
			}
		}
	}

	return bestVersion
}

func (v *VersionInfo) getVersionGreaterOrEqual(version string, npmPackage *manifest.NPMPackage) string {
	baseVersion := strings.TrimSpace(strings.TrimPrefix(version, ">="))
	vBase := "v" + baseVersion

	var bestVersion string
	var bestSemver string

	for k := range npmPackage.Versions {
		vCandidate := "v" + k
		if semver.Compare(vCandidate, vBase) >= 0 {
			if bestSemver == "" || semver.Compare(vCandidate, bestSemver) > 0 {
				bestVersion = k
				bestSemver = vCandidate
			}
		}
	}

	return bestVersion
}

func (v *VersionInfo) getVersionLessOrEqual(version string, npmPackage *manifest.NPMPackage) string {
	baseVersion := strings.TrimSpace(strings.TrimPrefix(version, "<="))
	vBase := "v" + baseVersion

	var bestVersion string
	var bestSemver string

	for k := range npmPackage.Versions {
		vCandidate := "v" + k
		if semver.Compare(vCandidate, vBase) <= 0 {
			if bestSemver == "" || semver.Compare(vCandidate, bestSemver) > 0 {
				bestVersion = k
				bestSemver = vCandidate
			}
		}
	}

	return bestVersion
}

func (v *VersionInfo) getVersionGreater(version string, npmPackage *manifest.NPMPackage) string {
	baseVersion := strings.TrimSpace(strings.TrimPrefix(version, ">"))
	vBase := "v" + baseVersion

	var bestVersion string
	var bestSemver string

	for k := range npmPackage.Versions {
		vCandidate := "v" + k
		if semver.Compare(vCandidate, vBase) > 0 {
			if bestSemver == "" || semver.Compare(vCandidate, bestSemver) > 0 {
				bestVersion = k
				bestSemver = vCandidate
			}
		}
	}

	return bestVersion
}

func (v *VersionInfo) getVersionLess(version string, npmPackage *manifest.NPMPackage) string {
	baseVersion := strings.TrimSpace(strings.TrimPrefix(version, "<"))
	vBase := "v" + baseVersion

	var bestVersion string
	var bestSemver string

	for k := range npmPackage.Versions {
		vCandidate := "v" + k
		if semver.Compare(vCandidate, vBase) < 0 {
			if bestSemver == "" || semver.Compare(vCandidate, bestSemver) > 0 {
				bestVersion = k
				bestSemver = vCandidate
			}
		}
	}

	return bestVersion
}

func (v *VersionInfo) getVersionHyphenRange(version string, npmPackage *manifest.NPMPackage) string {
	parts := strings.Split(version, " - ")
	if len(parts) != 2 {
		return npmPackage.DistTags.Latest
	}

	lowerBound := strings.TrimSpace(parts[0])
	upperBound := strings.TrimSpace(parts[1])
	vLower := "v" + lowerBound
	vUpper := "v" + upperBound

	var bestVersion string
	var bestSemver string

	for k := range npmPackage.Versions {
		vCandidate := "v" + k

		if semver.Compare(vCandidate, vLower) >= 0 && semver.Compare(vCandidate, vUpper) <= 0 {
			if bestSemver == "" || semver.Compare(vCandidate, bestSemver) > 0 {
				bestVersion = k
				bestSemver = vCandidate
			}
		}
	}

	return bestVersion
}
```

getVersion expect two parameters.   
- version: the version string from package.json, example 
- npmPackage: the manifest struct obtained before

The resolver reads the version requirement and determines whether it is a caret range, tilde range, wildcard, exact version, comparative range, or a tag like "latest".

It then loads all available versions of the package and filters them using SemVer rules. Examples:

- "^1.2.3" → any version ≥1.2.3 and <2.0.0

- "~1.2.0" → any version ≥1.2.0 and <1.3.0

- "1.x" → any version with major 1

- "1.2.x" → any version in the 1.2 line

- ">=1.0.0 <2.0.0" → versions from 1.0.0 up to (but not including) 2.0.0

After filtering, the resolver sorts the versions by SemVer order and picks the highest version that satisfies the constraint.
Example: for ["1.0.0", "1.5.0", "1.9.3"] and "^1.0.0", it selects "1.9.3".

Add test  

version/version_test.go

```go
package manager

import (
	"go-npm/manifest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func createTestPackage(versions []string, latest string) *manifest.NPMPackage {
	pkg := &manifest.NPMPackage{
		DistTags: manifest.DistTags{
			Latest: latest,
		},
		Versions: make(map[string]manifest.Version),
	}

	for _, v := range versions {
		pkg.Versions[v] = manifest.Version{
			Version: v,
		}
	}

	return pkg
}

func TestVersionInfo_getVersion_EmptyVersion(t *testing.T) {
	vi := newVersionInfo()
	pkg := createTestPackage([]string{"1.0.0", "1.1.0", "2.0.0"}, "2.0.0")

	result := vi.getVersion("", pkg)
	assert.Equal(t, "2.0.0", result, "Empty version should return latest")
}

func TestVersionInfo_getVersion_Latest(t *testing.T) {
	vi := newVersionInfo()
	pkg := createTestPackage([]string{"1.0.0", "1.5.0", "2.3.1"}, "2.3.1")

	testCases := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "Asterisk wildcard",
			version:  "*",
			expected: "2.3.1",
		},
		{
			name:     "Latest keyword",
			version:  "latest",
			expected: "2.3.1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := vi.getVersion(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVersionInfo_getVersion_ExactVersion(t *testing.T) {
	vi := newVersionInfo()
	pkg := createTestPackage([]string{"1.0.0", "1.2.3", "2.0.0"}, "2.0.0")

	testCases := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "Exact version exists",
			version:  "1.2.3",
			expected: "1.2.3",
		},
		{
			name:     "Exact version does not exist",
			version:  "1.2.4",
			expected: "2.0.0", // Falls back to latest
		},
		{
			name:     "Exact version with two parts only",
			version:  "1.2",
			expected: "2.0.0", // Falls back to latest (not 3 parts)
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := vi.getVersion(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVersionInfo_getVersionCaret(t *testing.T) {
	testCases := []struct {
		name      string
		version   string
		versions  []string
		latest    string
		expected  string
		expectErr bool
	}{
		{
			name:     "Caret allows minor and patch updates - major 1",
			version:  "^1.2.3",
			versions: []string{"1.0.0", "1.2.3", "1.2.5", "1.3.0", "1.9.9", "2.0.0", "2.1.0"},
			latest:   "2.1.0",
			expected: "1.9.9", // Highest in major version 1
		},
		{
			name:     "Caret with major version 0",
			version:  "^0.2.3",
			versions: []string{"0.1.0", "0.2.3", "0.2.5", "0.3.0", "1.0.0"},
			latest:   "1.0.0",
			expected: "0.2.5", // For 0.x, ^0.2.3 means >=0.2.3 <0.3.0 (only patch updates)
		},
		{
			name:     "Caret with exact match only",
			version:  "^1.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "1.0.0",
		},
		{
			name:     "Caret with multiple candidates",
			version:  "^2.0.0",
			versions: []string{"1.9.9", "2.0.0", "2.0.1", "2.1.0", "2.5.7", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.5.7",
		},
		{
			name:     "Caret with no matching versions",
			version:  "^5.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
			latest:   "4.0.0",
			expected: "", // No version satisfies
		},
		{
			name:     "Caret with lower base version",
			version:  "^1.0.0",
			versions: []string{"0.9.0", "1.0.0", "1.1.0", "1.2.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.2.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vi := newVersionInfo()
			pkg := createTestPackage(tc.versions, tc.latest)
			result := vi.getVersionCaret(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVersionInfo_getVersionTilde(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		versions []string
		latest   string
		expected string
	}{
		{
			name:     "Tilde allows patch updates only",
			version:  "~1.2.3",
			versions: []string{"1.0.0", "1.2.3", "1.2.5", "1.2.9", "1.3.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.2.9", // Highest patch in 1.2.x
		},
		{
			name:     "Tilde with exact match only",
			version:  "~1.2.3",
			versions: []string{"1.2.3", "1.3.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.2.3",
		},
		{
			name:     "Tilde with no higher patch version",
			version:  "~2.1.5",
			versions: []string{"2.0.0", "2.1.0", "2.1.3", "2.1.5", "2.2.0"},
			latest:   "2.2.0",
			expected: "2.1.5",
		},
		{
			name:     "Tilde with multiple patch versions",
			version:  "~3.0.0",
			versions: []string{"2.9.9", "3.0.0", "3.0.1", "3.0.5", "3.0.10", "3.1.0"},
			latest:   "3.1.0",
			expected: "3.0.10",
		},
		{
			name:     "Tilde with no matching versions",
			version:  "~5.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
			latest:   "4.0.0",
			expected: "", // No version satisfies
		},
		{
			name:     "Tilde excludes minor version changes",
			version:  "~1.2.0",
			versions: []string{"1.1.9", "1.2.0", "1.2.1", "1.3.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.2.1", // Does not include 1.3.0
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vi := newVersionInfo()
			pkg := createTestPackage(tc.versions, tc.latest)
			result := vi.getVersionTilde(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVersionInfo_getVersionComplexRange(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		versions []string
		latest   string
		expected string
	}{
		{
			name:     "Range with >= and <",
			version:  ">= 2.1.2 < 3.0.0",
			versions: []string{"2.0.0", "2.1.0", "2.1.2", "2.5.0", "2.9.9", "3.0.0", "3.1.0"},
			latest:   "3.1.0",
			expected: "2.9.9",
		},
		{
			name:     "Range with >= and <= (inclusive)",
			version:  ">= 1.0.0 <= 2.0.0",
			versions: []string{"0.9.0", "1.0.0", "1.5.0", "2.0.0", "2.1.0"},
			latest:   "2.1.0",
			expected: "2.0.0",
		},
		{
			name:     "Range with > and < (exclusive)",
			version:  "> 1.0.0 < 2.0.0",
			versions: []string{"1.0.0", "1.0.1", "1.5.0", "1.9.9", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.9.9",
		},
		{
			name:     "Range with > and <= (mixed)",
			version:  "> 1.5.0 <= 2.5.0",
			versions: []string{"1.5.0", "1.6.0", "2.0.0", "2.5.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.5.0",
		},
		{
			name:     "Range with no matching versions",
			version:  ">= 5.0.0 < 6.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
			latest:   "4.0.0",
			expected: "",
		},
		{
			name:     "Narrow range with one match",
			version:  ">= 1.2.3 < 1.2.5",
			versions: []string{"1.2.0", "1.2.3", "1.2.4", "1.2.5", "1.3.0"},
			latest:   "1.3.0",
			expected: "1.2.4",
		},
		{
			name:     "Range at boundary (lower bound inclusive)",
			version:  ">= 1.0.0 < 2.0.0",
			versions: []string{"0.9.9", "1.0.0", "1.5.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.5.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vi := newVersionInfo()
			pkg := createTestPackage(tc.versions, tc.latest)
			result := vi.getVersionComplexRange(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVersionInfo_getVersionWildcard(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		versions []string
		latest   string
		expected string
	}{
		{
			name:     "Single x returns latest",
			version:  "x",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "3.0.0",
		},
		{
			name:     "Major.x matches any minor/patch in that major",
			version:  "1.x",
			versions: []string{"1.0.0", "1.2.0", "1.5.9", "2.0.0", "2.1.0"},
			latest:   "2.1.0",
			expected: "1.5.9",
		},
		{
			name:     "Major.minor.x matches any patch",
			version:  "2.1.x",
			versions: []string{"2.0.0", "2.1.0", "2.1.5", "2.1.9", "2.2.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.1.9",
		},
		{
			name:     "Case insensitive X",
			version:  "1.X",
			versions: []string{"1.0.0", "1.3.0", "1.7.2", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.7.2",
		},
		{
			name:     "Major.X.X pattern",
			version:  "2.X.X",
			versions: []string{"1.9.9", "2.0.0", "2.1.0", "2.5.7", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.5.7",
		},
		{
			name:     "No matching versions for wildcard",
			version:  "5.x",
			versions: []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
			latest:   "4.0.0",
			expected: "",
		},
		{
			name:     "Wildcard with exact major match",
			version:  "3.x",
			versions: []string{"3.0.0", "3.0.1", "3.1.0", "4.0.0"},
			latest:   "4.0.0",
			expected: "3.1.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vi := newVersionInfo()
			pkg := createTestPackage(tc.versions, tc.latest)
			result := vi.getVersionWildcard(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVersionInfo_getVersionOr(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		versions []string
		latest   string
		expected string
	}{
		{
			name:     "OR with two caret ranges",
			version:  "^1.0.0 || ^2.0.0",
			versions: []string{"1.0.0", "1.2.0", "1.9.9", "2.0.0", "2.1.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.1.0", // Highest between 1.9.9 and 2.1.0
		},
		{
			name:     "OR with exact versions",
			version:  "1.0.0 || 2.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.0.0", // Higher of the two
		},
		{
			name:     "OR with one matching constraint",
			version:  "^1.0.0 || ^5.0.0",
			versions: []string{"1.0.0", "1.5.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "1.5.0", // Only ^1.0.0 matches
		},
		{
			name:     "OR with tilde and caret",
			version:  "~1.2.3 || ^2.0.0",
			versions: []string{"1.2.3", "1.2.5", "1.3.0", "2.0.0", "2.5.0"},
			latest:   "2.5.0",
			expected: "2.5.0", // ^2.0.0 gives higher version
		},
		{
			name:     "OR with no matching constraints",
			version:  "^5.0.0 || ^6.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
			latest:   "4.0.0",
			expected: "",
		},
		{
			name:     "OR with wildcards",
			version:  "1.x || 3.x",
			versions: []string{"1.0.0", "1.5.0", "2.0.0", "3.0.0", "3.2.0"},
			latest:   "3.2.0",
			expected: "3.2.0", // Highest between 1.5.0 and 3.2.0
		},
		{
			name:     "OR with multiple constraints (3 options)",
			version:  "^1.0.0 || ^2.0.0 || ^3.0.0",
			versions: []string{"1.0.0", "1.1.0", "2.0.0", "2.2.0", "3.0.0", "3.5.0"},
			latest:   "3.5.0",
			expected: "3.5.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vi := newVersionInfo()
			pkg := createTestPackage(tc.versions, tc.latest)
			result := vi.getVersionOr(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVersionInfo_getVersion_SimpleRanges(t *testing.T) {
	vi := newVersionInfo()
	pkg := createTestPackage([]string{"1.0.0", "2.0.0", "3.0.0"}, "3.0.0")

	testCases := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "Greater than or equal",
			version:  ">=1.0.0",
			expected: "3.0.0",
		},
		{
			name:     "Less than or equal",
			version:  "<=2.0.0",
			expected: "2.0.0",
		},
		{
			name:     "Greater than",
			version:  ">1.0.0",
			expected: "3.0.0",
		},
		{
			name:     "Less than",
			version:  "<2.0.0",
			expected: "1.0.0",
		},
		{
			name:     "Hyphen range",
			version:  "1.0.0 - 2.0.0",
			expected: "2.0.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := vi.getVersion(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVersionInfo_getVersionGreaterOrEqual(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		versions []string
		latest   string
		expected string
	}{
		{
			name:     "Greater or equal - returns highest version >= base",
			version:  ">=1.5.0",
			versions: []string{"1.0.0", "1.5.0", "1.8.0", "2.0.0", "2.5.0"},
			latest:   "2.5.0",
			expected: "2.5.0",
		},
		{
			name:     "Greater or equal - exact match at base",
			version:  ">=2.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "3.0.0",
		},
		{
			name:     "Greater or equal - no matching versions",
			version:  ">=5.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
			latest:   "4.0.0",
			expected: "",
		},
		{
			name:     "Greater or equal - all versions match",
			version:  ">=0.1.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "3.0.0",
		},
		{
			name:     "Greater or equal - with patch versions",
			version:  ">=1.2.3",
			versions: []string{"1.2.0", "1.2.3", "1.2.5", "1.3.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "2.0.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vi := newVersionInfo()
			pkg := createTestPackage(tc.versions, tc.latest)
			result := vi.getVersionGreaterOrEqual(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVersionInfo_getVersionLessOrEqual(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		versions []string
		latest   string
		expected string
	}{
		{
			name:     "Less or equal - returns highest version <= base",
			version:  "<=2.0.0",
			versions: []string{"1.0.0", "1.5.0", "2.0.0", "2.5.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.0.0",
		},
		{
			name:     "Less or equal - exact match at base",
			version:  "<=2.0.0",
			versions: []string{"1.0.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "2.0.0",
		},
		{
			name:     "Less or equal - no matching versions",
			version:  "<=0.5.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "",
		},
		{
			name:     "Less or equal - all versions match",
			version:  "<=10.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "3.0.0",
		},
		{
			name:     "Less or equal - with patch versions",
			version:  "<=1.2.5",
			versions: []string{"1.0.0", "1.2.3", "1.2.5", "1.2.7", "1.3.0"},
			latest:   "1.3.0",
			expected: "1.2.5",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vi := newVersionInfo()
			pkg := createTestPackage(tc.versions, tc.latest)
			result := vi.getVersionLessOrEqual(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVersionInfo_getVersionGreater(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		versions []string
		latest   string
		expected string
	}{
		{
			name:     "Greater than - returns highest version > base",
			version:  ">1.5.0",
			versions: []string{"1.0.0", "1.5.0", "1.8.0", "2.0.0", "2.5.0"},
			latest:   "2.5.0",
			expected: "2.5.0",
		},
		{
			name:     "Greater than - excludes exact match",
			version:  ">2.0.0",
			versions: []string{"1.0.0", "2.0.0", "2.0.1", "3.0.0"},
			latest:   "3.0.0",
			expected: "3.0.0",
		},
		{
			name:     "Greater than - no matching versions",
			version:  ">5.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0", "5.0.0"},
			latest:   "5.0.0",
			expected: "",
		},
		{
			name:     "Greater than - single match",
			version:  ">1.0.0",
			versions: []string{"0.9.0", "1.0.0", "1.0.1"},
			latest:   "1.0.1",
			expected: "1.0.1",
		},
		{
			name:     "Greater than - with patch versions",
			version:  ">1.2.3",
			versions: []string{"1.2.0", "1.2.3", "1.2.4", "1.3.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "2.0.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vi := newVersionInfo()
			pkg := createTestPackage(tc.versions, tc.latest)
			result := vi.getVersionGreater(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVersionInfo_getVersionLess(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		versions []string
		latest   string
		expected string
	}{
		{
			name:     "Less than - returns highest version < base",
			version:  "<2.0.0",
			versions: []string{"1.0.0", "1.5.0", "1.9.9", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "1.9.9",
		},
		{
			name:     "Less than - excludes exact match",
			version:  "<2.0.0",
			versions: []string{"1.0.0", "1.5.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.5.0",
		},
		{
			name:     "Less than - no matching versions",
			version:  "<1.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "",
		},
		{
			name:     "Less than - all versions match",
			version:  "<10.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "3.0.0",
		},
		{
			name:     "Less than - with patch versions",
			version:  "<1.3.0",
			versions: []string{"1.0.0", "1.2.3", "1.2.9", "1.3.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.2.9",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vi := newVersionInfo()
			pkg := createTestPackage(tc.versions, tc.latest)
			result := vi.getVersionLess(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVersionInfo_getVersionHyphenRange(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		versions []string
		latest   string
		expected string
	}{
		{
			name:     "Hyphen range - inclusive on both ends",
			version:  "1.0.0 - 2.0.0",
			versions: []string{"0.9.0", "1.0.0", "1.5.0", "2.0.0", "2.1.0"},
			latest:   "2.1.0",
			expected: "2.0.0",
		},
		{
			name:     "Hyphen range - narrow range",
			version:  "1.2.3 - 1.2.5",
			versions: []string{"1.2.0", "1.2.3", "1.2.4", "1.2.5", "1.3.0"},
			latest:   "1.3.0",
			expected: "1.2.5",
		},
		{
			name:     "Hyphen range - no matching versions",
			version:  "5.0.0 - 6.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0", "4.0.0"},
			latest:   "4.0.0",
			expected: "",
		},
		{
			name:     "Hyphen range - single version in range",
			version:  "1.5.0 - 2.5.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "2.0.0",
		},
		{
			name:     "Hyphen range - all versions in range",
			version:  "0.1.0 - 10.0.0",
			versions: []string{"1.0.0", "2.0.0", "3.0.0"},
			latest:   "3.0.0",
			expected: "3.0.0",
		},
		{
			name:     "Hyphen range - exact boundaries",
			version:  "2.0.0 - 3.0.0",
			versions: []string{"1.9.9", "2.0.0", "2.5.0", "3.0.0", "3.0.1"},
			latest:   "3.0.1",
			expected: "3.0.0",
		},
		{
			name:     "Hyphen range - malformed (missing space)",
			version:  "1.0.0-2.0.0",
			versions: []string{"1.0.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "2.0.0", // Falls back to latest
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vi := newVersionInfo()
			pkg := createTestPackage(tc.versions, tc.latest)
			result := vi.getVersionHyphenRange(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVersionInfo_getVersion_Integration(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		versions []string
		latest   string
		expected string
	}{
		{
			name:     "Caret range via getVersion",
			version:  "^1.2.0",
			versions: []string{"1.0.0", "1.2.0", "1.5.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.5.0",
		},
		{
			name:     "Tilde range via getVersion",
			version:  "~2.1.0",
			versions: []string{"2.0.0", "2.1.0", "2.1.5", "2.2.0"},
			latest:   "2.2.0",
			expected: "2.1.5",
		},
		{
			name:     "Complex range via getVersion",
			version:  ">= 1.0.0 < 2.0.0",
			versions: []string{"0.9.0", "1.0.0", "1.5.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.5.0",
		},
		{
			name:     "Wildcard via getVersion",
			version:  "1.x",
			versions: []string{"1.0.0", "1.9.0", "2.0.0"},
			latest:   "2.0.0",
			expected: "1.9.0",
		},
		{
			name:     "OR constraint via getVersion",
			version:  "^1.0.0 || ^2.0.0",
			versions: []string{"1.0.0", "1.5.0", "2.0.0", "2.3.0"},
			latest:   "2.3.0",
			expected: "2.3.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vi := newVersionInfo()
			pkg := createTestPackage(tc.versions, tc.latest)
			result := vi.getVersion(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVersionInfo_EdgeCases(t *testing.T) {
	testCases := []struct {
		name     string
		version  string
		versions []string
		latest   string
		expected string
	}{
		{
			name:     "Package with only one version",
			version:  "^1.0.0",
			versions: []string{"1.0.0"},
			latest:   "1.0.0",
			expected: "1.0.0",
		},
		{
			name:     "Empty versions map",
			version:  "^1.0.0",
			versions: []string{},
			latest:   "",
			expected: "",
		},
		{
			name:     "Version with prerelease tag (treated as string)",
			version:  "1.0.0-beta.1",
			versions: []string{"1.0.0-beta.1", "1.0.0"},
			latest:   "1.0.0",
			expected: "1.0.0", // Falls back to latest (not 3 numeric parts)
		},
		{
			name:     "Very high version numbers",
			version:  "^100.200.300",
			versions: []string{"100.200.300", "100.200.400", "100.300.0", "200.0.0"},
			latest:   "200.0.0",
			expected: "100.300.0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vi := newVersionInfo()
			pkg := createTestPackage(tc.versions, tc.latest)
			result := vi.getVersion(tc.version, pkg)
			assert.Equal(t, tc.expected, result)
		})
	}
}
```

There is complete suite of test of most of the cases for version parsing and selection logic.

Run test 
```bash
go test -v ./...
```

And verify all tests pass successfully.