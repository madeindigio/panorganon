package version

import "fmt"

var (
	// Version is the semantic version (injected at build time)
	Version = "dev"
	// Commit is the git commit hash (injected at build time)
	Commit = "unknown"
	// BuildTime is the build timestamp (injected at build time)
	BuildTime = "unknown"
)

// Info returns a formatted version string
func Info() string {
	return fmt.Sprintf("Panorganon %s (commit: %s, built: %s)", Version, Commit, BuildTime)
}

// Short returns just the version number
func Short() string {
	return Version
}
