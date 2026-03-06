// Package version provides version information for the Shepherd application
package version

import (
	"fmt"
	"runtime"
)

// Version information
var (
	// Version is the application version
	Version = "0.6.0"

	// GitCommit is the git commit hash
	GitCommit = "unknown"

	// BuildDate is the build date/time
	BuildDate = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = runtime.Version()

	// Platform is the OS/Arch combination
	Platform = runtime.GOOS + "/" + runtime.GOARCH
)

// VersionInfo contains complete version information
type VersionInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// GetVersionInfo returns complete version information
func GetVersionInfo() *VersionInfo {
	return &VersionInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		Platform:  Platform,
	}
}

// String returns the version as a string
func (v *VersionInfo) String() string {
	if v.GitCommit != "unknown" {
		return fmt.Sprintf("%s (commit: %s)", v.Version, v.GitCommit)
	}
	return v.Version
}

// FullString returns detailed version information
func (v *VersionInfo) FullString() string {
	return fmt.Sprintf("Shepherd %s\nGit Commit: %s\nBuild Date: %s\nGo Version: %s\nPlatform: %s",
		v.Version, v.GitCommit, v.BuildDate, v.GoVersion, v.Platform)
}

// GetVersion returns the current version string
func GetVersion() string {
	return Version
}

// SetVersion sets the version information (used during build)
func SetVersion(version, commit, date string) {
	Version = version
	GitCommit = commit
	BuildDate = date
}
