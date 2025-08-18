#!/bin/bash

# Go Falcon Changelog Management Script
# Usage: ./scripts/changelog.sh [command] [options]

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

# Function to add entry to changelog
add_entry() {
    local change_type="$1"
    local description="$2"
    
    if [[ -z "$change_type" || -z "$description" ]]; then
        print_error "Both change type and description are required"
        exit 1
    fi
    
    # Validate change type
    case "$change_type" in
        added|changed|deprecated|removed|fixed|security)
            ;;
        *)
            print_error "Invalid change type. Must be one of: added, changed, deprecated, removed, fixed, security"
            exit 1
            ;;
    esac
    
    # Capitalize first letter
    change_type="$(tr '[:lower:]' '[:upper:]' <<< ${change_type:0:1})${change_type:1}"
    
    if [[ ! -f "$CHANGELOG_FILE" ]]; then
        print_error "CHANGELOG.md not found. Run 'changelog.sh init' first."
        exit 1
    fi
    
    # Check if Unreleased section exists
    if ! grep -q "\[Unreleased\]" "$CHANGELOG_FILE"; then
        print_warning "No [Unreleased] section found. Adding one."
        add_unreleased_section
    fi
    
    # Find the line number of the change type section
    local section_line
    section_line=$(awk "/## \[Unreleased\]/,/## \[/ { 
        if (/### $change_type/) print NR; 
        if (/## \[/ && !/## \[Unreleased\]/) exit 
    }" "$CHANGELOG_FILE")
    
    if [[ -n "$section_line" ]]; then
        # Section exists, add entry after the header
        local insert_line=$((section_line + 1))
        # Skip empty line if it exists
        if sed -n "${insert_line}p" "$CHANGELOG_FILE" | grep -q '^[[:space:]]*$'; then
            insert_line=$((insert_line + 1))
        fi
        sed -i "${insert_line}i\\- $description" "$CHANGELOG_FILE"
    else
        # Section doesn't exist, create it
        # Find where to insert (after Unreleased header)
        local unreleased_line
        unreleased_line=$(grep -n "\[Unreleased\]" "$CHANGELOG_FILE" | cut -d: -f1)
        
        # Skip to first empty line after Unreleased
        local insert_line=$((unreleased_line + 2))
        
        # Insert new section
        {
            echo ""
            echo "### $change_type"
            echo ""
            echo "- $description"
        } | sed -i "${insert_line}r /dev/stdin" "$CHANGELOG_FILE"
    fi
    
    print_success "Added $change_type entry: $description"
}

# Function to add unreleased section
add_unreleased_section() {
    local header_line
    header_line=$(grep -n "# Changelog" "$CHANGELOG_FILE" | cut -d: -f1)
    
    if [[ -z "$header_line" ]]; then
        print_error "Invalid changelog format"
        exit 1
    fi
    
    # Find insertion point (after header and description)
    local insert_line=$((header_line + 6))
    
    {
        echo ""
        echo "## [Unreleased]"
        echo ""
    } | sed -i "${insert_line}r /dev/stdin" "$CHANGELOG_FILE"
}

# Function to generate changelog from git commits
generate_from_commits() {
    local since_tag="$1"
    local output_file="${2:-/tmp/changelog_commits.md}"
    
    print_info "Generating changelog from commits since $since_tag"
    
    # Get commits since tag
    local commits
    if [[ -n "$since_tag" ]]; then
        commits=$(git log --oneline --pretty=format:"%s" "$since_tag"..HEAD)
    else
        commits=$(git log --oneline --pretty=format:"%s")
    fi
    
    # Parse conventional commits
    echo "## Generated Changelog Entries" > "$output_file"
    echo "" >> "$output_file"
    
    local added_count=0
    local changed_count=0
    local fixed_count=0
    local other_count=0
    
    while IFS= read -r commit; do
        case "$commit" in
            feat:*|feat\(*\):*)
                if [[ $added_count -eq 0 ]]; then
                    echo "### Added" >> "$output_file"
                    echo "" >> "$output_file"
                fi
                echo "- ${commit#feat*: }" >> "$output_file"
                added_count=$((added_count + 1))
                ;;
            fix:*|fix\(*\):*)
                if [[ $fixed_count -eq 0 ]]; then
                    echo "" >> "$output_file"
                    echo "### Fixed" >> "$output_file"
                    echo "" >> "$output_file"
                fi
                echo "- ${commit#fix*: }" >> "$output_file"
                fixed_count=$((fixed_count + 1))
                ;;
            refactor:*|perf:*)
                if [[ $changed_count -eq 0 ]]; then
                    echo "" >> "$output_file"
                    echo "### Changed" >> "$output_file"
                    echo "" >> "$output_file"
                fi
                echo "- ${commit#*: }" >> "$output_file"
                changed_count=$((changed_count + 1))
                ;;
            *)
                if [[ $other_count -eq 0 ]]; then
                    echo "" >> "$output_file"
                    echo "### Other Changes" >> "$output_file"
                    echo "" >> "$output_file"
                fi
                echo "- $commit" >> "$output_file"
                other_count=$((other_count + 1))
                ;;
        esac
    done <<< "$commits"
    
    print_success "Generated changelog saved to $output_file"
    print_info "Review and manually add relevant entries to CHANGELOG.md"
}

