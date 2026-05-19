package cmd

import "fmt"

// Build information. Set via -ldflags at build time, e.g.:
//
//	go build -ldflags "-X github.com/victorseara/aipim/cmd.Version=0.1.0 -X github.com/victorseara/aipim/cmd.Commit=abc1234 -X github.com/victorseara/aipim/cmd.Date=2026-05-19"
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func versionString() string {
	return fmt.Sprintf("%s (commit %s, built %s)", Version, Commit, Date)
}
