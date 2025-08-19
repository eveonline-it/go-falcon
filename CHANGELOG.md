# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

### Added

- Implemented scheduler task enable/disable functionality with proper error handling and system task protection

### Changed

- Enhanced scheduler API DTOs with authentication headers for improved Huma v2 integration

### Added

- Comprehensive unified Huma API registration enforcement documentation with mandatory implementation guidelines and verification procedures

### Changed

- Updated development workflow documentation with hot reload instructions

### Added

- Automatic super_admin role assignment for first user registration

### Added

- Enhanced scheduler endpoint access control with dual permission checking for admin and super_admin roles

### Added

- Comprehensive CASBIN debug logging system for troubleshooting authentication and authorization issues

### Changed

- Reorganize CASBIN middleware into dedicated package structure for better maintainability

- Database optimization with comprehensive indexing strategy and monitoring tools
- Hierarchical permissions support for character/corporation/alliance levels
- 11 HUMA v2 API endpoints for role assignment and permission management
- Comprehensive CASBIN role-based authorization system with MongoDB adapter and Redis caching
- CASBIN middleware with debug logging for granular permission system

### Fixed

- Enhance /auth/status endpoint with character resolution and remove debug endpoint - Add character_ids, corporation_ids, alliance_ids fields to AuthStatusResponse - Remove unused /auth/debug/characters endpoint and debug DTOs - Remove unused permissions field from auth responses - Hide sensitive cookie values in debug logs for improved security - Provide unified endpoint for comprehensive auth status and character data

### Added

- Clean up middleware package and add character resolver debug endpoint - Remove 6 unnecessary files (test files, examples, duplicates) and consolidate functionality - Add comprehensive debug logging to UserCharacterResolver - Create debug endpoint /auth/debug/characters to test character resolution - Reduce middleware package from 15 to 8 core files for better maintainability

### Added

- Comprehensive middleware system with enhanced authentication and debug logging

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