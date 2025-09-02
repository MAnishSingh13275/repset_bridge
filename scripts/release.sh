#!/bin/bash

# Release script for Gym Door Access Bridge
# Automates the complete release process

set -e

# Configuration
PROJECT_NAME="gym-door-bridge"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[RELEASE]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

# Show usage
usage() {
    echo "Usage: $0 [OPTIONS] VERSION"
    echo ""
    echo "Create a new release of $PROJECT_NAME"
    echo ""
    echo "Arguments:"
    echo "  VERSION           Release version (e.g., 1.2.3)"
    echo ""
    echo "Options:"
    echo "  -h, --help        Show this help message"
    echo "  -d, --dry-run     Show what would be done without making changes"
    echo "  -p, --prerelease  Mark as prerelease"
    echo "  -s, --skip-tests  Skip running tests"
    echo "  -c, --skip-clean  Skip cleaning build artifacts"
    echo "  --rollout PERCENT Set rollout percentage (default: 100)"
    echo ""
    echo "Environment variables:"
    echo "  CDN_BASE_URL      Base URL for CDN distribution"
    echo "  GITHUB_TOKEN      GitHub token for creating releases"
    echo "  SLACK_WEBHOOK     Slack webhook for notifications"
    echo ""
    echo "Examples:"
    echo "  $0 1.2.3                    # Create stable release"
    echo "  $0 --prerelease 1.2.3-rc1   # Create prerelease"
    echo "  $0 --dry-run 1.2.3          # Show what would be done"
    echo "  $0 --rollout 50 1.2.3       # Gradual rollout at 50%"
}

# Parse command line arguments
DRY_RUN=false
PRERELEASE=false
SKIP_TESTS=false
SKIP_CLEAN=false
ROLLOUT_PERCENTAGE=100
VERSION=""

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            usage
            exit 0
            ;;
        -d|--dry-run)
            DRY_RUN=true
            shift
            ;;
        -p|--prerelease)
            PRERELEASE=true
            shift
            ;;
        -s|--skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        -c|--skip-clean)
            SKIP_CLEAN=true
            shift
            ;;
        --rollout)
            ROLLOUT_PERCENTAGE="$2"
            shift 2
            ;;
        -*)
            error "Unknown option: $1"
            ;;
        *)
            if [ -z "$VERSION" ]; then
                VERSION="$1"
            else
                error "Multiple versions specified: $VERSION and $1"
            fi
            shift
            ;;
    esac
done

# Validate version
if [ -z "$VERSION" ]; then
    error "Version is required. Use --help for usage information."
fi

# Validate version format
if ! echo "$VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$'; then
    error "Invalid version format: $VERSION. Expected format: X.Y.Z or X.Y.Z-suffix"
fi

# Validate rollout percentage
if ! echo "$ROLLOUT_PERCENTAGE" | grep -qE '^[0-9]+$' || [ "$ROLLOUT_PERCENTAGE" -lt 1 ] || [ "$ROLLOUT_PERCENTAGE" -gt 100 ]; then
    error "Invalid rollout percentage: $ROLLOUT_PERCENTAGE. Must be between 1 and 100."
fi

# Change to project root
cd "$PROJECT_ROOT"

log "Starting release process for $PROJECT_NAME v$VERSION"

if [ "$DRY_RUN" = true ]; then
    warn "DRY RUN MODE - No changes will be made"
fi

# Pre-flight checks
log "Running pre-flight checks..."

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    error "Not in a git repository"
fi

# Check if working directory is clean
if [ "$DRY_RUN" = false ] && ! git diff-index --quiet HEAD --; then
    error "Working directory is not clean. Commit or stash changes first."
fi

# Check if we're on the main branch
CURRENT_BRANCH=$(git branch --show-current)
if [ "$CURRENT_BRANCH" != "main" ] && [ "$CURRENT_BRANCH" != "master" ]; then
    warn "Not on main/master branch (currently on: $CURRENT_BRANCH)"
    read -p "Continue anyway? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        error "Aborted by user"
    fi
fi

# Check if version tag already exists
if git tag -l | grep -q "^v$VERSION$"; then
    error "Version tag v$VERSION already exists"
