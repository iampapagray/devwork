package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/iampapagray/devwork/internal/branch"
	"github.com/iampapagray/devwork/internal/config"
	"github.com/iampapagray/devwork/internal/creds"
	"github.com/iampapagray/devwork/internal/git"
	"github.com/iampapagray/devwork/internal/issue"
	"github.com/iampapagray/devwork/internal/router"
	"github.com/iampapagray/devwork/internal/slug"
)

// Options are the resolved flags/args for a run.
type Options struct {
	Task      string // task input (flag or positional)
	Provider  string // --provider
	From      string // --from base branch override
	AssumeYes bool   // --yes
	DryRun    bool   // --dry-run
	Push      bool   // --push
}

// Runner wires the run loop to its IO and credential resolver so it can be
// driven in tests.
type Runner struct {
	Out      io.Writer
	Err      io.Writer
	In       io.Reader
	Resolver creds.Resolver
}

// Run executes the full flow: load config, select+resolve+fetch the issue, run
// the version gate, build and confirm the branch, then create (and optionally
// push) it.
func (r *Runner) Run(ctx context.Context, cwd string, opts Options) error {
	if strings.TrimSpace(opts.Task) == "" {
		return fmt.Errorf("no task given (try: devwork PROJ-123, or --help)")
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		return err
	}

	repo, err := git.Open(cwd)
	if err != nil {
		return err
	}
	currentBranch, _ := repo.CurrentBranch()
	repoCtx := buildRepoContext(repo, cfg)

	// Select the provider (plan §2).
	p, err := router.Select(opts.Task, router.Inputs{
		FlagProvider:    opts.Provider,
		DefaultProvider: cfg.Repo.Provider,
		Hosts:           providerHosts(cfg),
		Configured:      configuredProviders(cfg),
	})
	if err != nil {
		return err
	}

	ref, err := p.Resolve(ctx, opts.Task, repoCtx)
	if err != nil {
		return err
	}

	c, err := r.Resolver.Resolve(p.Name(), cfg.Global.Providers[p.Name()])
	if err != nil {
		return err
	}

	fmt.Fprintf(r.Err, dim("Fetching %s from %s…")+"\n", ref.Key, p.Name())
	iss, err := p.Fetch(ctx, ref, c)
	if err != nil {
		return err
	}

	sl := slug.New(cfg.Repo.Slug.MaxWords, cfg.Repo.Slug.MaxChars, cfg.Repo.Slug.Stopwords)
	slugStr := sl.Slugify(iss.Title)

	// Version gate (decision C).
	gateActive := branch.HasVersionToken(cfg.Repo.Template)
	var gate branch.GateResult
	if gateActive {
		gate, err = r.runGate(repo, cfg, iss)
		if err != nil {
			return err
		}
		if gate.Abort {
			return fmt.Errorf("version mismatch (strict): %s", gate.Message)
		}
	}

	makeBranch := func(slugStr string) (string, error) {
		fields := map[string]string{
			"key":      iss.Key,
			"slug":     slugStr,
			"provider": p.Name(),
		}
		if gateActive {
			fields["version"] = gate.Version
		}
		return branch.Render(cfg.Repo.Template, fields)
	}

	branchName, err := makeBranch(slugStr)
	if err != nil {
		return err
	}

	r.printPlan(currentBranch, iss, gateActive, gate, branchName)
	if repo.BranchExists(branchName) {
		return fmt.Errorf("branch already exists: %s", branchName)
	}

	if opts.DryRun {
		fmt.Fprintln(r.Err, dim("dry run — no branch created"))
		return nil
	}

	if !opts.AssumeYes {
		ok, newName, err := r.confirm(currentBranch, slugStr, sl.Slugify, makeBranch)
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(r.Err, "aborted.")
			return nil
		}
		branchName = newName
		if repo.BranchExists(branchName) {
			return fmt.Errorf("branch already exists: %s", branchName)
		}
	}

	base, err := r.resolveBase(repo, cfg, opts)
	if err != nil {
		return err
	}
	if err := repo.CreateBranch(branchName, base); err != nil {
		return err
	}
	fmt.Fprintf(r.Err, green("✓ created and switched to")+" %s\n", branchName)

	if opts.Push {
		if err := repo.Push(branchName); err != nil {
			return err
		}
		fmt.Fprintf(r.Err, green("✓ pushed")+" %s -u origin\n", branchName)
	}
	return nil
}

func (r *Runner) runGate(repo *git.Repo, cfg *config.Config, iss issue.Issue) (branch.GateResult, error) {
	if iss.Version == nil {
		return branch.GateResult{}, fmt.Errorf("template uses {version} but %s has no version (fixVersion/milestone) — set one, or drop {version} from the template", iss.Key)
	}
	repoVer, err := branch.ResolveRepoVersion(repo.Root, cfg.Repo.Version.Source, func() (string, error) {
		return repo.DescribeTag()
	})
	if err != nil {
		return branch.GateResult{}, fmt.Errorf("template uses {version} but the repo version is unresolvable: %w", err)
	}
	if !repoVer.OK() {
		return branch.GateResult{}, fmt.Errorf("template uses {version} but could not parse a MAJOR.MINOR repo version from source %q", cfg.Repo.Version.Source)
	}
	strict := cfg.Repo.Version.Mismatch == "strict"
	return branch.Gate(*iss.Version, repoVer, strict), nil
}

// confirm runs the interactive [Y/e/n] loop, re-slugging edited input.
func (r *Runner) confirm(currentBranch, slugStr string, reslug func(string) string, makeBranch func(string) (string, error)) (bool, string, error) {
	reader := bufio.NewReader(r.In)
	name, err := makeBranch(slugStr)
	if err != nil {
		return false, "", err
	}
	for {
		fmt.Fprintf(r.Err, yellow("Create this branch from %s? [Y/e/n] "), currentBranch)
		line, err := reader.ReadString('\n')
		if err != nil && line == "" {
			return false, "", fmt.Errorf("no input")
		}
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "", "y":
			return true, name, nil
		case "n":
			return false, "", nil
		case "e":
			fmt.Fprintf(r.Err, "edit slug [%s]: ", slugStr)
			edit, _ := reader.ReadString('\n')
			edit = strings.TrimSpace(edit)
			if edit != "" {
				slugStr = reslug(edit)
			}
			name, err = makeBranch(slugStr)
			if err != nil {
				return false, "", err
			}
			fmt.Fprintf(r.Err, "  %s %s\n", dim("New branch  "), cyan(name))
		default:
			fmt.Fprintln(r.Err, "please answer Y, e, or n.")
		}
	}
}

// resolveBase picks the base branch: --from > per-repo base_branch > current.
func (r *Runner) resolveBase(repo *git.Repo, cfg *config.Config, opts Options) (string, error) {
	base := opts.From
	if base == "" {
		base = cfg.Repo.BaseBranch
	}
	if base == "{default}" {
		def, err := repo.DefaultBranch()
		if err != nil {
			return "", fmt.Errorf("could not resolve {default} base branch: %w", err)
		}
		return def, nil
	}
	return base, nil // "" means current branch
}
