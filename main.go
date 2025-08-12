package main

import (
	"mysql-schema-sync/cmd"
)

// Version information (set by build flags)
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
	GoVersion = "unknown"
)

func main() {
	// Set version information for the CLI
	cmd.SetVersionInfo(Version, BuildTime, GitCommit, GoVersion)
	cmd.Execute()
}
