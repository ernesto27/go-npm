#!/bin/bash

# Test script for npm-packager against popular GitHub repositories
# This script clones popular repos, runs npm-packager, and logs results

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_REPOS_DIR="$SCRIPT_DIR/repos"
LOG_FILE="$SCRIPT_DIR/test-results.log"
NPM_PACKAGER_BIN="$(cd "$SCRIPT_DIR/.." && pwd)/npm-packager"

# Create repos directory if it doesn't exist
mkdir -p "$TEST_REPOS_DIR"

go install

# Popular GitHub repositories to test (repo_name:github_url)
# Organized by category for comprehensive testing across different environments
declare -a REPOS=(
    # Simple utilities
    "lodash:https://github.com/lodash/lodash.git"
    "axios:https://github.com/axios/axios.git"
    "chalk:https://github.com/chalk/chalk.git"
    "commander:https://github.com/tj/commander.js.git"

    # Node.js frameworks (backend)
    "express:https://github.com/expressjs/express.git"
    "fastify:https://github.com/fastify/fastify.git"
    "nestjs:https://github.com/nestjs/nest.git"
    "koa:https://github.com/koajs/koa.git"

    # Full-stack/SSR frameworks
    "next.js:https://github.com/vercel/next.js.git"
    "nuxt:https://github.com/nuxt/nuxt.git"
    "gatsby:https://github.com/gatsbyjs/gatsby.git"

    # React Native projects
    "react-native:https://github.com/facebook/react-native.git"
    "expo:https://github.com/expo/expo.git"
    "ignite:https://github.com/infinitered/ignite.git"

    # Electron applications
    "electron:https://github.com/electron/electron.git"
    "hyper:https://github.com/vercel/hyper.git"
    "etcher:https://github.com/balena-io/etcher.git"

    # Build tools & bundlers (complex)
    "webpack:https://github.com/webpack/webpack.git"
    "vite:https://github.com/vitejs/vite.git"
    "rollup:https://github.com/rollup/rollup.git"
    "parcel:https://github.com/parcel-bundler/parcel.git"

    # Testing frameworks
    "jest:https://github.com/jestjs/jest.git"
    "mocha:https://github.com/mochajs/mocha.git"
    "playwright:https://github.com/microsoft/playwright.git"
    "cypress:https://github.com/cypress-io/cypress.git"

    # Compilers & transpilers
    "babel:https://github.com/babel/babel.git"
    "typescript:https://github.com/microsoft/TypeScript.git"
    "swc:https://github.com/swc-project/swc.git"

    # Code quality tools
    "eslint:https://github.com/eslint/eslint.git"
    "prettier:https://github.com/prettier/prettier.git"

    # Frontend frameworks
    "vue:https://github.com/vuejs/core.git"
    "svelte:https://github.com/sveltejs/svelte.git"
    "solid:https://github.com/solidjs/solid.git"

    # Real-world applications (complex)
    "strapi:https://github.com/strapi/strapi.git"
    "ghost:https://github.com/TryGhost/Ghost.git"
    "rocket.chat:https://github.com/RocketChat/Rocket.Chat.git"

    # Developer tools
    "npm:https://github.com/npm/cli.git"
    "yarn:https://github.com/yarnpkg/berry.git"
    "pnpm:https://github.com/pnpm/pnpm.git"
)

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

# Function to log messages to both console and file
log_message() {
    local message="$1"
    echo "$message" | tee -a "$LOG_FILE"
}

# Function to clone repository if it doesn't exist
clone_repo_if_needed() {
    local repo_name=$1
    local repo_url=$2
    local repo_path="$TEST_REPOS_DIR/$repo_name"

    if [ -d "$repo_path" ]; then
        print_status "$YELLOW" "Repository '$repo_name' already exists, skipping clone"
        log_message "[INFO] Repository '$repo_name' already exists at $repo_path"
    else
        print_status "$BLUE" "Cloning repository '$repo_name'..."
        log_message "[INFO] Cloning $repo_url to $repo_path"
        git clone --depth 1 "$repo_url" "$repo_path" &>> "$LOG_FILE"
        print_status "$GREEN" "Successfully cloned '$repo_name'"
    fi
}

# Function to clean up before test run
cleanup_repo() {
    local repo_path=$1
    local repo_name=$2

    print_status "$YELLOW" "Cleaning '$repo_name'..."

    # Remove node_modules if it exists
    if [ -d "$repo_path/node_modules" ]; then
        log_message "[INFO] Removing node_modules from $repo_name"
        rm -rf "$repo_path/node_modules"
    fi

    # Remove go-package-lock if it exists
    if [ -f "$repo_path/go-package-lock" ]; then
        log_message "[INFO] Removing go-package-lock from $repo_name"
        rm -f "$repo_path/go-package-lock"
    fi

    print_status "$GREEN" "Cleaned '$repo_name'"
}