fi

# Check required tools
log "Checking required tools..."
for tool in go git docker; do
    if ! command -v $tool &> /dev/null; then
        error "$tool is required but not installed"
    fi
done

# Check Go version
GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
REQUIRED_GO_VERSION="1.21"
if ! printf '%s\n%s\n' "$REQUIRED_GO_VERSION" "$GO_VERSION" | sort -V -C; then
    error "Go version $REQUIRED_GO_VERSION or higher is required (found: $GO_VERSION)"
fi

# Run tests
if [ "$SKIP_TESTS" = false ]; then
    log "Running tests..."
    if [ "$DRY_RUN" = false ]; then
        make test
        make test-integration
    else
        info "Would run: make test && make test-integration"
    fi
fi

# Clean build artifacts
if [ "$SKIP_CLEAN" = false ]; then
    log "Cleaning build artifacts..."
    if [ "$DRY_RUN" = false ]; then
        make clean
    else
        info "Would run: make clean"
    fi
fi

# Build all platforms
log "Building for all platforms..."
if [ "$DRY_RUN" = false ]; then
    VERSION="$VERSION" make build-all
else
    info "Would run: VERSION=$VERSION make build-all"
fi

# Generate manifest
log "Generating update manifest..."
if [ "$DRY_RUN" = false ]; then
    export VERSION="$VERSION"
    export ROLLOUT_PERCENTAGE="$ROLLOUT_PERCENTAGE"
    export CRITICAL="false"
    export BREAKING_CHANGES="false"
    export ROLLBACK_ENABLED="true"
    
    # Set CDN base URL
    if [ -n "$CDN_BASE_URL" ]; then
        export CDN_BASE_URL="$CDN_BASE_URL"
    else
        export CDN_BASE_URL="https://github.com/yourdomain/$PROJECT_NAME/releases/download/v$VERSION"
    fi
    
    # Generate release notes from git log
    PREVIOUS_TAG=$(git describe --tags --abbrev=0 HEAD^ 2>/dev/null || echo "")
    if [ -n "$PREVIOUS_TAG" ]; then
        export PREVIOUS_VERSION="${PREVIOUS_TAG#v}"
        
        # Extract features, improvements, and fixes from commit messages
        NEW_FEATURES=$(git log --pretty=format:"%s" "$PREVIOUS_TAG"..HEAD | grep -i "^feat\|^add\|^new" | sed 's/^/"/;s/$/"/' | paste -sd ',' || echo '[]')
        IMPROVEMENTS=$(git log --pretty=format:"%s" "$PREVIOUS_TAG"..HEAD | grep -i "^improve\|^enhance\|^update" | sed 's/^/"/;s/$/"/' | paste -sd ',' || echo '[]')
        BUG_FIXES=$(git log --pretty=format:"%s" "$PREVIOUS_TAG"..HEAD | grep -i "^fix\|^bug" | sed 's/^/"/;s/$/"/' | paste -sd ',' || echo '[]')
        SECURITY_FIXES=$(git log --pretty=format:"%s" "$PREVIOUS_TAG"..HEAD | grep -i "^security\|^sec\|^cve" | sed 's/^/"/;s/$/"/' | paste -sd ',' || echo '[]')
        
        export NEW_FEATURES="[$NEW_FEATURES]"
        export IMPROVEMENTS="[$IMPROVEMENTS]"
        export BUG_FIXES="[$BUG_FIXES]"
        export SECURITY_FIXES="[$SECURITY_FIXES]"
    fi
    
    ./scripts/generate-manifest.sh
else
    info "Would generate manifest with:"
    info "  Version: $VERSION"
    info "  Rollout: $ROLLOUT_PERCENTAGE%"
    info "  CDN URL: ${CDN_BASE_URL:-https://github.com/yourdomain/$PROJECT_NAME/releases/download/v$VERSION}"
fi

# Create git tag
log "Creating git tag..."
if [ "$DRY_RUN" = false ]; then
    git tag -a "v$VERSION" -m "Release v$VERSION"
    info "Created tag: v$VERSION"
else
    info "Would create tag: v$VERSION"
fi

