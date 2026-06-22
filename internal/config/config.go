// Package config loads and merges devwork configuration.
//
// Precedence (low -> high): built-in defaults -> global config.toml ->
// per-repo .devwork.toml -> environment variables -> flags. This package owns
// everything up to (and including) the environment layer; the cli package
// overlays flags on top of the Config it returns.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"

	"github.com/iampapagray/devwork/internal/slug"
)

// ProviderConfig is the per-provider section of the global config.
type ProviderConfig struct {
	BaseURL string `toml:"base_url"`
	Email   string `toml:"email"`
	Token   string `toml:"token"`
}

// GlobalConfig mirrors ~/.config/devwork/config.toml (secrets live here).
type GlobalConfig struct {
	DefaultProfile string                    `toml:"default_profile"`
	Providers      map[string]ProviderConfig `toml:"providers"`
}

// VersionConfig is the [version] table of a per-repo config.
type VersionConfig struct {
	Source string `toml:"source"`
	// Mismatch controls a behind/incompatible issue version: "" (default) =
	// warn + confirm; "strict" = hard abort. Only consulted when the version
	// gate is active (template contains {version}).
	Mismatch string `toml:"mismatch"`
}

// SlugConfig is the [slug] table of a per-repo config.
type SlugConfig struct {
	MaxWords  int      `toml:"max_words"`
	MaxChars  int      `toml:"max_chars"`
	Stopwords []string `toml:"stopwords"`
}

// RepoConfig mirrors a committed .devwork.toml (behavior, no secrets).
type RepoConfig struct {
	Provider   string        `toml:"provider"`
	Template   string        `toml:"template"`
	BaseBranch string        `toml:"base_branch"`
	Version    VersionConfig `toml:"version"`
	Slug       SlugConfig    `toml:"slug"`
}

// Config is the effective configuration after merging all layers below flags.
type Config struct {
	Global GlobalConfig
	Repo   RepoConfig

	// GlobalPath is where the global config was (or would be) loaded from.
	GlobalPath string
}

// DefaultTemplate produces a valid branch on any tracker and keeps the version
// gate off (no {version} token), per decision C.
const DefaultTemplate = "{key}_{slug}"

func defaultRepoConfig() RepoConfig {
	return RepoConfig{
		Template: DefaultTemplate,
		Version:  VersionConfig{Source: "package.json"},
		Slug: SlugConfig{
			MaxWords:  slug.DefaultMaxWords,
			MaxChars:  slug.DefaultMaxChars,
			Stopwords: slug.DefaultStopwords,
		},
	}
}

// GlobalDir returns ~/.config/devwork (honoring XDG_CONFIG_HOME).
func GlobalDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "devwork"), nil
}

// GlobalPath returns the path to the global config.toml.
func GlobalPath() (string, error) {
	dir, err := GlobalDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// Load reads the global config and the per-repo .devwork.toml found by walking
// up from startDir, merges them over the built-in defaults, and applies
// environment overrides for credentials. A missing global file is not an
// error here (callers decide whether credentials are required); a missing
// per-repo file simply leaves the defaults in place.
func Load(startDir string) (*Config, error) {
	cfg := &Config{
		Global: GlobalConfig{Providers: map[string]ProviderConfig{}},
		Repo:   defaultRepoConfig(),
	}

	gp, err := GlobalPath()
	if err != nil {
		return nil, err
	}
	cfg.GlobalPath = gp
	if err := loadGlobal(gp, &cfg.Global); err != nil {
		return nil, err
	}

	if rp := findRepoConfig(startDir); rp != "" {
		if err := mergeRepo(rp, &cfg.Repo); err != nil {
			return nil, err
		}
	}

	applyEnvOverrides(&cfg.Global)
	return cfg, nil
}

func loadGlobal(path string, g *GlobalConfig) error {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	if err := toml.Unmarshal(data, g); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	if g.Providers == nil {
		g.Providers = map[string]ProviderConfig{}
	}
	return nil
}

// mergeRepo decodes a .devwork.toml over an already-defaulted RepoConfig so
// that omitted keys keep their default values rather than zeroing out.
func mergeRepo(path string, r *RepoConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}
	md, err := toml.Decode(string(data), r)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}
	// toml zeroes scalars it does set but leaves untouched keys alone since we
	// decoded into a pre-populated struct. Guard against an explicit empty/zero
	// slug config that would disable slugging.
	if !md.IsDefined("slug", "max_words") || r.Slug.MaxWords <= 0 {
		r.Slug.MaxWords = slug.DefaultMaxWords
	}
	if !md.IsDefined("slug", "max_chars") || r.Slug.MaxChars <= 0 {
		r.Slug.MaxChars = slug.DefaultMaxChars
	}
	if !md.IsDefined("slug", "stopwords") {
		r.Slug.Stopwords = slug.DefaultStopwords
	}
	if !md.IsDefined("template") || r.Template == "" {
		r.Template = DefaultTemplate
	}
	return nil
}

// findRepoConfig walks up from dir looking for a .devwork.toml, stopping at the
// filesystem root. Returns "" when none is found.
func findRepoConfig(dir string) string {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, ".devwork.toml")
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// applyEnvOverrides lets DEVWORK_*_TOKEN env vars override the file tokens.
// Provider-specific credential resolution (including GITHUB_TOKEN / gh CLI
// fallback) lives in the creds package; this is the coarse file-level override.
func applyEnvOverrides(g *GlobalConfig) {
	override := func(name, env string) {
		if v := os.Getenv(env); v != "" {
			pc := g.Providers[name]
			pc.Token = v
			g.Providers[name] = pc
		}
	}
	override("jira", "DEVWORK_JIRA_TOKEN")
	override("github", "DEVWORK_GITHUB_TOKEN")
	override("gitlab", "DEVWORK_GITLAB_TOKEN")
}
