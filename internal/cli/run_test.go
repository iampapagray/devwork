package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iampapagray/devwork/internal/config"
	"github.com/iampapagray/devwork/internal/creds"
	"github.com/iampapagray/devwork/internal/git"
	"github.com/iampapagray/devwork/internal/issue"
	"github.com/iampapagray/devwork/internal/provider"
)

// The stub provider is registered once; tests configure its canned response
// through these package-level vars.
var (
	stubTitle = "Fix login redirect"
	stubVer   *issue.Version
)

func init() { provider.Register(stubProvider{}) }

type stubProvider struct{}

func (stubProvider) Name() string { return "stub" }
func (stubProvider) Match(string) provider.MatchResult {
	return provider.MatchResult{Confidence: provider.Strong}
}
func (stubProvider) Resolve(_ context.Context, in string, _ provider.RepoContext) (issue.IssueRef, error) {
	return issue.IssueRef{Provider: "stub", Key: strings.ToUpper(in)}, nil
}
func (stubProvider) Fetch(context.Context, issue.IssueRef, creds.Credentials) (issue.Issue, error) {
	return issue.Issue{Provider: "stub", Key: "TASK-1", Title: stubTitle, Version: stubVer}, nil
}

type nopResolver struct{}

func (nopResolver) Resolve(string, config.ProviderConfig) (creds.Credentials, error) {
	return creds.Credentials{}, nil
}

func initRepo(t *testing.T, devworkTOML string) string {
	t.Helper()
	dir := t.TempDir()
	cmds := [][]string{
		{"-C", dir, "init", "-q"},
		{"-C", dir, "config", "user.email", "t@example.com"},
		{"-C", dir, "config", "user.name", "t"},
		{"-C", dir, "commit", "--allow-empty", "-q", "-m", "init"},
	}
	for _, args := range cmds {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	if devworkTOML != "" {
		if err := os.WriteFile(filepath.Join(dir, ".devwork.toml"), []byte(devworkTOML), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func newRunner(in string) (*Runner, *bytes.Buffer) {
	useColor = false
	var errBuf bytes.Buffer
	return &Runner{
		Out:      &bytes.Buffer{},
		Err:      &errBuf,
		In:       strings.NewReader(in),
		Resolver: nopResolver{},
	}, &errBuf
}

func TestRunCreatesBranch(t *testing.T) {
	stubTitle, stubVer = "Fix login redirect", nil
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := initRepo(t, "provider = \"stub\"\ntemplate = \"{key}_{slug}\"\n")
	r, errBuf := newRunner("y\n")

	if err := r.Run(context.Background(), dir, Options{Task: "task-1", AssumeYes: true}); err != nil {
		t.Fatalf("run: %v\n%s", err, errBuf.String())
	}
	repo, _ := git.Open(dir)
	if !repo.BranchExists("TASK-1_fix-login-redirect") {
		t.Errorf("branch not created; output:\n%s", errBuf.String())
	}
}

func TestRunInteractiveEditSlug(t *testing.T) {
	stubTitle, stubVer = "Fix login redirect", nil
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := initRepo(t, "provider = \"stub\"\n")
	// edit the slug, then accept.
	r, errBuf := newRunner("e\ncustom branch name\ny\n")

	if err := r.Run(context.Background(), dir, Options{Task: "task-1"}); err != nil {
		t.Fatalf("run: %v\n%s", err, errBuf.String())
	}
	repo, _ := git.Open(dir)
	if !repo.BranchExists("TASK-1_custom-branch-name") {
		t.Errorf("edited branch not created; output:\n%s", errBuf.String())
	}
}

func TestRunAbortOnN(t *testing.T) {
	stubTitle, stubVer = "Add dark mode", nil
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := initRepo(t, "provider = \"stub\"\n")
	r, _ := newRunner("n\n")
	if err := r.Run(context.Background(), dir, Options{Task: "task-1"}); err != nil {
		t.Fatal(err)
	}
	repo, _ := git.Open(dir)
	if repo.BranchExists("TASK-1_add-dark-mode") {
		t.Error("answering n should not create a branch")
	}
}

func TestRunDryRunDoesNotCreate(t *testing.T) {
	stubTitle, stubVer = "Add dark mode", nil
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := initRepo(t, "provider = \"stub\"\n")
	r, errBuf := newRunner("")
	if err := r.Run(context.Background(), dir, Options{Task: "task-1", DryRun: true}); err != nil {
		t.Fatal(err)
	}
	repo, _ := git.Open(dir)
	if repo.BranchExists("TASK-1_add-dark-mode") {
		t.Error("dry-run should not create a branch")
	}
	if !strings.Contains(errBuf.String(), "dry run") {
		t.Errorf("expected dry-run notice; got:\n%s", errBuf.String())
	}
}

func TestRunVersionGateAheadCreatesWithIssueVersion(t *testing.T) {
	v := issue.ParseVersion("7.10")
	stubTitle, stubVer = "Bump things", &v
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := initRepo(t, "provider = \"stub\"\ntemplate = \"v{version}/{key}_{slug}\"\n[version]\nsource = \"package.json\"\n")
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"version":"7.9.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r, errBuf := newRunner("")
	if err := r.Run(context.Background(), dir, Options{Task: "task-1", AssumeYes: true}); err != nil {
		t.Fatalf("run: %v\n%s", err, errBuf.String())
	}
	repo, _ := git.Open(dir)
	if !repo.BranchExists("v7.10/TASK-1_bump-things") {
		t.Errorf("expected issue version prefix; output:\n%s", errBuf.String())
	}
}

func TestRunVersionGateStrictAborts(t *testing.T) {
	stubTitle, stubVer = "x", nil
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := initRepo(t, "provider = \"stub\"\ntemplate = \"v{version}/{key}_{slug}\"\n[version]\nsource = \"package.json\"\nmismatch = \"strict\"\n")
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"version":"7.3.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r, _ := newRunner("")
	err := r.Run(context.Background(), dir, Options{Task: "task-1", AssumeYes: true})
	if err == nil || !strings.Contains(err.Error(), "no version") {
		t.Fatalf("expected hard error for missing issue version, got %v", err)
	}
}
