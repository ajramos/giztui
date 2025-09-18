package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

var (
	// Version is the semantic version number
	Version = "1.2.3"

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
	Version     string `json:"version"`
	GitCommit   string `json:"git_commit"`
	GitBranch   string `json:"git_branch"`
	BuildDate   string `json:"build_date"`
	BuildUser   string `json:"build_user"`
	GoVersion   string `json:"go_version"`
	Platform    string `json:"platform"`
	BuildMethod string `json:"build_method"` // "make", "go-install", or "unknown"
	VCSTime     string `json:"vcs_time"`     // VCS commit time
	VCSModified bool   `json:"vcs_modified"` // Whether VCS had uncommitted changes
}

// VCSInfo contains version control information detected from build info
type VCSInfo struct {
	Revision string
	Time     string
	Modified bool
	Found    bool
}

// getVCSInfo extracts VCS information from runtime debug info
func getVCSInfo() VCSInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return VCSInfo{}
	}

	vcsInfo := VCSInfo{Found: false}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			vcsInfo.Revision = setting.Value
			vcsInfo.Found = true
		case "vcs.time":
			vcsInfo.Time = setting.Value
		case "vcs.modified":
			vcsInfo.Modified = setting.Value == "true"
		}
	}

	return vcsInfo
}

// getBuildMethod determines how the binary was built
func getBuildMethod() string {
	// If we have custom build info (from make build), it's a make build
	if GitCommit != "unknown" && BuildDate != "unknown" {
		return "make"
	}

	// If we have VCS info but no custom build info, it's likely go install
	vcs := getVCSInfo()
	if vcs.Found {
		return "go-install"
	}

	return "unknown"
}

// GetVersion returns the version string
func GetVersion() string {
	return Version
}

// GetInfo returns comprehensive version information
func GetInfo() Info {
	vcs := getVCSInfo()
	buildMethod := getBuildMethod()

	// Determine the best Git commit to show
	gitCommit := GitCommit
	buildDate := BuildDate
	vcsTime := ""

	if gitCommit == "unknown" && vcs.Found {
		// Use VCS info when custom build info is not available
		gitCommit = vcs.Revision
		if vcs.Time != "" {
			vcsTime = vcs.Time
			// If no custom build date, use VCS time as build date
			if buildDate == "unknown" {
				buildDate = vcs.Time
			}
		}
	} else if vcs.Found {
		vcsTime = vcs.Time
	}

	return Info{
		Version:     Version,
		GitCommit:   gitCommit,
		GitBranch:   GitBranch,
		BuildDate:   buildDate,
		BuildUser:   BuildUser,
		GoVersion:   runtime.Version(),
		Platform:    fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		BuildMethod: buildMethod,
		VCSTime:     vcsTime,
		VCSModified: vcs.Modified,
	}
}

// GetVersionString returns a formatted version string
func GetVersionString() string {
	info := GetInfo()

	if info.GitCommit == "unknown" {
		return fmt.Sprintf("GizTUI %s", info.Version)
	}

	// Truncate git commit to 8 characters for display
	shortCommit := info.GitCommit
	if len(shortCommit) > 8 {
		shortCommit = shortCommit[:8]
	}

	// Add build method indicator for go-install builds
	suffix := shortCommit
	if info.BuildMethod == "go-install" {
		if info.VCSModified {
			suffix = fmt.Sprintf("%s-modified", shortCommit)
		}
	}

	return fmt.Sprintf("GizTUI %s (%s)", info.Version, suffix)
}

// GetDetailedVersionString returns a detailed version string for --version output
func GetDetailedVersionString() string {
	info := GetInfo()

	result := fmt.Sprintf("GizTUI %s\n", info.Version)
	result += fmt.Sprintf("Git commit: %s\n", info.GitCommit)

	// Show branch info if available
	if info.GitBranch != "unknown" {
		result += fmt.Sprintf("Git branch: %s\n", info.GitBranch)
	}

	// Show build date, preferring custom build date over VCS time
	if info.BuildDate != "unknown" {
		// Parse and format the date if it looks like RFC3339 (from VCS)
		if strings.Contains(info.BuildDate, "T") && strings.Contains(info.BuildDate, "Z") {
			if t, err := time.Parse(time.RFC3339, info.BuildDate); err == nil {
				result += fmt.Sprintf("Build date: %s\n", t.Format("2006-01-02 15:04:05 UTC"))
			} else {
				result += fmt.Sprintf("Build date: %s\n", info.BuildDate)
			}
		} else {
			result += fmt.Sprintf("Build date: %s\n", info.BuildDate)
		}
	}

	// Show build user if available
	if info.BuildUser != "unknown" {
		result += fmt.Sprintf("Built by: %s\n", info.BuildUser)
	}

	result += fmt.Sprintf("Build method: %s\n", info.BuildMethod)

	// Show VCS modification status for development builds
	if info.BuildMethod == "go-install" && info.VCSModified {
		result += "VCS status: modified (uncommitted changes)\n"
	}

	result += fmt.Sprintf("Go version: %s\n", info.GoVersion)
	result += fmt.Sprintf("Platform: %s", info.Platform)

	return result
}

// IsRelease returns true if this is a release version (not a dev build)
func IsRelease() bool {
	info := GetInfo()

	// It's a release if:
	// 1. Has version number
	// 2. Has Git commit info (either injected or from VCS)
	// 3. Version doesn't contain "dev"
	// 4. Built using make (has full build metadata) OR is a clean go-install build
	hasGitInfo := info.GitCommit != "unknown"
	isCleanBuild := info.BuildMethod == "make" || (info.BuildMethod == "go-install" && !info.VCSModified)

	return Version != "" && hasGitInfo && !contains(Version, "dev") && isCleanBuild
}

// IsDevelopment returns true if this is a development build
func IsDevelopment() bool {
	return !IsRelease()
}

// GetBuildMethod returns the method used to build this binary
func GetBuildMethod() string {
	return getBuildMethod()
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