# Function to validate changelog format
validate_changelog() {
    if [[ ! -f "$CHANGELOG_FILE" ]]; then
        print_error "CHANGELOG.md not found"
        return 1
    fi
    
    print_info "Validating CHANGELOG.md format..."
    
    local errors=0
    
    # Check for required header
    if ! grep -q "# Changelog" "$CHANGELOG_FILE"; then
        print_error "Missing main header '# Changelog'"
        errors=$((errors + 1))
    fi
    
    # Check for Keep a Changelog reference
    if ! grep -q "Keep a Changelog" "$CHANGELOG_FILE"; then
        print_warning "Missing Keep a Changelog reference"
    fi
    
    # Check for Semantic Versioning reference
    if ! grep -q "Semantic Versioning" "$CHANGELOG_FILE"; then
        print_warning "Missing Semantic Versioning reference"
    fi
    
    # Check version format
    local invalid_versions
    invalid_versions=$(grep -n "^## \[" "$CHANGELOG_FILE" | grep -v "\[Unreleased\]" | grep -v "\[[0-9]\+\.[0-9]\+\.[0-9]\+\]")
    
    if [[ -n "$invalid_versions" ]]; then
        print_error "Invalid version format found:"
        echo "$invalid_versions"
        errors=$((errors + 1))
    fi
    
    if [[ $errors -eq 0 ]]; then
        print_success "Changelog format is valid"
        return 0
    else
        print_error "Found $errors error(s) in changelog"
        return 1
    fi
}

# Function to show latest entries
show_latest() {
    local count="${1:-10}"
    
    if [[ ! -f "$CHANGELOG_FILE" ]]; then
        print_error "CHANGELOG.md not found"
        exit 1
    fi
    
    print_info "Latest $count changelog entries:"
    echo ""
    
    # Extract latest entries
    awk "/^## \[/ { 
        if (++sections <= $count) print; 
        else exit 
    } 
    sections > 0 && sections <= $count { print }
    " "$CHANGELOG_FILE"
}

# Function to show help
show_help() {
    echo "Go Falcon Changelog Management Script"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  add <type> <description>    Add entry to unreleased section"
    echo "  generate [since_tag]        Generate from git commits"
    echo "  validate                    Validate changelog format"
    echo "  latest [count]              Show latest entries (default: 10)"
    echo "  help                        Show this help message"
    echo ""
    echo "Change Types:"
    echo "  added       New features"
    echo "  changed     Changes in existing functionality"
    echo "  deprecated  Soon-to-be removed features"
    echo "  removed     Removed features"
    echo "  fixed       Bug fixes"
    echo "  security    Security fixes"
    echo ""
    echo "Examples:"
    echo "  $0 add added \"New user authentication system\""
    echo "  $0 add fixed \"Resolve memory leak in scheduler\""
    echo "  $0 generate v1.0.0"
    echo "  $0 validate"
    echo "  $0 latest 5"
}

# Main function
main() {
    local command="$1"
    
    case "$command" in
        add)
            add_entry "$2" "$3"
            ;;
        generate)
            generate_from_commits "$2"
            ;;
        validate)
            validate_changelog
            ;;
        latest)
            show_latest "$2"
            ;;
        help|--help|-h)
            show_help
            ;;
        "")
            print_error "Command required"
            show_help
            exit 1
            ;;
        *)
            print_error "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"