package creds

import (
	"testing"

	"github.com/iampapagray/devwork/internal/config"
)

func TestResolveJira(t *testing.T) {
	r := EnvFileResolver{}
	_, err := r.Resolve("jira", config.ProviderConfig{Token: "t"})
	if err == nil {
		t.Error("expected error for missing email/base_url")
	}
	c, err := r.Resolve("jira", config.ProviderConfig{
		Token: "t", Email: "me@acme.com", BaseURL: "https://acme.atlassian.net/",
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL != "https://acme.atlassian.net" {
		t.Errorf("base_url not trimmed: %q", c.BaseURL)
	}
}

func TestResolveGitHubFallbacks(t *testing.T) {
	r := EnvFileResolver{}

	// gh CLI fallback when nothing else set.
	old := ghToken
	defer func() { ghToken = old }()
	ghToken = func() string { return "ghcli" }

	c, err := r.Resolve("github", config.ProviderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if c.Token != "ghcli" {
		t.Errorf("expected gh CLI token, got %q", c.Token)
	}

	// GITHUB_TOKEN beats gh CLI.
	t.Setenv("GITHUB_TOKEN", "envtok")
	c, err = r.Resolve("github", config.ProviderConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if c.Token != "envtok" {
		t.Errorf("expected GITHUB_TOKEN, got %q", c.Token)
	}

	// Config/DEVWORK token (already in cfg.Token) beats everything.
	c, err = r.Resolve("github", config.ProviderConfig{Token: "cfgtok"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Token != "cfgtok" {
		t.Errorf("expected config token, got %q", c.Token)
	}
}

func TestResolveGitHubMissing(t *testing.T) {
	r := EnvFileResolver{}
	old := ghToken
	defer func() { ghToken = old }()
	ghToken = func() string { return "" }
	t.Setenv("GITHUB_TOKEN", "")
	if _, err := r.Resolve("github", config.ProviderConfig{}); err == nil {
		t.Error("expected error when no GitHub token available")
	}
}

func TestResolveGitLabDefaultsBaseURL(t *testing.T) {
	c, err := EnvFileResolver{}.Resolve("gitlab", config.ProviderConfig{Token: "t"})
	if err != nil {
		t.Fatal(err)
	}
	if c.BaseURL != "https://gitlab.com" {
		t.Errorf("default base_url = %q", c.BaseURL)
	}
}
