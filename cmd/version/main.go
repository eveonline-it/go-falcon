package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"go-falcon/pkg/version"
)

func main() {
	if len(os.Args) < 2 {
		showHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "info", "i":
		showVersionInfo()
	case "current", "c":
		showCurrentVersion()
	case "bump", "b":
		if len(os.Args) < 3 {
			fmt.Println("Error: bump type required (patch|minor|major)")
			os.Exit(1)
		}
		bumpVersion(os.Args[2])
	case "parse", "p":
		if len(os.Args) < 3 {
			fmt.Println("Error: version string required")
			os.Exit(1)
		}
		parseVersionString(os.Args[2])
	case "compare", "cmp":
		if len(os.Args) < 4 {
			fmt.Println("Error: two version strings required")
			os.Exit(1)
		}
		compareVersions(os.Args[2], os.Args[3])
	case "validate", "v":
		if len(os.Args) < 3 {
			fmt.Println("Error: version string required")
			os.Exit(1)
		}
		validateVersion(os.Args[2])
	case "changelog", "cl":
		handleChangelogCommands(os.Args[2:])
	case "json", "j":
		showVersionJSON()
	case "help", "h", "--help", "-h":
		showHelp()
	default:
		fmt.Printf("Error: unknown command '%s'\n", command)
		showHelp()
		os.Exit(1)
	}
}

func showVersionInfo() {
	fmt.Println(version.GetBuildInfo())
}

func showCurrentVersion() {
	fmt.Println(version.GetVersionString())
}

func showVersionJSON() {
	info := version.Get()
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		log.Fatalf("Error encoding JSON: %v", err)
	}
	fmt.Println(string(data))
}

func bumpVersion(bumpType string) {
	var vt version.VersionType
	switch bumpType {
	case "patch", "p":
		vt = version.Patch
	case "minor", "m":
		vt = version.Minor
	case "major", "M":
		vt = version.Major
	default:
		fmt.Printf("Error: invalid bump type '%s'. Must be patch, minor, or major\n", bumpType)
		os.Exit(1)
	}

	newVersion, err := version.BumpVersion(vt)
	if err != nil {
		log.Fatalf("Error bumping version: %v", err)
	}

	fmt.Printf("Current: %s\n", version.GetVersionString())
	fmt.Printf("New: %s\n", newVersion)
}

func parseVersionString(versionStr string) {
	sv, err := version.ParseVersion(versionStr)
	if err != nil {
		log.Fatalf("Error parsing version: %v", err)
	}

	fmt.Printf("Version: %s\n", versionStr)
	fmt.Printf("Major: %d\n", sv.Major)
	fmt.Printf("Minor: %d\n", sv.Minor)
	fmt.Printf("Patch: %d\n", sv.Patch)
}

func compareVersions(v1, v2 string) {
	result, err := version.CompareVersions(v1, v2)
	if err != nil {
		log.Fatalf("Error comparing versions: %v", err)
	}

	fmt.Printf("Comparing: %s vs %s\n", v1, v2)
	switch result {
	case -1:
		fmt.Printf("Result: %s < %s\n", v1, v2)
	case 0:
		fmt.Printf("Result: %s = %s\n", v1, v2)
	case 1:
		fmt.Printf("Result: %s > %s\n", v1, v2)
	}
}

func validateVersion(versionStr string) {
	if version.IsValidVersion(versionStr) {
		fmt.Printf("✓ '%s' is a valid semantic version\n", versionStr)
	} else {
		fmt.Printf("✗ '%s' is not a valid semantic version\n", versionStr)
		os.Exit(1)
	}
}

func handleChangelogCommands(args []string) {
	if len(args) == 0 {
		fmt.Println("Error: changelog command required")
		showChangelogHelp()
		os.Exit(1)
	}

	command := args[0]
	switch command {
	case "add":
		if len(args) < 3 {
			fmt.Println("Error: changelog add requires type and description")
			os.Exit(1)
		}
		addChangelogEntry(args[1], args[2])
	case "release":
		if len(args) < 2 {
			fmt.Println("Error: changelog release requires version")
			os.Exit(1)
		}
		releaseChangelog(args[1])
	case "latest":
		count := 5
		if len(args) > 1 {
			var err error
			count, err = strconv.Atoi(args[1])
			if err != nil {
				fmt.Printf("Error: invalid count '%s'\n", args[1])
				os.Exit(1)
			}
		}
		showLatestChangelog(count)
	case "help":
		showChangelogHelp()
	default:
		fmt.Printf("Error: unknown changelog command '%s'\n", command)
		showChangelogHelp()
		os.Exit(1)
	}
}

