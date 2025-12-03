// Package version provides build information and version handling for AGIS.
package version

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"runtime/debug"
	"time"
)

// These variables are set at build time via ldflags.
// Example: go build -ldflags "-X github.com/wethegamers/agis/internal/version.Version=1.0.0"
var (
	// Version is the semantic version of the application.
	Version = "dev"

	// GitCommit is the git commit SHA.
	GitCommit = "unknown"

	// GitBranch is the git branch.
	GitBranch = "unknown"

	// BuildTime is the build timestamp.
	BuildTime = "unknown"

	// GoVersion is the Go version used to build.
	GoVersion = runtime.Version()
)

// Info holds all version and build information.
type Info struct {
	Version      string           `json:"version"`
	GitCommit    string           `json:"git_commit"`
	GitBranch    string           `json:"git_branch"`
	BuildTime    string           `json:"build_time"`
	GoVersion    string           `json:"go_version"`
	Platform     string           `json:"platform"`
	Compiler     string           `json:"compiler"`
	StartTime    time.Time        `json:"start_time"`
	Uptime       string           `json:"uptime,omitempty"`
	VCSInfo      *VCSInfo         `json:"vcs_info,omitempty"`
	Dependencies []DependencyInfo `json:"dependencies,omitempty"`
}

// VCSInfo holds version control information from debug.BuildInfo.
type VCSInfo struct {
	Revision string `json:"revision,omitempty"`
	Time     string `json:"time,omitempty"`
	Modified bool   `json:"modified,omitempty"`
}

// DependencyInfo holds dependency information.
type DependencyInfo struct {
	Path    string `json:"path"`
	Version string `json:"version"`
}

var startTime = time.Now()

// Get returns the current version info.
func Get() Info {
	info := Info{
		Version:   Version,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
		BuildTime: BuildTime,
		GoVersion: GoVersion,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		Compiler:  runtime.Compiler,
		StartTime: startTime,
		Uptime:    time.Since(startTime).Round(time.Second).String(),
	}

	// Try to get build info from runtime
	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		// Extract VCS information
		vcsInfo := &VCSInfo{}
		hasVCS := false
		for _, setting := range buildInfo.Settings {
			switch setting.Key {
			case "vcs.revision":
				vcsInfo.Revision = setting.Value
				hasVCS = true
			case "vcs.time":
				vcsInfo.Time = setting.Value
			case "vcs.modified":
				vcsInfo.Modified = setting.Value == "true"
			}
		}
		if hasVCS {
			info.VCSInfo = vcsInfo
			// Use VCS revision if GitCommit wasn't set via ldflags
			if info.GitCommit == "unknown" && vcsInfo.Revision != "" {
				info.GitCommit = vcsInfo.Revision
			}
		}

		// Extract key dependencies
		for _, dep := range buildInfo.Deps {
			// Only include direct dependencies (not replace or transitive)
			if dep.Path != "" && !isTransitiveDep(dep.Path) {
				info.Dependencies = append(info.Dependencies, DependencyInfo{
					Path:    dep.Path,
					Version: dep.Version,
				})
			}
		}
	}

	return info
}

// isTransitiveDep checks if a dependency path looks like a transitive dependency.
func isTransitiveDep(path string) bool {
	// Keep only key direct dependencies
	keyDeps := map[string]bool{
		"github.com/bwmarrin/discordgo":       true,
		"github.com/prometheus/client_golang": true,
		"go.opentelemetry.io/otel":            true,
		"k8s.io/client-go":                    true,
		"github.com/lib/pq":                   true,
		"github.com/google/uuid":              true,
		"golang.org/x/time":                   true,
	}
	return !keyDeps[path]
}

// String returns a human-readable version string.
func (i Info) String() string {
	return fmt.Sprintf("%s (commit: %s, built: %s, go: %s)",
		i.Version, shortCommit(i.GitCommit), i.BuildTime, i.GoVersion)
}

// ShortString returns a short version string.
func (i Info) ShortString() string {
	return fmt.Sprintf("%s-%s", i.Version, shortCommit(i.GitCommit))
}

func shortCommit(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}

// Handler returns an HTTP handler for the version endpoint.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := Get()

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")

		if err := json.NewEncoder(w).Encode(info); err != nil {
			http.Error(w, "failed to encode version info", http.StatusInternalServerError)
		}
	})
}

// ShortHandler returns an HTTP handler that returns just the version string.
func ShortHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := Get()
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintln(w, info.ShortString())
	})
}

// Runtime returns runtime statistics.
type Runtime struct {
	NumGoroutine int    `json:"num_goroutine"`
	NumCPU       int    `json:"num_cpu"`
	GOMAXPROCS   int    `json:"gomaxprocs"`
	MemAlloc     uint64 `json:"mem_alloc_bytes"`
	MemTotal     uint64 `json:"mem_total_alloc_bytes"`
	MemSys       uint64 `json:"mem_sys_bytes"`
	NumGC        uint32 `json:"num_gc"`
}

// GetRuntime returns current runtime statistics.
func GetRuntime() Runtime {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return Runtime{
		NumGoroutine: runtime.NumGoroutine(),
		NumCPU:       runtime.NumCPU(),
		GOMAXPROCS:   runtime.GOMAXPROCS(0),
		MemAlloc:     m.Alloc,
		MemTotal:     m.TotalAlloc,
		MemSys:       m.Sys,
		NumGC:        m.NumGC,
	}
}

// RuntimeHandler returns an HTTP handler for runtime stats.
func RuntimeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stats := GetRuntime()

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")

		if err := json.NewEncoder(w).Encode(stats); err != nil {
			http.Error(w, "failed to encode runtime stats", http.StatusInternalServerError)
		}
	})
}

// FullInfoHandler returns complete info including version and runtime.
func FullInfoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := struct {
			Version Info    `json:"version"`
			Runtime Runtime `json:"runtime"`
		}{
			Version: Get(),
			Runtime: GetRuntime(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache")

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "failed to encode info", http.StatusInternalServerError)
		}
	})
}
