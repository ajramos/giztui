package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version number
	Version = "1.0.1"
	
	// GitCommit is the git commit hash (injected at build time)
	GitCommit = "unknown"
	
	// GitBranch is the git branch (injected at build time)
	GitBranch = "unknown"
	
	// BuildDate is the build date (injected at build time)
	BuildDate = "unknown"
	
	// BuildUser is the user who built the binary (injected at build time)
	BuildUser = "unknown"
)

// Info contains version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	GitBranch string `json:"git_branch"`
	BuildDate string `json:"build_date"`
	BuildUser string `json:"build_user"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// GetVersion returns the version string
func GetVersion() string {
	return Version
}

// GetInfo returns comprehensive version information
func GetInfo() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
		BuildDate: BuildDate,
		BuildUser: BuildUser,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// GetVersionString returns a formatted version string
func GetVersionString() string {
	info := GetInfo()
	
	if GitCommit == "unknown" {
		return fmt.Sprintf("GizTUI %s", info.Version)
	}
	
	// Truncate git commit to 8 characters for display
	shortCommit := GitCommit
	if len(shortCommit) > 8 {
		shortCommit = shortCommit[:8]
	}
	
	return fmt.Sprintf("GizTUI %s (%s)", info.Version, shortCommit)
}

// GetDetailedVersionString returns a detailed version string for --version output
func GetDetailedVersionString() string {
	info := GetInfo()
	
	result := fmt.Sprintf("GizTUI %s\n", info.Version)
	result += fmt.Sprintf("Git commit: %s\n", info.GitCommit)
	result += fmt.Sprintf("Git branch: %s\n", info.GitBranch)
	result += fmt.Sprintf("Build date: %s\n", info.BuildDate)
	result += fmt.Sprintf("Built by: %s\n", info.BuildUser)
	result += fmt.Sprintf("Go version: %s\n", info.GoVersion)
	result += fmt.Sprintf("Platform: %s", info.Platform)
	
	return result
}

// IsRelease returns true if this is a release version (not a dev build)
func IsRelease() bool {
	return Version != "" && GitCommit != "unknown" && !contains(Version, "dev")
}

// IsDevelopment returns true if this is a development build
func IsDevelopment() bool {
	return !IsRelease()
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}