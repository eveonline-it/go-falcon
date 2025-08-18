#!/bin/bash

# Go Falcon Release Script
# Usage: ./scripts/release.sh [patch|minor|major]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CHANGELOG_FILE="$PROJECT_ROOT/CHANGELOG.md"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if we're in a git repository
check_git_repo() {
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        print_error "Not in a git repository"
        exit 1
    fi
}

# Function to check if working directory is clean
check_clean_working_dir() {
    if [[ -n $(git status --porcelain) ]]; then
        print_error "Working directory is not clean. Please commit or stash your changes."
        git status --short
        exit 1
    fi
}

# Function to get current version from git tags
get_current_version() {
    # Try to get the latest tag
    local latest_tag
    latest_tag=$(git describe --tags --abbrev=0 2>/dev/null || echo "")
    
    if [[ -z "$latest_tag" ]]; then
        echo "0.0.0"
    else
        # Remove 'v' prefix if present
        echo "${latest_tag#v}"
    fi
}

# Function to bump version
bump_version() {
    local current_version="$1"
    local bump_type="$2"
    
    # Split version into major.minor.patch
    local major minor patch
    IFS='.' read -r major minor patch <<< "$current_version"
    
    case "$bump_type" in
        "patch")
            patch=$((patch + 1))
            ;;
        "minor")
            minor=$((minor + 1))
            patch=0
            ;;
        "major")
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        *)
            print_error "Invalid bump type: $bump_type"
            exit 1
            ;;
    esac
    
    echo "$major.$minor.$patch"
}

# Function to update changelog
update_changelog() {
    local new_version="$1"
    local release_date
    release_date=$(date +%Y-%m-%d)
    
    if [[ ! -f "$CHANGELOG_FILE" ]]; then
        print_warning "CHANGELOG.md not found, skipping changelog update"
        return
    fi
    
    # Replace [Unreleased] with [version] - date
    if grep -q "\[Unreleased\]" "$CHANGELOG_FILE"; then
        print_info "Updating changelog for version $new_version"
        
        # Create backup
        cp "$CHANGELOG_FILE" "$CHANGELOG_FILE.bak"
        
        # Update the changelog
        sed -i.tmp "s/## \[Unreleased\]/## [$new_version] - $release_date/" "$CHANGELOG_FILE"
        
        # Add new Unreleased section at the top
        awk '
        /^# Changelog/ { print; getline; print; getline; print; getline; print; print "## [Unreleased]\n"; print; next }
        { print }
        ' "$CHANGELOG_FILE" > "$CHANGELOG_FILE.new"
        mv "$CHANGELOG_FILE.new" "$CHANGELOG_FILE"
        
        # Remove temp files
        rm -f "$CHANGELOG_FILE.tmp" "$CHANGELOG_FILE.bak"
        
        print_success "Changelog updated"
    else
        print_warning "No [Unreleased] section found in changelog"
    fi
}

# Function to create git tag
create_git_tag() {
    local version="$1"
    local tag_name="v$version"
    
    print_info "Creating git tag: $tag_name"
    
    # Add and commit changelog changes
    git add "$CHANGELOG_FILE"
    git commit -m "chore: release version $version

🤖 Generated with [Claude Code](https://claude.ai/code)

Co-Authored-By: Claude <noreply@anthropic.com>"
    
    # Create annotated tag
    git tag -a "$tag_name" -m "Release $version

$(extract_release_notes "$version")"
    
    print_success "Git tag created: $tag_name"
}

# Function to extract release notes from changelog
extract_release_notes() {
    local version="$1"
    
    if [[ ! -f "$CHANGELOG_FILE" ]]; then
        echo "Release $version"
        return
    fi
    
    # Extract the section for this version
    awk "/## \[$version\]/, /## \[/ { 
        if (/## \[$version\]/) next
        if (/## \[/ && !/## \[$version\]/) exit
        print 
    }" "$CHANGELOG_FILE" | sed '/^$/d'
}

# Function to build the project
build_project() {
    print_info "Building project..."
    
    cd "$PROJECT_ROOT"
    
    # Clean and build
    go clean ./...
    go mod tidy
    go mod verify
    
    # Run tests
    go test ./...
    
    # Build main application
    go build -o falcon ./cmd/gateway
    
    print_success "Build completed successfully"
}

# Function to show help
show_help() {
    echo "Go Falcon Release Script"
    echo ""
    echo "Usage: $0 [patch|minor|major] [options]"
    echo ""
    echo "Arguments:"
    echo "  patch    Bump patch version (x.y.Z)"
    echo "  minor    Bump minor version (x.Y.0)"
    echo "  major    Bump major version (X.0.0)"
    echo ""
    echo "Options:"
    echo "  --dry-run    Show what would be done without making changes"
    echo "  --no-build   Skip building the project"
    echo "  --no-tag     Skip creating git tag"
    echo "  --help       Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 patch                    # Release new patch version"
    echo "  $0 minor --dry-run         # Preview minor version bump"
    echo "  $0 major --no-build        # Major release without building"
}

# Main function
main() {
    local bump_type=""
    local dry_run=false
    local no_build=false
    local no_tag=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            patch|minor|major)
                bump_type="$1"
                shift
                ;;
            --dry-run)
                dry_run=true
                shift
                ;;
            --no-build)
                no_build=true
                shift
                ;;
            --no-tag)
                no_tag=true
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    if [[ -z "$bump_type" ]]; then
        print_error "Bump type required (patch|minor|major)"
        show_help
        exit 1
    fi
    
    print_info "Starting release process..."
    print_info "Bump type: $bump_type"
    
    # Pre-flight checks
    check_git_repo
    check_clean_working_dir
    
    # Get current version
    current_version=$(get_current_version)
    print_info "Current version: $current_version"
    
    # Calculate new version
    new_version=$(bump_version "$current_version" "$bump_type")
    print_info "New version: $new_version"
    
    if [[ "$dry_run" == true ]]; then
        print_warning "DRY RUN - No changes will be made"
        print_info "Would bump version from $current_version to $new_version"
        print_info "Would update CHANGELOG.md"
        print_info "Would create git tag: v$new_version"
        exit 0
    fi
    
    # Confirm release
    echo -n "Proceed with release $new_version? (y/N) "
    read -r confirm
    if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
        print_info "Release cancelled"
        exit 0
    fi
    
    # Build project
    if [[ "$no_build" != true ]]; then
        build_project
    fi
    
    # Update changelog
    update_changelog "$new_version"
    
    # Create git tag
    if [[ "$no_tag" != true ]]; then
        create_git_tag "$new_version"
    fi
    
    print_success "Release $new_version completed successfully!"
    print_info "Next steps:"
    print_info "  - Push commits: git push origin main"
    print_info "  - Push tags: git push origin v$new_version"
    print_info "  - Create GitHub release from tag"
}

# Run main function with all arguments
main "$@"