#!/bin/bash

# File Cleanup Automation Script (Shell Wrapper)
# This script provides a shell interface to the Go-based cleanup automation

set -e

# Default values
DRY_RUN=false
VERBOSE=false
BACKUP_DIR=""
SHOW_HELP=false

# Function to show usage
show_usage() {
    echo "File Cleanup Automation Script"
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  --dry-run       Show what would be done without making changes"
    echo "  --verbose       Enable verbose output"
    echo "  --backup-dir    Specify backup directory (default: cleanup-backup-TIMESTAMP)"
    echo "  --help          Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 --dry-run --verbose"
    echo "  $0 --backup-dir my-backup"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --backup-dir)
            BACKUP_DIR="$2"
            shift 2
            ;;
        --help)
            SHOW_HELP=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

if [ "$SHOW_HELP" = true ]; then
    show_usage
    exit 0
fi

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH."
    echo "Please install Go from: https://golang.org/dl/"
    exit 1
fi

echo "Using Go: $(go version)"

# Get the script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GO_SCRIPT="$SCRIPT_DIR/cleanup-automation.go"

# Check if the Go script exists
if [ ! -f "$GO_SCRIPT" ]; then
    echo "Error: Go script not found at: $GO_SCRIPT"
    exit 1
fi

# Build arguments for the Go script
GO_ARGS=()

if [ "$DRY_RUN" = true ]; then
    GO_ARGS+=(--dry-run)
fi

if [ "$VERBOSE" = true ]; then
    GO_ARGS+=(--verbose)
fi

if [ -n "$BACKUP_DIR" ]; then
    GO_ARGS+=(--backup-dir "$BACKUP_DIR")
fi

echo "Starting file cleanup automation..."
echo "Script location: $GO_SCRIPT"

# Change to the project root directory (parent of scripts)
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_ROOT"

# Run the Go script
if [ ${#GO_ARGS[@]} -gt 0 ]; then
    go run "$GO_SCRIPT" "${GO_ARGS[@]}"
else
    go run "$GO_SCRIPT"
fi

echo "Cleanup completed successfully!"