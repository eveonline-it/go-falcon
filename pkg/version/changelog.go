package version

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// ChangeType represents the type of change in changelog
type ChangeType int

const (
	Added ChangeType = iota
	Changed
	Deprecated
	Removed
	Fixed
	Security
)

// String returns the string representation of ChangeType
func (ct ChangeType) String() string {
	switch ct {
	case Added:
		return "Added"
	case Changed:
		return "Changed"
	case Deprecated:
		return "Deprecated"
	case Removed:
		return "Removed"
	case Fixed:
		return "Fixed"
	case Security:
		return "Security"
	default:
		return "Unknown"
	}
}

// ChangeEntry represents a single changelog entry
type ChangeEntry struct {
	Type        ChangeType
	Description string
}

// Release represents a version release in the changelog
type Release struct {
	Version     string
	Date        string
	Changes     []ChangeEntry
	IsUnreleased bool
}

// Changelog represents the entire changelog structure
type Changelog struct {
	Releases []Release
}

// AddEntry adds a new entry to the unreleased section
func (c *Changelog) AddEntry(changeType ChangeType, description string) {
	// Find or create unreleased section
	var unreleased *Release
	for i := range c.Releases {
		if c.Releases[i].IsUnreleased {
			unreleased = &c.Releases[i]
			break
		}
	}
	
	if unreleased == nil {
		// Create new unreleased section
		unreleased = &Release{
			Version:      "Unreleased",
			Date:         "",
			Changes:      []ChangeEntry{},
			IsUnreleased: true,
		}
		// Prepend to releases
		c.Releases = append([]Release{*unreleased}, c.Releases...)
		unreleased = &c.Releases[0]
	}
	
	// Add the new entry
	unreleased.Changes = append(unreleased.Changes, ChangeEntry{
		Type:        changeType,
		Description: description,
	})
}

// ReleaseVersion converts unreleased changes to a versioned release
func (c *Changelog) ReleaseVersion(version string) error {
	// Find unreleased section
	var unreleasedIndex = -1
	for i, release := range c.Releases {
		if release.IsUnreleased {
			unreleasedIndex = i
			break
		}
	}
	
	if unreleasedIndex == -1 {
		return fmt.Errorf("no unreleased changes found")
	}
	
	// Update the release
	c.Releases[unreleasedIndex].Version = version
	c.Releases[unreleasedIndex].Date = time.Now().Format("2006-01-02")
	c.Releases[unreleasedIndex].IsUnreleased = false
	
	// Create new empty unreleased section
	newUnreleased := Release{
		Version:      "Unreleased",
		Date:         "",
		Changes:      []ChangeEntry{},
		IsUnreleased: true,
	}
	c.Releases = append([]Release{newUnreleased}, c.Releases...)
	
	return nil
}

// ParseChangelog parses a CHANGELOG.md file
func ParseChangelog(filename string) (*Changelog, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	
	changelog := &Changelog{}
	scanner := bufio.NewScanner(file)
	
	var currentRelease *Release
	var currentChangeType *ChangeType
	
	// Regex patterns
	releasePattern := regexp.MustCompile(`^## \[(.*?)\](?:\s*-\s*(.+))?`)
	changeTypePattern := regexp.MustCompile(`^### (Added|Changed|Deprecated|Removed|Fixed|Security)`)
	entryPattern := regexp.MustCompile(`^-\s+(.+)`)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "<!--") {
			continue
		}
		
		// Check for release header
		if matches := releasePattern.FindStringSubmatch(line); matches != nil {
			// Save previous release if exists
			if currentRelease != nil {
				changelog.Releases = append(changelog.Releases, *currentRelease)
			}
			
			version := matches[1]
			date := ""
			if len(matches) > 2 && matches[2] != "" {
				date = matches[2]
			}
			
			currentRelease = &Release{
				Version:      version,
				Date:         date,
				Changes:      []ChangeEntry{},
				IsUnreleased: version == "Unreleased",
			}
			currentChangeType = nil
			continue
		}
		
		// Check for change type header
		if matches := changeTypePattern.FindStringSubmatch(line); matches != nil {
			changeTypeStr := matches[1]
			var ct ChangeType
			switch changeTypeStr {
			case "Added":
				ct = Added
			case "Changed":
				ct = Changed
			case "Deprecated":
				ct = Deprecated
			case "Removed":
				ct = Removed
			case "Fixed":
				ct = Fixed
			case "Security":
				ct = Security
			}
			currentChangeType = &ct
			continue
		}
		
		// Check for change entry
		if currentRelease != nil && currentChangeType != nil {
			if matches := entryPattern.FindStringSubmatch(line); matches != nil {
				entry := ChangeEntry{
					Type:        *currentChangeType,
					Description: matches[1],
				}
				currentRelease.Changes = append(currentRelease.Changes, entry)
			}
		}
	}
	
	// Add the last release
	if currentRelease != nil {
		changelog.Releases = append(changelog.Releases, *currentRelease)
	}
	
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	
	return changelog, nil
}

