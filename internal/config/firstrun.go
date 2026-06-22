package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// exampleConfig is the template written on first run. Kept in sync with the
// repo-root config.example.toml (which docs reference); embedding it here means
// the binary is self-contained and needs no install-time asset.
const exampleConfig = `# devwork global config — fill in a provider, then re-run.
# This file holds secrets; it is created with 0600 permissions.

# default_profile = "work"   # (phase 2) multi-profile selection

[providers.jira]
base_url = "https://your-org.atlassian.net"
email    = "you@example.com"
# Create a token: https://id.atlassian.com/manage-profile/security/api-tokens
token    = ""   # or set DEVWORK_JIRA_TOKEN

[providers.github]
# or set DEVWORK_GITHUB_TOKEN / GITHUB_TOKEN, or rely on the gh CLI
token = ""

[providers.gitlab]
base_url = "https://gitlab.com"
token    = ""   # or set DEVWORK_GITLAB_TOKEN
`

// NotConfiguredError signals that no usable credentials were found and a
// template has been written for the user to fill in.
type NotConfiguredError struct {
	Path    string
	Created bool // true if this call wrote the template
}

func (e *NotConfiguredError) Error() string {
	if e.Created {
		return fmt.Sprintf("no credentials yet — template written to %s; fill in a provider and re-run", e.Path)
	}
	return fmt.Sprintf("no credentials configured in %s; fill in a provider and re-run", e.Path)
}

// EnsureGlobal writes the template config if the global file does not exist,
// returning a *NotConfiguredError so the caller can print guidance and exit.
// If the file already exists it returns nil and leaves it untouched.
func EnsureGlobal() error {
	path, err := GlobalPath()
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	if err := os.WriteFile(path, []byte(exampleConfig), 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return &NotConfiguredError{Path: path, Created: true}
}