func addChangelogEntry(changeType, description string) {
	changelog, err := version.ParseChangelog("CHANGELOG.md")
	if err != nil {
		log.Fatalf("Error parsing changelog: %v", err)
	}

	var ct version.ChangeType
	switch changeType {
	case "added", "add":
		ct = version.Added
	case "changed", "change":
		ct = version.Changed
	case "deprecated", "deprecate":
		ct = version.Deprecated
	case "removed", "remove":
		ct = version.Removed
	case "fixed", "fix":
		ct = version.Fixed
	case "security", "sec":
		ct = version.Security
	default:
		fmt.Printf("Error: invalid change type '%s'\n", changeType)
		os.Exit(1)
	}

	changelog.AddEntry(ct, description)

	err = changelog.WriteChangelog("CHANGELOG.md")
	if err != nil {
		log.Fatalf("Error writing changelog: %v", err)
	}

	fmt.Printf("✓ Added %s entry: %s\n", ct.String(), description)
}

func releaseChangelog(versionStr string) {
	if !version.IsValidVersion(versionStr) {
		fmt.Printf("Error: invalid version '%s'\n", versionStr)
		os.Exit(1)
	}

	changelog, err := version.ParseChangelog("CHANGELOG.md")
	if err != nil {
		log.Fatalf("Error parsing changelog: %v", err)
	}

	err = changelog.ReleaseVersion(versionStr)
	if err != nil {
		log.Fatalf("Error releasing changelog: %v", err)
	}

	err = changelog.WriteChangelog("CHANGELOG.md")
	if err != nil {
		log.Fatalf("Error writing changelog: %v", err)
	}

	fmt.Printf("✓ Released changelog for version %s\n", versionStr)
}

func showLatestChangelog(count int) {
	changelog, err := version.ParseChangelog("CHANGELOG.md")
	if err != nil {
		log.Fatalf("Error parsing changelog: %v", err)
	}

	fmt.Printf("Latest %d changelog entries:\n\n", count)
	
	shown := 0
	for _, release := range changelog.Releases {
		if shown >= count {
			break
		}
		
		if release.Date != "" {
			fmt.Printf("## [%s] - %s\n", release.Version, release.Date)
		} else {
			fmt.Printf("## [%s]\n", release.Version)
		}
		
		// Group changes by type
		changesByType := make(map[version.ChangeType][]string)
		for _, change := range release.Changes {
			changesByType[change.Type] = append(changesByType[change.Type], change.Description)
		}
		
		// Display changes
		for _, changeType := range []version.ChangeType{version.Added, version.Changed, version.Deprecated, version.Removed, version.Fixed, version.Security} {
			if changes, exists := changesByType[changeType]; exists && len(changes) > 0 {
				fmt.Printf("\n### %s\n", changeType.String())
				for _, change := range changes {
					fmt.Printf("- %s\n", change)
				}
			}
		}
		
		fmt.Println()
		shown++
	}
}

func showChangelogHelp() {
	fmt.Println("Changelog commands:")
	fmt.Println("  add <type> <description>  Add entry to unreleased section")
	fmt.Println("  release <version>         Release unreleased changes")
	fmt.Println("  latest [count]            Show latest entries (default: 5)")
	fmt.Println("  help                      Show this help")
	fmt.Println("")
	fmt.Println("Change types: added, changed, deprecated, removed, fixed, security")
}

func showHelp() {
	fmt.Println("Go Falcon Version Management Tool")
	fmt.Println("")
	fmt.Println("Usage: version <command> [options]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  info, i              Show detailed version information")
	fmt.Println("  current, c           Show current version string")
	fmt.Println("  json, j              Show version information as JSON")
	fmt.Println("  bump, b <type>       Calculate bumped version (patch|minor|major)")
	fmt.Println("  parse, p <version>   Parse and display version components")
	fmt.Println("  compare, cmp <v1> <v2>  Compare two versions")
	fmt.Println("  validate, v <version>   Validate semantic version format")
	fmt.Println("  changelog, cl <cmd>     Changelog management commands")
	fmt.Println("  help, h              Show this help message")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  version info                    # Show build information")
	fmt.Println("  version bump patch              # Calculate next patch version")
	fmt.Println("  version compare 1.0.0 2.0.0     # Compare versions")
	fmt.Println("  version validate v1.2.3         # Validate version format")
	fmt.Println("  version changelog add fixed \"Fix bug\"  # Add changelog entry")
	fmt.Println("  version changelog release 1.2.3 # Release changelog version")
}