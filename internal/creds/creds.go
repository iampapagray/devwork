// Package creds resolves provider credentials from the config file and the
// environment, behind a CredentialResolver interface so an OS-keychain
// resolver can be added later without touching call sites.
package creds

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/iampapagray/devwork/internal/config"
)

func envToken(name string) string { return strings.TrimSpace(os.Getenv(name)) }

// Credentials are the resolved secrets/coordinates for a single provider.
type Credentials struct {
	Token   string
	Email   string
	BaseURL string
}

// Resolver turns a provider's config section into usable Credentials.
type Resolver interface {
	Resolve(provider string, cfg config.ProviderConfig) (Credentials, error)
}

// EnvFileResolver is the v1 resolver: it trusts the config file (which already
// has DEVWORK_*_TOKEN env overrides applied by the config package) and adds the
// provider-specific fallbacks documented in the plan.
type EnvFileResolver struct{}

// ghToken is overridable in tests to avoid shelling out to the gh CLI.
var ghToken = ghTokenFromCLI

func (EnvFileResolver) Resolve(provider string, cfg config.ProviderConfig) (Credentials, error) {
	c := Credentials{
		Token:   cfg.Token,
		Email:   cfg.Email,
		BaseURL: strings.TrimRight(cfg.BaseURL, "/"),
	}

	switch provider {
	case "github":
		// Order: DEVWORK_GITHUB_TOKEN (already in cfg.Token via config) ->
		// GITHUB_TOKEN -> gh CLI token -> config file (already in cfg.Token).
		if c.Token == "" {
			if t := envToken("GITHUB_TOKEN"); t != "" {
				c.Token = t
			} else if t := ghToken(); t != "" {
				c.Token = t
			}
		}
		if c.Token == "" {
			return c, fmt.Errorf("no GitHub token: set DEVWORK_GITHUB_TOKEN, GITHUB_TOKEN, authenticate the gh CLI, or fill [providers.github].token")
		}

	case "jira":
		if c.Token == "" {
			return c, fmt.Errorf("no Jira token: set DEVWORK_JIRA_TOKEN or fill [providers.jira].token")
		}
		if c.Email == "" {
			return c, fmt.Errorf("no Jira email: fill [providers.jira].email")
		}
		if c.BaseURL == "" {
			return c, fmt.Errorf("no Jira base_url: fill [providers.jira].base_url")
		}

	case "gitlab":
		if c.Token == "" {
			return c, fmt.Errorf("no GitLab token: set DEVWORK_GITLAB_TOKEN or fill [providers.gitlab].token")
		}
		if c.BaseURL == "" {
			c.BaseURL = "https://gitlab.com"
		}

	default:
		return c, fmt.Errorf("unknown provider %q", provider)
	}

	return c, nil
}

func ghTokenFromCLI() string {
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
