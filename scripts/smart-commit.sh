#!/bin/bash

# Smart Commit Script with Changelog Integration
# Usage: ./scripts/smart-commit.sh "commit message" [--no-changelog]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

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

show_help() {
    echo "Smart Commit Script with Changelog Integration"
    echo ""
    echo "Usage: $0 \"commit message\" [options]"
    echo ""
    echo "Options:"
    echo "  --no-changelog    Skip changelog update"
    echo "  --type TYPE       Specify changelog type directly (added|changed|fixed|security|deprecated|removed)"
    echo "  --help           Show this help"
    echo ""
    echo "Examples:"
    echo "  $0 \"feat: add user authentication\""
    echo "  $0 \"fix: resolve memory leak\" --type fixed"
    echo "  $0 \"chore: update dependencies\" --no-changelog"
}

# Parse arguments
commit_message=""
skip_changelog=false
change_type=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --no-changelog)
            skip_changelog=true
            shift
            ;;
        --type)
            change_type="$2"
            shift 2
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            if [[ -z "$commit_message" ]]; then
                commit_message="$1"
            else
                print_error "Multiple commit messages not supported"
                exit 1
            fi
            shift
            ;;
    esac
done

if [[ -z "$commit_message" ]]; then
    print_error "Commit message required"
    show_help
    exit 1
fi

# Check if we're in a git repo
if ! git rev-parse --git-dir >/dev/null 2>&1; then
    print_error "Not in a git repository"
    exit 1
fi

# Check if there are changes to commit
if [[ -z $(git status --porcelain) ]]; then
    print_warning "No changes to commit"
    exit 0
fi

cd "$PROJECT_ROOT"

# Function to get changelog type from user
get_changelog_type() {
    if [[ -n "$change_type" ]]; then
        case "$change_type" in
            added|changed|fixed|security|deprecated|removed)
                echo "$change_type"
                return 0
                ;;
            *)
                print_error "Invalid change type: $change_type"
                print_error "Valid types: added, changed, fixed, security, deprecated, removed"
                exit 1
                ;;
        esac
    fi
    
    echo ""
    echo -e "${CYAN}What type of change is this for the changelog?${NC}"
    echo "1. added     - New features or functionality"
    echo "2. changed   - Changes in existing functionality"
    echo "3. fixed     - Bug fixes and error corrections"
    echo "4. security  - Security improvements"
    echo "5. deprecated - Soon-to-be removed features"
    echo "6. removed   - Features that were removed"
    echo "7. skip      - Don't update changelog"
    echo ""
    
    while true; do
        echo -n "Your choice (1-7): "
        read -r choice
        
        case "$choice" in
            1) echo "added"; return 0 ;;
            2) echo "changed"; return 0 ;;
            3) echo "fixed"; return 0 ;;
            4) echo "security"; return 0 ;;
            5) echo "deprecated"; return 0 ;;
            6) echo "removed"; return 0 ;;
            7) echo "skip"; return 0 ;;
            *) echo "Invalid choice. Please enter 1-7." ;;
        esac
    done
}

# Function to extract description from commit message
extract_description() {
    local msg="$1"
    
    # Remove conventional commit prefix (feat:, fix:, etc.)
    msg=$(echo "$msg" | sed -E 's/^(feat|fix|docs|style|refactor|test|chore|perf|ci|build)(\([^)]*\))?:\s*//')
    
    # Capitalize first letter
    msg="$(tr '[:lower:]' '[:upper:]' <<< ${msg:0:1})${msg:1}"
    
    echo "$msg"
}

# Main execution
print_info "Preparing commit: $commit_message"

# Handle changelog
if [[ "$skip_changelog" != true ]]; then
    changelog_type=$(get_changelog_type)
    
    if [[ "$changelog_type" != "skip" ]]; then
        if [[ -f "CHANGELOG.md" ]]; then
            description=$(extract_description "$commit_message")
            
            print_info "Adding changelog entry: $changelog_type - $description"
            
            if ./scripts/changelog.sh add "$changelog_type" "$description"; then
                print_success "Changelog updated"
                
                # Stage the changelog
                git add CHANGELOG.md
            else
                print_error "Failed to update changelog"
                echo -n "Continue with commit anyway? (y/N) "
                read -r continue_commit
                if [[ "$continue_commit" != "y" && "$continue_commit" != "Y" ]]; then
                    print_info "Commit cancelled"
                    exit 1
                fi
            fi
        else
            print_warning "CHANGELOG.md not found, skipping changelog update"
        fi
    fi
fi

# Show what will be committed
echo ""
print_info "Files to be committed:"
git status --porcelain

echo ""
print_info "Creating commit..."

# Create the commit
if git commit -m "$commit_message"; then
    print_success "Commit created successfully"
    
    # Show the commit
    echo ""
    git log -1 --oneline
    
    echo ""
    print_info "Next steps:"
    echo "  - Push changes: git push origin $(git rev-parse --abbrev-ref HEAD)"
    if [[ "$skip_changelog" != true && "$changelog_type" != "skip" ]]; then
        echo "  - Changelog updated automatically"
    fi
else
    print_error "Commit failed"
    exit 1
fi