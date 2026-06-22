// Package git wraps the git operations devwork needs. Side-effecting calls
// shell out to the git binary; the remote-URL parsing is pure and tested
// without a real repo.
package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// Repo is a handle to a working tree rooted at Root.
type Repo struct {
	Root string
}

func run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Open finds the repository root containing dir.
func Open(dir string) (*Repo, error) {
	root, err := run("-C", dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, fmt.Errorf("not inside a git repository: %w", err)
	}
	return &Repo{Root: root}, nil
}

// CurrentBranch returns the abbreviated current branch (or "HEAD" when detached).
func (r *Repo) CurrentBranch() (string, error) {
	return run("-C", r.Root, "rev-parse", "--abbrev-ref", "HEAD")
}

// RemoteURL returns the URL of the named remote (e.g. "origin").
func (r *Repo) RemoteURL(remote string) (string, error) {
	return run("-C", r.Root, "remote", "get-url", remote)
}

// BranchExists reports whether a local branch already exists.
func (r *Repo) BranchExists(name string) bool {
	err := exec.Command("git", "-C", r.Root, "show-ref", "--verify", "--quiet", "refs/heads/"+name).Run()
	return err == nil
}

// CreateBranch creates and switches to name. When base is non-empty the branch
// is started from base; otherwise from the current branch.
func (r *Repo) CreateBranch(name, base string) error {
	args := []string{"-C", r.Root, "switch", "-c", name}
	if base != "" {
		args = append(args, base)
	}
	_, err := run(args...)
	return err
}

// Push pushes name to origin and sets upstream tracking.
func (r *Repo) Push(name string) error {
	_, err := run("-C", r.Root, "push", "-u", "origin", name)
	return err
}

// DefaultBranch resolves origin's default branch (the {default} sentinel),
// e.g. "main", from origin/HEAD.
func (r *Repo) DefaultBranch() (string, error) {
	out, err := run("-C", r.Root, "symbolic-ref", "refs/remotes/origin/HEAD")
	if err != nil {
		return "", err
	}
	// out looks like "refs/remotes/origin/main"
	if i := strings.LastIndex(out, "/"); i >= 0 {
		return out[i+1:], nil
	}
	return out, nil
}

// DescribeTag returns the most recent tag (git describe --tags --abbrev=0),
// used by the "git-tag" version source.
func (r *Repo) DescribeTag() (string, error) {
	return run("-C", r.Root, "describe", "--tags", "--abbrev=0")
}

// Remote identifies the host and owner/repo parsed from a git remote URL.
type Remote struct {
	Host  string // e.g. "github.com", "gitlab.com", "gitlab.acme.com"
	Owner string // owner / namespace (may contain subgroups for GitLab)
	Repo  string // repository name, without ".git"
}

// ParseRemoteURL parses SSH ("git@host:owner/repo.git"), scp-like, and HTTP(S)
// remote URLs into a Remote. It supports GitLab subgroups in Owner.
func ParseRemoteURL(raw string) (Remote, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Remote{}, fmt.Errorf("empty remote URL")
	}

	var host, path string
	switch {
	case strings.HasPrefix(raw, "git@") || (strings.Contains(raw, "@") && strings.Contains(raw, ":") && !strings.Contains(raw, "://")):
		// scp-like: [user@]host:owner/repo(.git)
		at := strings.Index(raw, "@")
		rest := raw[at+1:]
		colon := strings.Index(rest, ":")
		if colon < 0 {
			return Remote{}, fmt.Errorf("unrecognized remote URL: %s", raw)
		}
		host = rest[:colon]
		path = rest[colon+1:]
	case strings.Contains(raw, "://"):
		// scheme://[user@]host[:port]/owner/repo(.git)
		rest := raw[strings.Index(raw, "://")+3:]
		if at := strings.Index(rest, "@"); at >= 0 {
			rest = rest[at+1:]
		}
		slash := strings.Index(rest, "/")
		if slash < 0 {
			return Remote{}, fmt.Errorf("unrecognized remote URL: %s", raw)
		}
		host = rest[:slash]
		if colon := strings.Index(host, ":"); colon >= 0 {
			host = host[:colon] // strip :port
		}
		path = rest[slash+1:]
	default:
		return Remote{}, fmt.Errorf("unrecognized remote URL: %s", raw)
	}

	path = strings.TrimPrefix(path, "/")
	path = strings.TrimSuffix(path, ".git")
	path = strings.TrimSuffix(path, "/")
	if path == "" || host == "" {
		return Remote{}, fmt.Errorf("could not parse owner/repo from: %s", raw)
	}

	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return Remote{}, fmt.Errorf("remote URL missing owner/repo: %s", raw)
	}
	return Remote{Host: host, Owner: path[:idx], Repo: path[idx+1:]}, nil
}
