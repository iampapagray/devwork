// Package gitlab implements the GitLab issues adapter. The project is the
// namespace/path inferred from the origin remote (or given explicitly), and the
// branch key is the bare issue iid.
package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/iampapagray/devwork/internal/creds"
	"github.com/iampapagray/devwork/internal/issue"
	"github.com/iampapagray/devwork/internal/provider"
)

func init() { provider.Register(New()) }

// urlRe matches a GitLab issue URL: host/group[/sub]/proj/-/issues/123.
var urlRe = regexp.MustCompile(`(gitlab\.[^/]+|[^/]*gitlab[^/]*)/(.+?)/-/issues/(\d+)`)

// pathRe matches "group/proj#123" (subgroups allowed in the namespace).
var pathRe = regexp.MustCompile(`^(.+/[^/#]+)#(\d+)$`)

var bareRe = regexp.MustCompile(`^#?(\d+)$`)

// Provider is the GitLab adapter.
type Provider struct {
	client *http.Client
}

// New returns a GitLab provider.
func New() *Provider {
	return &Provider{client: &http.Client{Timeout: 20 * time.Second}}
}

func (*Provider) Name() string { return "gitlab" }

func (*Provider) Match(input string) provider.MatchResult {
	if strings.Contains(input, "gitlab.") {
		return provider.MatchResult{Confidence: provider.Strong, Reason: "GitLab URL"}
	}
	if bareRe.MatchString(input) {
		return provider.MatchResult{Confidence: provider.Weak, Reason: "issue-iid shape"}
	}
	return provider.MatchResult{Confidence: provider.NoMatch}
}

func (*Provider) Resolve(_ context.Context, input string, repo provider.RepoContext) (issue.IssueRef, error) {
	input = strings.TrimSpace(input)

	if m := urlRe.FindStringSubmatch(input); m != nil {
		return issue.IssueRef{Provider: "gitlab", Project: m[2], Key: m[3], BaseURL: hostBase(input)}, nil
	}
	if m := pathRe.FindStringSubmatch(input); m != nil {
		return issue.IssueRef{Provider: "gitlab", Project: m[1], Key: m[2]}, nil
	}
	if m := bareRe.FindStringSubmatch(input); m != nil {
		proj, err := inferProject(repo)
		if err != nil {
			return issue.IssueRef{}, err
		}
		return issue.IssueRef{Provider: "gitlab", Project: proj, Key: m[1]}, nil
	}
	return issue.IssueRef{}, fmt.Errorf("could not parse a GitLab issue from: %s", input)
}

func inferProject(repo provider.RepoContext) (string, error) {
	if !repo.HasRemote || repo.Remote.Owner == "" || repo.Remote.Repo == "" {
		return "", fmt.Errorf("could not infer project; use group/project#123 or set an origin remote")
	}
	return repo.Remote.Owner + "/" + repo.Remote.Repo, nil
}

// hostBase pulls scheme://host from a full URL for self-hosted instances.
func hostBase(input string) string {
	i := strings.Index(input, "://")
	if i < 0 {
		return ""
	}
	scheme := input[:i]
	rest := input[i+3:]
	if slash := strings.Index(rest, "/"); slash >= 0 {
		return scheme + "://" + rest[:slash]
	}
	return scheme + "://" + rest
}

type glIssue struct {
	Title     string `json:"title"`
	WebURL    string `json:"web_url"`
	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
}

func (p *Provider) Fetch(ctx context.Context, ref issue.IssueRef, c creds.Credentials) (issue.Issue, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	if ref.BaseURL != "" {
		base = strings.TrimRight(ref.BaseURL, "/") // URL input pins the host
	}
	if base == "" {
		base = "https://gitlab.com"
	}
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s/issues/%s", base, url.PathEscape(ref.Project), ref.Key)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return issue.Issue{}, err
	}
	req.Header.Set("PRIVATE-TOKEN", c.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return issue.Issue{}, fmt.Errorf("could not reach GitLab: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return issue.Issue{}, fmt.Errorf("GitLab auth failed (HTTP %d) — check your token/scopes", resp.StatusCode)
	case http.StatusNotFound:
		return issue.Issue{}, fmt.Errorf("issue %s#%s not found (HTTP 404)", ref.Project, ref.Key)
	default:
		return issue.Issue{}, fmt.Errorf("GitLab returned HTTP %d", resp.StatusCode)
	}

	var gi glIssue
	if err := json.NewDecoder(resp.Body).Decode(&gi); err != nil {
		return issue.Issue{}, fmt.Errorf("decoding GitLab response: %w", err)
	}
	if gi.Title == "" {
		return issue.Issue{}, fmt.Errorf("issue %s#%s has no title", ref.Project, ref.Key)
	}

	out := issue.Issue{Provider: "gitlab", Key: ref.Key, Title: gi.Title, URL: gi.WebURL}
	if gi.Milestone != nil {
		if v := issue.ParseVersion(gi.Milestone.Title); v.OK() {
			vv := v
			out.Version = &vv
		}
	}
	return out, nil
}
