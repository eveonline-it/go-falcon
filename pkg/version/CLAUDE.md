# Version Package (pkg/version)

## Overview
Application version information and build metadata management. Provides consistent version reporting across all modules and health endpoints.

## Core Features
- **Build Information**: Version, git commit, build date tracking
- **Environment Detection**: Go version and platform information
- **Health Integration**: Version data in health check responses
- **Development Support**: "dev" version for development builds

## Version Information
```go
type Info struct {
    Version    string  // Semantic version or "dev"
    GitCommit  string  // Git commit hash
    BuildDate  string  // Build timestamp
    GoVersion  string  // Go compiler version
    Platform   string  // OS/Architecture
}
```

## Usage Examples
```go
// Get version information
versionInfo := version.Get()
versionString := version.GetVersionString()

// Health check integration
response := fmt.Sprintf(`{
    "status": "healthy",
    "version": "%s",
    "git_commit": "%s",
    "build_date": "%s"
}`, versionInfo.Version, versionInfo.GitCommit, versionInfo.BuildDate)
```

## Build Integration
- **Compile-Time Variables**: Set via -ldflags during build
- **Default Values**: "dev" and "unknown" for development builds
- **CI/CD Integration**: Automatic version injection in builds

## Features
- **Consistent Reporting**: Same version info across all modules
- **Development Friendly**: Clear distinction between dev and production
- **Build Metadata**: Complete build environment information
- **Health Monitoring**: Version tracking in health endpoints

## Integration
- Used in health check handlers
- Displayed in application startup logs
- Available for monitoring and debugging
- Consistent across all modules and endpoints