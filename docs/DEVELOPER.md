# Developer Guide - Version Management & Release Process

This guide explains how to manage versions, maintain the changelog, and create releases for the Go Falcon project.

## 📋 Table of Contents

- [Overview](#overview)
- [Version Management](#version-management)
- [Changelog Management](#changelog-management)
- [Release Process](#release-process)
- [Development Workflow](#development-workflow)
- [CLI Tools](#cli-tools)
- [Build Integration](#build-integration)
- [Troubleshooting](#troubleshooting)

## 🚀 Overview

Go Falcon uses **Semantic Versioning** (SemVer) and follows the **Keep a Changelog** format for version management and release documentation. The system provides automated tools for:

- Version bumping and validation
- Changelog maintenance
- Automated releases with git tagging
- Build-time version injection

## 📦 Version Management

### Semantic Versioning Format

We use **MAJOR.MINOR.PATCH** format:
- **MAJOR** (X.0.0): Breaking changes, major features
- **MINOR** (0.X.0): New features, backwards compatible
- **PATCH** (0.0.X): Bug fixes, security patches

### Current Version Information

```bash
# Show detailed version info
go run cmd/version/main.go info

# Show current version only
go run cmd/version/main.go current

# Show as JSON
go run cmd/version/main.go json
```

### Version Calculation

```bash
# Calculate next version (doesn't modify anything)
go run cmd/version/main.go bump patch    # 1.0.0 → 1.0.1
go run cmd/version/main.go bump minor    # 1.0.0 → 1.1.0
go run cmd/version/main.go bump major    # 1.0.0 → 2.0.0

# Validate version format
go run cmd/version/main.go validate 1.2.3
go run cmd/version/main.go validate v2.0.0-beta.1

# Compare versions
go run cmd/version/main.go compare 1.0.0 2.0.0
```

## 📝 Changelog Management

### Adding Entries During Development

As you work on features and fixes, add entries to the changelog:

```bash
# Add different types of changes
./scripts/changelog.sh add added "New user authentication system"
./scripts/changelog.sh add changed "Improved API response format"
./scripts/changelog.sh add fixed "Memory leak in scheduler service"
./scripts/changelog.sh add security "Updated JWT token validation"
./scripts/changelog.sh add deprecated "Legacy API endpoints marked for removal"
./scripts/changelog.sh add removed "Obsolete configuration options"
```

### Change Types

- **`added`** - New features, functionality, or capabilities
- **`changed`** - Changes in existing functionality or behavior
- **`deprecated`** - Soon-to-be removed features (still functional)
- **`removed`** - Features that have been completely removed
- **`fixed`** - Bug fixes and error corrections
- **`security`** - Security improvements and vulnerability fixes

### Generating from Git Commits

For automated changelog generation from conventional commits:

```bash
# Generate since last tag
./scripts/changelog.sh generate

# Generate since specific tag
./scripts/changelog.sh generate v1.0.0

# Review generated file
cat /tmp/changelog_commits.md
```

### Viewing Changelog

```bash
# Show latest 5 releases
./scripts/changelog.sh latest 5

# Show latest 10 releases
./scripts/changelog.sh latest 10

# Validate changelog format
./scripts/changelog.sh validate
```

## 🎯 Release Process

### Automated Release (Recommended)

The release script handles everything automatically:

```bash
# Create patch release (bug fixes)
./scripts/release.sh patch

# Create minor release (new features)
./scripts/release.sh minor

# Create major release (breaking changes)
./scripts/release.sh major
```

**What the script does:**
1. ✅ Validates git repository state (clean working directory)
2. ✅ Calculates new version number
3. ✅ Runs tests and builds the project
4. ✅ Updates CHANGELOG.md with release date
5. ✅ Creates git commit with changes
6. ✅ Creates annotated git tag
7. ✅ Shows next steps for pushing

### Release Options

```bash
# Preview what would happen (safe)
./scripts/release.sh minor --dry-run

# Skip building (faster, for docs-only releases)
./scripts/release.sh patch --no-build

# Skip git tagging (manual tagging later)
./scripts/release.sh minor --no-tag

# Get help
./scripts/release.sh --help
```

### Manual Release Process

If you prefer manual control:

1. **Update version information**
   ```bash
   # Calculate next version
   NEW_VERSION=$(go run cmd/version/main.go bump patch | grep "New:" | cut -d' ' -f2)
   echo "Releasing version: $NEW_VERSION"
   ```

2. **Release changelog**
   ```bash
   go run cmd/version/main.go changelog release $NEW_VERSION
   ```

3. **Build and test**
   ```bash
   go test ./...
   go build -o falcon ./cmd/gateway
   ```

4. **Create git tag**
   ```bash
   git add CHANGELOG.md
   git commit -m "chore: release version $NEW_VERSION"
   git tag -a "v$NEW_VERSION" -m "Release $NEW_VERSION"
   ```

### Post-Release Steps

After running the release script:

```bash
# Push commits and tags to remote
git push origin main
git push origin v1.2.3

# Create GitHub release (optional)
gh release create v1.2.3 --notes-from-tag
```

## 🔄 Development Workflow

### Daily Development

1. **Work on features/fixes**
   ```bash
   git checkout -b feature/new-authentication
   # ... make changes ...
   git commit -m "feat: add OAuth2 authentication support"
   ```

2. **Add changelog entries as you go**
   ```bash
   ./scripts/changelog.sh add added "OAuth2 authentication support"
   ```

3. **Create pull request**
   ```bash
   git push origin feature/new-authentication
   # Create PR via GitHub UI or gh CLI
   ```

### Pre-Release Checklist

Before creating a release:

- [ ] All tests pass: `go test ./...`
- [ ] Code builds successfully: `go build ./...`
- [ ] Changelog has unreleased entries
- [ ] Working directory is clean
- [ ] On main branch with latest changes

### Hotfix Process

For urgent fixes that need immediate release:

```bash
# Create hotfix branch
git checkout -b hotfix/critical-security-fix

# Make minimal changes
# ... fix critical issue ...

# Add changelog entry
./scripts/changelog.sh add security "Fixed critical authentication bypass"

# Commit changes
git commit -m "fix: resolve critical authentication bypass vulnerability"

# Create immediate patch release
./scripts/release.sh patch

# Push everything
git push origin main
git push origin v1.2.4
```

## 🛠️ CLI Tools

### Version CLI (`cmd/version/main.go`)

Complete version management from command line:

```bash
# Build the CLI tool
go build -o version-tool ./cmd/version

# Use the tool
./version-tool info
./version-tool bump minor
./version-tool changelog add fixed "Bug in user service"
./version-tool changelog latest 3
```

### Available Commands

| Command | Purpose | Example |
|---------|---------|---------|
| `info` | Show build information | `./version-tool info` |
| `current` | Show current version | `./version-tool current` |
| `json` | Version info as JSON | `./version-tool json` |
| `bump` | Calculate new version | `./version-tool bump patch` |
| `compare` | Compare versions | `./version-tool compare 1.0.0 2.0.0` |
| `validate` | Check version format | `./version-tool validate 1.2.3` |
| `changelog` | Manage changelog | `./version-tool changelog add fixed "Bug"` |

### Changelog CLI Commands

```bash
# Add entries
./version-tool changelog add added "New feature"
./version-tool changelog add fixed "Bug fix"

# Release version
./version-tool changelog release 1.2.3

# View latest entries
./version-tool changelog latest 5
```

## 🏗️ Build Integration

### Version Injection at Build Time

The version information is injected during build using Go's `-ldflags`:

```bash
# Manual build with version info
VERSION="1.2.3"
GIT_COMMIT=$(git rev-parse HEAD)
GIT_BRANCH=$(git rev-parse --abbrev-ref HEAD)
BUILD_DATE=$(date -u '+%Y-%m-%d_%H:%M:%S')
BUILD_USER=$(whoami)

go build \
  -ldflags "-X go-falcon/pkg/version.Version=$VERSION \
           -X go-falcon/pkg/version.GitCommit=$GIT_COMMIT \
           -X go-falcon/pkg/version.GitBranch=$GIT_BRANCH \
           -X go-falcon/pkg/version.BuildDate=$BUILD_DATE \
           -X go-falcon/pkg/version.BuildUser=$BUILD_USER" \
  -o falcon ./cmd/gateway
```

### Docker Build Integration

For Docker builds, version information is automatically injected:

```dockerfile
# In Dockerfile
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE=unknown

RUN go build \
    -ldflags "-X go-falcon/pkg/version.Version=${VERSION} \
             -X go-falcon/pkg/version.GitCommit=${GIT_COMMIT} \
             -X go-falcon/pkg/version.BuildDate=${BUILD_DATE}" \
    -o falcon ./cmd/gateway
```

```bash
# Build with version info
docker build \
  --build-arg VERSION=$(git describe --tags --abbrev=0) \
  --build-arg GIT_COMMIT=$(git rev-parse HEAD) \
  --build-arg BUILD_DATE=$(date -u '+%Y-%m-%d_%H:%M:%S') \
  -t go-falcon:latest .
```

### CI/CD Integration

Example GitHub Actions workflow:

```yaml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v3
        with:
          go-version: 1.21
      
      - name: Get version info
        id: version
        run: |
          echo "VERSION=${GITHUB_REF#refs/tags/}" >> $GITHUB_OUTPUT
          echo "GIT_COMMIT=${GITHUB_SHA}" >> $GITHUB_OUTPUT
          echo "BUILD_DATE=$(date -u '+%Y-%m-%d_%H:%M:%S')" >> $GITHUB_OUTPUT
      
      - name: Build
        run: |
          go build \
            -ldflags "-X go-falcon/pkg/version.Version=${{ steps.version.outputs.VERSION }} \
                     -X go-falcon/pkg/version.GitCommit=${{ steps.version.outputs.GIT_COMMIT }} \
                     -X go-falcon/pkg/version.BuildDate=${{ steps.version.outputs.BUILD_DATE }}" \
            -o falcon ./cmd/gateway
```

## 🐛 Troubleshooting

### Common Issues

#### "Working directory is not clean"
```bash
# Check what's changed
git status

# Commit or stash changes
git add -A
git commit -m "chore: save work in progress"

# Or stash temporarily
git stash push -m "temporary changes"
```

#### "No unreleased changes found"
```bash
# Add some changes to the changelog first
./scripts/changelog.sh add fixed "Some bug fix"

# Then try release again
./scripts/release.sh patch
```

#### "Invalid semantic version"
```bash
# Check current version
go run cmd/version/main.go current

# Validate version format
go run cmd/version/main.go validate 1.2.3

# If using git tags, ensure they follow v1.2.3 format
git tag -l
```

#### "CHANGELOG.md not found"
```bash
# Recreate changelog file
cat > CHANGELOG.md << 'EOF'
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project setup
EOF
```

### Validation Commands

```bash
# Validate everything is working
./scripts/changelog.sh validate
go run cmd/version/main.go current
git status

# Test release process (dry run)
./scripts/release.sh patch --dry-run
```

### Reset/Recovery

If something goes wrong during release:

```bash
# Remove last commit (if not pushed)
git reset --hard HEAD~1

# Remove last tag (if not pushed)
git tag -d v1.2.3

# Reset changelog (if backed up)
cp CHANGELOG.md.bak CHANGELOG.md
```

## 📚 Best Practices

### Commit Messages

Use conventional commits for better changelog generation:

```bash
feat: add user authentication system
fix: resolve memory leak in scheduler
docs: update API documentation
refactor: improve database connection handling
test: add integration tests for auth module
chore: update dependencies
```

### Changelog Entries

- Write clear, user-focused descriptions
- Start with an action verb
- Be specific about what changed
- Include relevant context when needed

**Good examples:**
```bash
./scripts/changelog.sh add added "OAuth2 authentication with Google and GitHub providers"
./scripts/changelog.sh add fixed "Memory leak in task scheduler affecting long-running processes"
./scripts/changelog.sh add changed "API response format now includes metadata and pagination"
```

**Avoid:**
```bash
./scripts/changelog.sh add added "New stuff"
./scripts/changelog.sh add fixed "Bug"
./scripts/changelog.sh add changed "Updates"
```

### Release Timing

- **Patch releases**: Bug fixes, security patches (can be released immediately)
- **Minor releases**: New features, planned regularly (weekly/bi-weekly)
- **Major releases**: Breaking changes, planned well in advance (quarterly/yearly)

### Version Strategy

- Start with `0.1.0` for initial development
- Use `1.0.0` for first stable public release
- Reserve major bumps for breaking changes
- Use patch releases for hotfixes

## 🔗 Related Documentation

- [Keep a Changelog](https://keepachangelog.com/en/1.0.0/)
- [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [Go Falcon CLAUDE.md](./CLAUDE.md) - Main project documentation

---

For questions or issues with the version management system, check the troubleshooting section above or create an issue in the project repository.