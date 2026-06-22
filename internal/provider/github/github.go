// Package github implements the GitHub issues adapter. owner/repo are inferred
// from the origin remote unless given explicitly; the branch key is the bare
// issue number (no "#").
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/iampapagray/devwork/internal/creds"
	"github.com/iampapagray/devwork/internal/issue"
	"github.com/iampapagray/devwork/internal/provider"
)

func init() { provider.Register(New()) }

// urlRe matches a GitHub issue/PR URL: .../owner/repo/issues/123.
var urlRe = regexp.MustCompile(`github\.com/([^/]+)/([^/]+)/(?:issues|pull)/(\d+)`)

// ownerRepoRe matches "owner/repo#123".
var ownerRepoRe = regexp.MustCompile(`^([^/\s]+)/([^/#\s]+)#(\d+)$`)

// bareRe matches "#123" or "123".
var bareRe = regexp.MustCompile(`^#?(\d+)$`)

// Provider is the GitHub adapter.
type Provider struct {
	client  *http.Client
	apiBase string // overridable for tests
}

// New returns a GitHub provider talking to api.github.com.
func New() *Provider {
	return &Provider{client: &http.Client{Timeout: 20 * time.Second}, apiBase: "https://api.github.com"}
}

func (*Provider) Name() string { return "github" }

func (*Provider) Match(input string) provider.MatchResult {
	if strings.Contains(input, "github.com") {
		return provider.MatchResult{Confidence: provider.Strong, Reason: "GitHub URL"}
	}
	if bareRe.MatchString(input) {
		return provider.MatchResult{Confidence: provider.Weak, Reason: "issue-number shape"}
	}
	return provider.MatchResult{Confidence: provider.NoMatch}
}

func (*Provider) Resolve(_ context.Context, input string, repo provider.RepoContext) (issue.IssueRef, error) {
	input = strings.TrimSpace(input)

	if m := urlRe.FindStringSubmatch(input); m != nil {
		return issue.IssueRef{Provider: "github", Owner: m[1], Repo: strings.TrimSuffix(m[2], ".git"), Key: m[3]}, nil
	}
	if m := ownerRepoRe.FindStringSubmatch(input); m != nil {
		return issue.IssueRef{Provider: "github", Owner: m[1], Repo: m[2], Key: m[3]}, nil
	}
	if m := bareRe.FindStringSubmatch(input); m != nil {
		owner, repoName, err := inferOwnerRepo(repo)
		if err != nil {
			return issue.IssueRef{}, err
		}
		return issue.IssueRef{Provider: "github", Owner: owner, Repo: repoName, Key: m[1]}, nil
	}
	return issue.IssueRef{}, fmt.Errorf("could not parse a GitHub issue from: %s", input)
}

func inferOwnerRepo(repo provider.RepoContext) (string, string, error) {
	if !repo.HasRemote || repo.Remote.Owner == "" || repo.Remote.Repo == "" {
		return "", "", fmt.Errorf("could not infer owner/repo; use owner/repo#123 or set an origin remote")
	}
	if repo.Remote.Host != "" && repo.Remote.Host != "github.com" {
		return "", "", fmt.Errorf("origin remote host %q is not github.com; use owner/repo#123", repo.Remote.Host)
	}
	return repo.Remote.Owner, repo.Remote.Repo, nil
}

type ghIssue struct {
	Title     string `json:"title"`
	HTMLURL   string `json:"html_url"`
	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
	Message string `json:"message"` // present on error bodies
}

func (p *Provider) Fetch(ctx context.Context, ref issue.IssueRef, c creds.Credentials) (issue.Issue, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%s", strings.TrimRight(p.apiBase, "/"), ref.Owner, ref.Repo, ref.Key)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return issue.Issue{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return issue.Issue{}, fmt.Errorf("could not reach GitHub: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return issue.Issue{}, fmt.Errorf("GitHub auth failed (HTTP %d) — check your token/scopes", resp.StatusCode)
	case http.StatusNotFound:
		return issue.Issue{}, fmt.Errorf("issue %s/%s#%s not found (HTTP 404)", ref.Owner, ref.Repo, ref.Key)
	default:
		return issue.Issue{}, fmt.Errorf("GitHub returned HTTP %d", resp.StatusCode)
	}

	var gi ghIssue
	if err := json.NewDecoder(resp.Body).Decode(&gi); err != nil {
		return issue.Issue{}, fmt.Errorf("decoding GitHub response: %w", err)
	}
	if gi.Title == "" {
		return issue.Issue{}, fmt.Errorf("issue %s/%s#%s has no title", ref.Owner, ref.Repo, ref.Key)
	}

	out := issue.Issue{Provider: "github", Key: ref.Key, Title: gi.Title, URL: gi.HTMLURL}
	if gi.Milestone != nil {
		if v := issue.ParseVersion(gi.Milestone.Title); v.OK() {
			vv := v
			out.Version = &vv
		}
	}
	return out, nil
}
