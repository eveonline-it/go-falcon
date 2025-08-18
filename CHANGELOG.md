# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Added

- Enhanced commit workflow with automatic changelog integration

### Added

- Smart commit script with interactive changelog prompts and commit workflow documentation
- Initial version management and changelog system
- Comprehensive version information tracking
- Build metadata collection

### Changed
- Enhanced version package with management utilities

### Deprecated

### Removed

### Fixed

### Security

## [0.1.0] - 2025-08-18

### Added
- Initial Go Falcon monolithic API server
- EVE Online SSO integration
- Task scheduling system with cron support
- Modular architecture with unified HUMA API
- OpenTelemetry observability
- MongoDB and Redis integration
- Docker containerization
- Comprehensive permission system

### Security
- JWT-based authentication
- EVE SSO OAuth2 integration
- CSRF protection with state validation

---

## Release Types

- **Major** (X.0.0): Breaking changes, major features
- **Minor** (0.X.0): New features, backwards compatible
- **Patch** (0.0.X): Bug fixes, security patches

## Conventional Commits

This project uses [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes
- `refactor:` - Code refactoring
- `test:` - Test additions/changes
- `chore:` - Maintenance tasks
- `perf:` - Performance improvements
- `style:` - Code style changes
- `ci:` - CI/CD changes
- `build:` - Build system changes

## Breaking Changes

Breaking changes should be marked with `!` in the commit type (e.g., `feat!:`) and detailed in the commit body.