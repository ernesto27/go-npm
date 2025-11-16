# NPM Package Manager (Go)

A high-performance npm package manager written in Go that downloads and installs npm packages with full dependency resolution.

## Local Development

### Prerequisites

- Go 1.25 or higher
- A `package.json` file in your working directory (for most commands)

### Build

```bash
# Build the binary
go build -o go-npm

# After building, you can use the binary
./go-npm <command>
```

## Commands

### Install

Install packages from `package.json` or install a specific package.

```bash
# Install all dependencies from package.json
./go-npm install
./go-npm i  # Short alias

# Install only production dependencies (skip devDependencies)
./go-npm install --production

# Install a specific package globally
./go-npm install -g <package>
./go-npm install --global <package>[@version]

# Examples
./go-npm install -g lodash
./go-npm install -g express@4.18.0
```

**Flags:**
- `-g, --global` - Install package globally to `~/.config/go-npm/global/`
- `--production` - Install only production dependencies, skip devDependencies

### Add

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

### Remove

Remove a package from `package.json` and delete it from `node_modules`.

```bash
./go-npm remove <package>
./go-npm rm <package>  # Short alias

# Examples
./go-npm remove lodash
./go-npm rm express
```

### Uninstall

Uninstall a package from `node_modules` or from global installation.

```bash
# Uninstall from local node_modules
./go-npm uninstall <package>

# Uninstall from global installation
./go-npm uninstall -g <package>
./go-npm uninstall --global <package>

# Examples
./go-npm uninstall lodash
./go-npm uninstall -g express
```

**Flags:**
- `-g, --global` - Uninstall from global installation

### Cache

Manage the package cache.

```bash
# Clear all cached packages and manifests
./go-npm cache rm
```

## Development

### Testing

```bash
# Run all tests
go test

# Run tests with verbose output
go test -v

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

### Example package.json

```json
{
  "name": "my-project",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.18.0",
    "lodash": "~4.17.21",
    "@types/node": "^18.0.0"
  },
  "devDependencies": {
    "jest": "^29.5.0"
  }
}
```

## License

MIT
