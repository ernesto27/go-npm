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
	"os"
	"path/filepath"
)

type Config struct {
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

	return &Config{
		BaseDir:     baseDir,
		ManifestDir: filepath.Join(baseDir, "manifest"),
		TarballDir:  filepath.Join(baseDir, "tarball"),
		PackagesDir: filepath.Join(baseDir, "packages"),

		LocalNodeModules: "./node_modules",
	}, nil
}
```


in this file we define all the config directories that the we need for our package manager, 
some are created in .config/go-npm folder and others are local to the project like node_modules, 
New method is used to initialize all config dirs with the correct paths.


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



