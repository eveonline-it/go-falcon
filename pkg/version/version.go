package version

import (
	"fmt"
	"runtime"
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