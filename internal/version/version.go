package version

import (
	"runtime"
	"time"
)

var (
	// Version is the current version of the application
	Version = "v1.6.0"
	// GitCommit is the git commit hash
	GitCommit = "unknown"
	// BuildDate is when the binary was built
	BuildDate = "unknown"
	// GoVersion is the Go version used to build the binary
	GoVersion = runtime.Version()
)

// BuildInfo represents build information
type BuildInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	StartTime string `json:"start_time"`
	Uptime    string `json:"uptime"`
}

var startTime = time.Now()

// GetBuildInfo returns the current build information
func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: GoVersion,
		StartTime: startTime.Format(time.RFC3339),
		Uptime:    time.Since(startTime).String(),
	}
}
