# go-npm


## Quick Start

### Prerequisites

- Go 1.25 or higher

### Build

```bash
# Build the binary
go build -o go-npm

# Or run directly
go run main.go <command>
```

## Commands

### install (alias: `i`)

Install packages from `package.json` or install a specific package.

```bash
# Install all dependencies from package.json
./go-npm install
./go-npm i

# Install only production dependencies (skip devDependencies)
./go-npm install --production

# Install with verbose output
./go-npm install -v
./go-npm install --verbose

# Skip lifecycle scripts
./go-npm install --ignore-scripts

# Install a specific package globally
./go-npm install -g <package>
./go-npm install --global <package>[@version]

# Examples
./go-npm install -g lodash
./go-npm install -g typescript@5.0.0
./go-npm i --production --verbose
```

**Flags:**
| Flag | Description |
|------|-------------|
| `-g, --global` | Install package globally to `~/.config/go-npm/global/` |
| `-v, --verbose` | Show verbose output with all installed packages |
| `--production` | Install only production dependencies, skip devDependencies |
| `--ignore-scripts` | Skip running lifecycle scripts (preinstall, install, postinstall) |

### add

Add a package to `package.json` dependencies and install it.

```bash
# Add a package (latest version)
./go-npm add <package>

# Add a specific version
./go-npm add <package>@<version>

# Examples
./go-npm add lodash
./go-npm add express@4.18.0
./go-npm add @types/node@18.0.0
```

### remove (alias: `rm`)

Remove a package from `package.json` and delete it from `node_modules`.

```bash
./go-npm remove <package>
./go-npm rm <package>

# Examples
./go-npm remove lodash
./go-npm rm express
```

### uninstall

Uninstall a package from `node_modules` or from global installation.

```bash
# Uninstall from local node_modules
./go-npm uninstall <package>

# Uninstall from global installation
./go-npm uninstall -g <package>
./go-npm uninstall --global <package>

# Examples
./go-npm uninstall lodash
./go-npm uninstall -g typescript
```

**Flags:**
| Flag | Description |
|------|-------------|
| `-g, --global` | Uninstall from global installation |

### run

Run a script defined in `package.json`.

```bash
./go-npm run <script>

# Examples
./go-npm run build
./go-npm run test
./go-npm run start
```

**Features:**
- Executes scripts from the `scripts` section of package.json
- Adds `node_modules/.bin` to PATH automatically
- Sets environment variables: `npm_lifecycle_event`, `npm_package_name`, `npm_package_version`
- Default timeout: 5 minutes per script
- Shows available scripts if the specified script is not found

### list (alias: `ls`)

Display a tree of installed packages and their dependencies.

```bash
# Show top-level dependencies
./go-npm list
./go-npm ls

# Show full dependency tree
./go-npm list --all
./go-npm ls --all
```

**Flags:**
| Flag | Description |
|------|-------------|
| `--all` | Show all dependencies (full tree instead of just top-level) |

**Note:** Requires a lock file (go-npm-lock.json, package-lock.json, or yarn.lock).

### cache

Manage the package cache.

```bash
# Clear all cached packages and manifests
./go-npm cache rm
```



### version

Display the current version.

```bash
./go-npm --version
./go-npm -v
```

|

**Security:** Scripts only run for packages listed in `trustedDependencies` in your package.json:

```json
{
  "trustedDependencies": ["esbuild", "node-sass"]
}
```

Use `--ignore-scripts` to skip all lifecycle scripts.

### Lock File Support

Compatible with multiple lock file formats:

| File | Format |
|------|--------|
| `go-npm-lock.json` | Native format |
| `package-lock.json` | npm format |
| `yarn.lock` | Yarn v1 format |

Lock files store:
- Exact resolved versions
- Resolved download URLs
- Integrity hashes
- Full dependency trees

### Workspace Support

Supports monorepo setups with the `workspaces` field in package.json:

```json
{
  "workspaces": ["packages/*", "apps/*"]
}
```

Or object format:

```json
{
  "workspaces": {
    "packages": ["packages/*", "libs/*"]
  }
}
```

### Binary Linking

Automatically links package executables:

- **Local:** `./node_modules/.bin/`
- **Global:** `~/.config/go-npm/global/bin/`

Supports scoped packages (e.g., `@scope/package`).


## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `GO_NPM_HOME` | Override base config directory | `~/.config/go-npm` |

```bash
# Example: Use custom config directory
GO_NPM_HOME=/custom/path ./go-npm install
```


## Development

### Testing

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -run TestName

# Run specific test with verbose output
go test -v -run TestDownloadManifest
```

### Dependencies

```bash
# Download Go dependencies
go mod download

# Clean up dependencies
go mod tidy
```


## License

MIT
