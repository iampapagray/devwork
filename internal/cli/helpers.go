package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/iampapagray/devwork/internal/branch"
	"github.com/iampapagray/devwork/internal/config"
	"github.com/iampapagray/devwork/internal/git"
	"github.com/iampapagray/devwork/internal/issue"
	"github.com/iampapagray/devwork/internal/provider"
)

// buildRepoContext gathers the git/remote hints providers use during Resolve.
func buildRepoContext(repo *git.Repo, cfg *config.Config) provider.RepoContext {
	ctx := provider.RepoContext{DefaultProvider: cfg.Repo.Provider}
	if url, err := repo.RemoteURL("origin"); err == nil {
		if rem, err := git.ParseRemoteURL(url); err == nil {
			ctx.Remote = rem
			ctx.HasRemote = true
		}
	}
	return ctx
}

// providerHosts maps provider name -> configured host, for URL host detection.
func providerHosts(cfg *config.Config) map[string]string {
	hosts := map[string]string{}
	for name, pc := range cfg.Global.Providers {
		if h := hostOf(pc.BaseURL); h != "" {
			hosts[name] = h
		}
	}
	return hosts
}

// configuredProviders lists providers that have a config section present.
func configuredProviders(cfg *config.Config) []string {
	var names []string
	for name := range cfg.Global.Providers {
		names = append(names, name)
	}
	return names
}

func hostOf(rawURL string) string {
	i := strings.Index(rawURL, "://")
	if i < 0 {
		return ""
	}
	rest := rawURL[i+3:]
	if slash := strings.IndexAny(rest, "/:"); slash >= 0 {
		rest = rest[:slash]
	}
	return rest
}

func (r *Runner) printPlan(currentBranch string, iss issue.Issue, gateActive bool, gate branch.GateResult, branchName string) {
	fmt.Fprintln(r.Err)
	fmt.Fprintf(r.Err, "  %s %s %s\n", dim("Base branch "), currentBranch, dim("(current)"))
	fmt.Fprintf(r.Err, "  %s %s — %s\n", dim("Task        "), iss.Key, iss.Title)
	if gateActive {
		switch gate.Status {
		case branch.StatusEqual:
			fmt.Fprintf(r.Err, "  %s %s %s\n", dim("Version     "), gate.Version, green("✓ "+gate.Message))
		case branch.StatusAhead:
			fmt.Fprintf(r.Err, "  %s %s %s\n", dim("Version     "), gate.Version, yellow("⚠ "+gate.Message))
		case branch.StatusBehind:
			fmt.Fprintf(r.Err, "  %s %s %s\n", dim("Version     "), gate.Version, yellow("⚠ "+gate.Message))
		}
	}
	fmt.Fprintf(r.Err, "  %s %s\n", dim("New branch  "), cyan(branchName))
	fmt.Fprintln(r.Err)
}

// ── ANSI colors, disabled when NO_COLOR is set or stderr is not a terminal ──

var useColor = colorEnabled()

func colorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	fi, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func wrap(code, s string) string {
	if !useColor {
		return s
	}
	return "\033[" + code + "m" + s + "\033[0m"
}

func dim(s string) string    { return wrap("2", s) }
func red(s string) string    { return wrap("31", s) }
func green(s string) string  { return wrap("32", s) }
func yellow(s string) string { return wrap("33", s) }
func cyan(s string) string   { return wrap("36", s) }
