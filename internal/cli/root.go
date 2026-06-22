// Package cli wires the cobra command tree and drives the run loop.
package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/iampapagray/devwork/internal/config"
	"github.com/iampapagray/devwork/internal/creds"
)

// BuildInfo is injected by GoReleaser via -ldflags.
type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// Execute builds and runs the root command. It returns the process exit code.
func Execute(info BuildInfo) int {
	var opts Options

	root := &cobra.Command{
		Use:   "devwork [TASK]",
		Short: "Create a git branch from a tracker issue (Jira, GitHub, GitLab)",
		Long: "devwork turns a tracker issue into a well-named git branch.\n\n" +
			"  devwork PROJ-123\n" +
			"  devwork https://github.com/acme/widget/issues/42\n" +
			"  devwork --provider gitlab 17 --push\n\n" +
			"Per-repo behavior lives in a committed .devwork.toml; credentials live in\n" +
			"~/.config/devwork/config.toml (or DEVWORK_*_TOKEN env vars).",
		Args:          cobra.MaximumNArgs(1),
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       fmt.Sprintf("%s (commit %s, built %s)", info.Version, info.Commit, info.Date),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && opts.Task == "" {
				opts.Task = args[0]
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			// First-run: write the config template and guide the user.
			if err := config.EnsureGlobal(); err != nil {
				return err
			}
			runner := &Runner{
				Out:      cmd.OutOrStdout(),
				Err:      cmd.ErrOrStderr(),
				In:       cmd.InOrStdin(),
				Resolver: creds.EnvFileResolver{},
			}
			return runner.Run(context.Background(), cwd, opts)
		},
	}

	f := root.Flags()
	f.StringVarP(&opts.Task, "task", "t", "", "task to branch from (issue key, #number, or URL)")
	f.StringVar(&opts.Provider, "provider", "", "force a provider (jira|github|gitlab)")
	f.StringVar(&opts.From, "from", "", "base branch to start from (default: current)")
	f.BoolVarP(&opts.AssumeYes, "yes", "y", false, "skip the confirmation prompt")
	f.BoolVar(&opts.DryRun, "dry-run", false, "resolve and print the branch without creating it")
	f.BoolVar(&opts.Push, "push", false, "push the new branch and set upstream")

	root.SetVersionTemplate("devwork {{.Version}}\n")

	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%s %s\n", red("error:"), err)
		return 1
	}
	return 0
}
