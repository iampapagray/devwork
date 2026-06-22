// Command devwork creates a git branch from a tracker issue.
package main

import (
	"os"

	"github.com/iampapagray/devwork/internal/cli"

	// Blank-import the providers so their init() registers them. Adding a
	// provider is a new package plus one line here.
	_ "github.com/iampapagray/devwork/internal/provider/github"
	_ "github.com/iampapagray/devwork/internal/provider/gitlab"
	_ "github.com/iampapagray/devwork/internal/provider/jira"
)

// Injected by GoReleaser via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(cli.Execute(cli.BuildInfo{Version: version, Commit: commit, Date: date}))
}
