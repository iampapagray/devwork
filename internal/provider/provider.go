// Package provider defines the Provider interface every tracker adapter
// implements, plus a package-level registry. Adapters live in subpackages and
// register themselves in init(); cmd blank-imports them. Adding a provider is a
// new package + one import line, with no central switch to edit.
package provider

import (
	"context"

	"github.com/iampapagray/devwork/internal/creds"
	"github.com/iampapagray/devwork/internal/git"
	"github.com/iampapagray/devwork/internal/issue"
)

// Confidence ranks how strongly an input looks like it belongs to a provider.
type Confidence int

const (
	// NoMatch: the input is definitely not for this provider.
	NoMatch Confidence = iota
	// Weak: a shape heuristic matched but is ambiguous (e.g. bare "123").
	Weak
	// Strong: an unambiguous signal matched (e.g. the provider's host in a URL).
	Strong
)

// MatchResult reports a provider's confidence that an input belongs to it.
type MatchResult struct {
	Confidence Confidence
	Reason     string
}

// RepoContext carries repo-derived hints used during Resolve.
type RepoContext struct {
	DefaultProvider string     // per-repo default provider for bare ids
	Remote          git.Remote // parsed origin remote (zero value if none)
	HasRemote       bool
}

// Provider is a tracker adapter. Resolve is pure (parsing + owner/repo
// inference, unit-testable, and where --dry-run can stop); Fetch performs the
// network call.
type Provider interface {
	Name() string
	Match(input string) MatchResult
	Resolve(ctx context.Context, input string, repo RepoContext) (issue.IssueRef, error)
	Fetch(ctx context.Context, ref issue.IssueRef, c creds.Credentials) (issue.Issue, error)
}