# Build Docker image
log "Building Docker image..."
if [ "$DRY_RUN" = false ]; then
    VERSION="$VERSION" make docker-build
else
    info "Would run: VERSION=$VERSION make docker-build"
fi

# Push to git repository
log "Pushing to git repository..."
if [ "$DRY_RUN" = false ]; then
    git push origin "v$VERSION"
    git push origin HEAD
    info "Pushed tag and commits to origin"
else
    info "Would push tag v$VERSION and commits to origin"
fi

# Create GitHub release (if GITHUB_TOKEN is available)
if [ -n "$GITHUB_TOKEN" ]; then
    log "Creating GitHub release..."
    if [ "$DRY_RUN" = false ]; then
        # Generate release notes
        RELEASE_NOTES_FILE=$(mktemp)
        if [ -n "$PREVIOUS_TAG" ]; then
            echo "## Changes since $PREVIOUS_TAG" > "$RELEASE_NOTES_FILE"
            echo "" >> "$RELEASE_NOTES_FILE"
            git log --pretty=format:"- %s" "$PREVIOUS_TAG"..HEAD >> "$RELEASE_NOTES_FILE"
        else
            echo "## Initial Release" > "$RELEASE_NOTES_FILE"
            echo "" >> "$RELEASE_NOTES_FILE"
            echo "First release of $PROJECT_NAME v$VERSION" >> "$RELEASE_NOTES_FILE"
        fi
        
        # Create release
        PRERELEASE_FLAG=""
        if [ "$PRERELEASE" = true ]; then
            PRERELEASE_FLAG="--prerelease"
        fi
        
        gh release create "v$VERSION" \
            $PRERELEASE_FLAG \
            --title "Release v$VERSION" \
            --notes-file "$RELEASE_NOTES_FILE" \
            dist/* build/*
        
        rm "$RELEASE_NOTES_FILE"
        info "Created GitHub release: v$VERSION"
    else
        info "Would create GitHub release with artifacts from dist/ and build/"
    fi
else
    warn "GITHUB_TOKEN not set, skipping GitHub release creation"
fi

# Push Docker image (if logged in)
if docker info > /dev/null 2>&1; then
    log "Pushing Docker image..."
    if [ "$DRY_RUN" = false ]; then
        VERSION="$VERSION" make docker-push
    else
        info "Would run: VERSION=$VERSION make docker-push"
    fi
else
    warn "Not logged into Docker registry, skipping Docker push"
fi

# Send notification
if [ -n "$SLACK_WEBHOOK" ]; then
    log "Sending Slack notification..."
    if [ "$DRY_RUN" = false ]; then
        SLACK_MESSAGE="🚀 Released $PROJECT_NAME v$VERSION"
        if [ "$PRERELEASE" = true ]; then
            SLACK_MESSAGE="$SLACK_MESSAGE (prerelease)"
        fi
        if [ "$ROLLOUT_PERCENTAGE" -lt 100 ]; then
            SLACK_MESSAGE="$SLACK_MESSAGE with $ROLLOUT_PERCENTAGE% rollout"
        fi
        
        curl -X POST -H 'Content-type: application/json' \
            --data "{\"text\":\"$SLACK_MESSAGE\"}" \
            "$SLACK_WEBHOOK"
    else
        info "Would send Slack notification about release"
    fi
fi

# Summary
log "Release process completed successfully!"
echo ""
info "Release Summary:"
info "  Version: $VERSION"
info "  Prerelease: $PRERELEASE"
info "  Rollout: $ROLLOUT_PERCENTAGE%"
info "  Git tag: v$VERSION"
info "  Artifacts: dist/ and build/"
if [ -n "$GITHUB_TOKEN" ]; then
    info "  GitHub release: https://github.com/yourdomain/$PROJECT_NAME/releases/tag/v$VERSION"
fi
echo ""

if [ "$DRY_RUN" = false ]; then
    log "Next steps:"
    info "1. Monitor the rollout in the admin portal"
    info "2. Watch for any issues in monitoring dashboards"
    info "3. Increase rollout percentage if needed"
    info "4. Update documentation if required"
else
    warn "This was a dry run - no changes were made"
    info "Run without --dry-run to perform the actual release"
fi