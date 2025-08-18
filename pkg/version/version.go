package version

import (
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Build information. These variables are set at build time via ldflags.
var (
	// Version is the semantic version (e.g., "1.2.3")
	Version = "dev"
	
	// GitCommit is the git commit hash
	GitCommit = "unknown"
	
	// GitBranch is the git branch name
	GitBranch = "unknown"
	
	// BuildDate is the build timestamp
	BuildDate = "unknown"
	
	// BuildUser is who built the binary
	BuildUser = "unknown"
)

// Info contains all version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	GitBranch string `json:"git_branch"`
	BuildDate string `json:"build_date"`
	BuildUser string `json:"build_user"`
	GoVersion string `json:"go_version"`
	Compiler  string `json:"compiler"`
	Platform  string `json:"platform"`
}

// Get returns the version information
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
		BuildDate: BuildDate,
		BuildUser: BuildUser,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// GetVersionString returns a formatted version string
func GetVersionString() string {
	if GitCommit != "unknown" {
		if len(GitCommit) > 7 {
			return fmt.Sprintf("%s (%s)", Version, GitCommit[:7])
		}
		return fmt.Sprintf("%s (%s)", Version, GitCommit)
	}
	return Version
}

// GetBuildInfo returns detailed build information
func GetBuildInfo() string {
	info := Get()
	return fmt.Sprintf("Version: %s\nCommit: %s\nBranch: %s\nBuilt: %s by %s\nGo: %s (%s)\nPlatform: %s",
		info.Version,
		info.GitCommit,
		info.GitBranch,
		info.BuildDate,
		info.BuildUser,
		info.GoVersion,
		info.Compiler,
		info.Platform,
	)
}

// String implements the Stringer interface for Info
func (i Info) String() string {
	return fmt.Sprintf("%s (%s)", i.Version, i.GitCommit[:min(7, len(i.GitCommit))])
}

// VersionType represents the type of version bump
type VersionType int

const (
	Patch VersionType = iota
	Minor
	Major
)

// String returns the string representation of VersionType
func (vt VersionType) String() string {
	switch vt {
	case Patch:
		return "patch"
	case Minor:
		return "minor"
	case Major:
		return "major"
	default:
		return "unknown"
	}
}

// SemanticVersion represents a semantic version
type SemanticVersion struct {
	Major int
	Minor int
	Patch int
}

// String returns the semantic version as a string
func (sv SemanticVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", sv.Major, sv.Minor, sv.Patch)
}

// ParseVersion parses a semantic version string
func ParseVersion(version string) (SemanticVersion, error) {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")
	
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-.*)?(?:\+.*)?$`)
	matches := re.FindStringSubmatch(version)
	
	if len(matches) < 4 {
		return SemanticVersion{}, fmt.Errorf("invalid semantic version: %s", version)
	}
	
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return SemanticVersion{}, fmt.Errorf("invalid major version: %s", matches[1])
	}
	
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return SemanticVersion{}, fmt.Errorf("invalid minor version: %s", matches[2])
	}
	
	patch, err := strconv.Atoi(matches[3])
	if err != nil {
		return SemanticVersion{}, fmt.Errorf("invalid patch version: %s", matches[3])
	}
	
	return SemanticVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, nil
}

// Bump increments the version based on the bump type
func (sv SemanticVersion) Bump(versionType VersionType) SemanticVersion {
	switch versionType {
	case Major:
		return SemanticVersion{Major: sv.Major + 1, Minor: 0, Patch: 0}
	case Minor:
		return SemanticVersion{Major: sv.Major, Minor: sv.Minor + 1, Patch: 0}
	case Patch:
		return SemanticVersion{Major: sv.Major, Minor: sv.Minor, Patch: sv.Patch + 1}
	default:
		return sv
	}
}

// GetCurrentVersion returns the current version as a SemanticVersion
func GetCurrentVersion() (SemanticVersion, error) {
	if Version == "dev" {
		return SemanticVersion{Major: 0, Minor: 1, Patch: 0}, nil
	}
	return ParseVersion(Version)
}

// BumpVersion returns a new version string bumped by the specified type
func BumpVersion(versionType VersionType) (string, error) {
	current, err := GetCurrentVersion()
	if err != nil {
		return "", err
	}
	
	bumped := current.Bump(versionType)
	return bumped.String(), nil
}

// IsValidVersion checks if a version string is valid semantic version
func IsValidVersion(version string) bool {
	_, err := ParseVersion(version)
	return err == nil
}

// CompareVersions compares two version strings
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func CompareVersions(v1, v2 string) (int, error) {
	sv1, err := ParseVersion(v1)
	if err != nil {
		return 0, err
	}
	
	sv2, err := ParseVersion(v2)
	if err != nil {
		return 0, err
	}
	
	if sv1.Major != sv2.Major {
		if sv1.Major < sv2.Major {
			return -1, nil
		}
		return 1, nil
	}
	
	if sv1.Minor != sv2.Minor {
		if sv1.Minor < sv2.Minor {
			return -1, nil
		}
		return 1, nil
	}
	
	if sv1.Patch != sv2.Patch {
		if sv1.Patch < sv2.Patch {
			return -1, nil
		}
		return 1, nil
	}
	
	return 0, nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}