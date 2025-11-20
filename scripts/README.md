# Test Scripts

This folder contains test package.json examples for validating the go-npm package manager.

## Test Cases

### 1. express5-enterprise-app
A backend Express 5 application with a moderate number of dependencies.

**Dependencies:**
- express: ^5.0.1
- body-parser: 2.2.0

**Dev Dependencies:**
- @babel/cli: ^7.0.0
- @babel/core: ^7.0.0
- nodemon: ^2.0.0
- prettier: ^2.0.0

**To test:**
```bash
cd scripts/express5-enterprise-app
go run ../../main.go
```

### 2. house4you
A React Native Expo application with many dependencies (25+ production dependencies).

**Key Dependencies:**
- React Native, Expo packages
- Apollo Client, GraphQL
- Various Expo modules (file-system, router, secure-store, etc.)
- Sentry, NativeWind, Zod

**To test:**
```bash
cd scripts/house4you
go run ../../main.go
```

## Usage

Each test case is in its own directory with a `package.json` file. To test the install command:

1. Navigate to the test case directory
2. Run the go-npm package manager
3. Verify that all dependencies are downloaded and extracted to `node_modules/`

## Expected Behavior

The package manager should:
- Parse the package.json file
- Download all dependencies and devDependencies
- Resolve version ranges correctly (^, ~, exact versions)
- Handle nested dependencies recursively
- Extract packages to node_modules/
- Cache manifests and tarballs in ~/.config/go-npm/