# Function to test repository with npm-packager
test_repo() {
    local repo_path=$1
    local repo_name=$2

    print_status "$BLUE" "Testing '$repo_name' with npm-packager..."
    log_message ""
    log_message "=========================================="
    log_message "Testing: $repo_name"
    log_message "Path: $repo_path"
    log_message "Timestamp: $(date '+%Y-%m-%d %H:%M:%S')"
    log_message "=========================================="

    # Check if package.json exists
    if [ ! -f "$repo_path/package.json" ]; then
        print_status "$RED" "No package.json found in '$repo_name', skipping"
        log_message "[ERROR] No package.json found in $repo_path"
        return 1
    fi

    # Run npm-packager
    cd "$repo_path"
    local start_time=$(date +%s)
    local temp_output=$(mktemp)

    log_message "[INFO] Running npm-packager for repository: $repo_name"

    if "$NPM_PACKAGER_BIN" i > "$temp_output" 2>&1; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        print_status "$GREEN" "Successfully installed dependencies for '$repo_name' (${duration}s)"
        log_message "[SUCCESS] npm-packager completed successfully in ${duration}s"

        # Append output to log
        cat "$temp_output" >> "$LOG_FILE"
        rm -f "$temp_output"

        # Count installed packages
        if [ -d "node_modules" ]; then
            local pkg_count=$(find node_modules -maxdepth 1 -type d | wc -l)
            log_message "[INFO] Installed $pkg_count packages in node_modules"
        fi
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        print_status "$RED" "Failed to install dependencies for '$repo_name' (${duration}s)"
        log_message "[ERROR] Repository: $repo_name - npm-packager failed after ${duration}s"
        log_message "[ERROR] Error output for repository '$repo_name':"

        # Add error output with repository context
        while IFS= read -r line; do
            log_message "[ERROR][$repo_name] $line"
        done < "$temp_output"

        rm -f "$temp_output"
        return 1
    fi
}

# Main execution
main() {
    print_status "$BLUE" "=========================================="
    print_status "$BLUE" "  NPM-PACKAGER REPOSITORY TEST SUITE"
    print_status "$BLUE" "=========================================="
    echo ""

    # Remove old log file and create fresh one
    rm -f "$LOG_FILE"
    log_message "NPM-Packager Repository Test Suite"
    log_message "Started: $(date '+%Y-%m-%d %H:%M:%S')"
    log_message ""

    # Check if npm-packager binary exists
    if [ ! -f "$NPM_PACKAGER_BIN" ]; then
        print_status "$RED" "npm-packager binary not found at $NPM_PACKAGER_BIN"
        print_status "$YELLOW" "Building npm-packager..."
        cd "$(dirname "$NPM_PACKAGER_BIN")"
        go build -o npm-packager main.go
        print_status "$GREEN" "Built npm-packager"
    fi

    # Track test results
    local total_tests=0
    local successful_tests=0
    local failed_tests=0

    # Process each repository
    for repo_entry in "${REPOS[@]}"; do
        IFS=':' read -r repo_name repo_url <<< "$repo_entry"
        total_tests=$((total_tests + 1))

        echo ""
        print_status "$BLUE" "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        print_status "$BLUE" "Repository: $repo_name"
        print_status "$BLUE" "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

        # Clone if needed
        clone_repo_if_needed "$repo_name" "$repo_url"

        # Clean up
        cleanup_repo "$TEST_REPOS_DIR/$repo_name" "$repo_name"

        # Test
        if test_repo "$TEST_REPOS_DIR/$repo_name" "$repo_name"; then
            successful_tests=$((successful_tests + 1))
        else
            failed_tests=$((failed_tests + 1))
        fi
    done

    # Print summary
    echo ""
    print_status "$BLUE" "=========================================="
    print_status "$BLUE" "  TEST SUMMARY"
    print_status "$BLUE" "=========================================="
    log_message ""
    log_message "=========================================="
    log_message "TEST SUMMARY"
    log_message "=========================================="
    log_message "Total repositories tested: $total_tests"
    log_message "Successful: $successful_tests"
    log_message "Failed: $failed_tests"
    log_message "Completed: $(date '+%Y-%m-%d %H:%M:%S')"
    log_message "=========================================="

    print_status "$GREEN" "Successful: $successful_tests"
    if [ $failed_tests -gt 0 ]; then
        print_status "$RED" "Failed: $failed_tests"
    fi
    print_status "$BLUE" "Full logs saved to: $LOG_FILE"
    echo ""

    # Return exit code based on failures
    if [ $failed_tests -gt 0 ]; then
        return 1
    fi
    return 0
}

# Run main function
main "$@"