// WriteChangelog writes the changelog to a file
func (c *Changelog) WriteChangelog(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write header
	fmt.Fprintln(file, "# Changelog")
	fmt.Fprintln(file, "")
	fmt.Fprintln(file, "All notable changes to this project will be documented in this file.")
	fmt.Fprintln(file, "")
	fmt.Fprintln(file, "The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),")
	fmt.Fprintln(file, "and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).")
	fmt.Fprintln(file, "")
	
	// Write releases
	for _, release := range c.Releases {
		if release.Date != "" {
			fmt.Fprintf(file, "## [%s] - %s\n", release.Version, release.Date)
		} else {
			fmt.Fprintf(file, "## [%s]\n", release.Version)
		}
		fmt.Fprintln(file, "")
		
		// Group changes by type
		changesByType := make(map[ChangeType][]string)
		for _, change := range release.Changes {
			changesByType[change.Type] = append(changesByType[change.Type], change.Description)
		}
		
		// Write changes in order
		for _, changeType := range []ChangeType{Added, Changed, Deprecated, Removed, Fixed, Security} {
			if changes, exists := changesByType[changeType]; exists && len(changes) > 0 {
				fmt.Fprintf(file, "### %s\n", changeType.String())
				fmt.Fprintln(file, "")
				for _, change := range changes {
					fmt.Fprintf(file, "- %s\n", change)
				}
				fmt.Fprintln(file, "")
			}
		}
	}
	
	// Write footer information
	fmt.Fprintln(file, "---")
	fmt.Fprintln(file, "")
	fmt.Fprintln(file, "## Release Types")
	fmt.Fprintln(file, "")
	fmt.Fprintln(file, "- **Major** (X.0.0): Breaking changes, major features")
	fmt.Fprintln(file, "- **Minor** (0.X.0): New features, backwards compatible")
	fmt.Fprintln(file, "- **Patch** (0.0.X): Bug fixes, security patches")
	fmt.Fprintln(file, "")
	fmt.Fprintln(file, "## Conventional Commits")
	fmt.Fprintln(file, "")
	fmt.Fprintln(file, "This project uses [Conventional Commits](https://www.conventionalcommits.org/):")
	fmt.Fprintln(file, "")
	fmt.Fprintln(file, "- `feat:` - New features")
	fmt.Fprintln(file, "- `fix:` - Bug fixes")
	fmt.Fprintln(file, "- `docs:` - Documentation changes")
	fmt.Fprintln(file, "- `refactor:` - Code refactoring")
	fmt.Fprintln(file, "- `test:` - Test additions/changes")
	fmt.Fprintln(file, "- `chore:` - Maintenance tasks")
	fmt.Fprintln(file, "- `perf:` - Performance improvements")
	fmt.Fprintln(file, "- `style:` - Code style changes")
	fmt.Fprintln(file, "- `ci:` - CI/CD changes")
	fmt.Fprintln(file, "- `build:` - Build system changes")
	fmt.Fprintln(file, "")
	fmt.Fprintln(file, "## Breaking Changes")
	fmt.Fprintln(file, "")
	fmt.Fprintln(file, "Breaking changes should be marked with `!` in the commit type (e.g., `feat!:`) and detailed in the commit body.")
	
	return nil
}

// GetLatestRelease returns the latest non-unreleased version
func (c *Changelog) GetLatestRelease() *Release {
	for _, release := range c.Releases {
		if !release.IsUnreleased {
			return &release
		}
	}
	return nil
}

// HasUnreleasedChanges returns true if there are unreleased changes
func (c *Changelog) HasUnreleasedChanges() bool {
	for _, release := range c.Releases {
		if release.IsUnreleased && len(release.Changes) > 0 {
			return true
		}
	}
	return false
}