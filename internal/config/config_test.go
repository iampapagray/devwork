package config

import (
	"os"
	"path/filepath"
	"testing"
)

// withGlobalDir points config at a temp XDG_CONFIG_HOME for the test.
func withGlobalDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	return filepath.Join(dir, "devwork")
}

func TestLoadDefaultsWhenNothingPresent(t *testing.T) {
	withGlobalDir(t)
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Repo.Template != DefaultTemplate {
		t.Errorf("template = %q, want default", cfg.Repo.Template)
	}
	if cfg.Repo.Slug.MaxWords != 6 || cfg.Repo.Slug.MaxChars != 50 {
		t.Errorf("slug defaults not applied: %+v", cfg.Repo.Slug)
	}
}

func TestRepoConfigMergePreservesDefaults(t *testing.T) {
	withGlobalDir(t)
	repoDir := t.TempDir()
	// Only override the template; slug rules should keep defaults.
	if err := os.WriteFile(filepath.Join(repoDir, ".devwork.toml"),
		[]byte("provider = \"jira\"\ntemplate = \"v{version}/{key}_{slug}\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(repoDir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Repo.Template != "v{version}/{key}_{slug}" {
		t.Errorf("template = %q", cfg.Repo.Template)
	}
	if cfg.Repo.Provider != "jira" {
		t.Errorf("provider = %q", cfg.Repo.Provider)
	}
	if cfg.Repo.Slug.MaxWords != 6 {
		t.Errorf("slug max_words = %d, want default 6", cfg.Repo.Slug.MaxWords)
	}
}

func TestFindRepoConfigWalksUp(t *testing.T) {
	withGlobalDir(t)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".devwork.toml"), []byte("provider=\"github\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(nested)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Repo.Provider != "github" {
		t.Errorf("did not find parent .devwork.toml; provider = %q", cfg.Repo.Provider)
	}
}

func TestGlobalLoadAndEnvOverride(t *testing.T) {
	dir := withGlobalDir(t)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	global := "[providers.jira]\nbase_url=\"https://acme.atlassian.net\"\nemail=\"me@acme.com\"\ntoken=\"filetoken\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(global), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEVWORK_JIRA_TOKEN", "envtoken")

	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	jira := cfg.Global.Providers["jira"]
	if jira.BaseURL != "https://acme.atlassian.net" || jira.Email != "me@acme.com" {
		t.Errorf("global jira not loaded: %+v", jira)
	}
	if jira.Token != "envtoken" {
		t.Errorf("env override not applied: token = %q", jira.Token)
	}
}

func TestEnsureGlobalWritesTemplate(t *testing.T) {
	dir := withGlobalDir(t)
	err := EnsureGlobal()
	nce, ok := err.(*NotConfiguredError)
	if !ok || !nce.Created {
		t.Fatalf("expected NotConfiguredError(Created), got %v", err)
	}
	path := filepath.Join(dir, "config.toml")
	info, statErr := os.Stat(path)
	if statErr != nil {
		t.Fatalf("template not written: %v", statErr)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("template perm = %o, want 600", perm)
	}
	// Second call: file exists, returns nil.
	if err := EnsureGlobal(); err != nil {
		t.Errorf("second EnsureGlobal returned %v, want nil", err)
	}
}
