// Package jira implements the Jira tracker adapter. It is a port of the bash
// reference: extract an issue key, fetch summary + fixVersions from the v3 REST
// API, and normalize into issue.Issue.
package jira

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

// keyRe matches a Jira issue key like "QA-2840" (case-insensitive on input).
var keyRe = regexp.MustCompile(`[A-Za-z][A-Za-z0-9]+-[0-9]+`)

// Provider is the Jira adapter.
type Provider struct {
	client *http.Client
}

// New returns a Jira provider with a default HTTP client.
func New() *Provider {
	return &Provider{client: &http.Client{Timeout: 20 * time.Second}}
}

func (*Provider) Name() string { return "jira" }

func (*Provider) Match(input string) provider.MatchResult {
	if strings.Contains(input, "atlassian.net") || strings.Contains(input, "/browse/") {
		return provider.MatchResult{Confidence: provider.Strong, Reason: "Jira URL"}
	}
	if keyRe.MatchString(input) && strings.Contains(input, "-") {
		return provider.MatchResult{Confidence: provider.Strong, Reason: "Jira issue key shape"}
	}
	return provider.MatchResult{Confidence: provider.NoMatch}
}

// ExtractKey pulls the first Jira key out of input and upper-cases it.
func ExtractKey(input string) (string, bool) {
	m := keyRe.FindString(input)
	if m == "" {
		return "", false
	}
	return strings.ToUpper(m), true
}

func (*Provider) Resolve(_ context.Context, input string, _ provider.RepoContext) (issue.IssueRef, error) {
	key, ok := ExtractKey(input)
	if !ok {
		return issue.IssueRef{}, fmt.Errorf("could not find a Jira issue key (like QA-2840) in: %s", input)
	}
	return issue.IssueRef{Provider: "jira", Key: key}, nil
}

type jiraIssue struct {
	Fields struct {
		Summary     string `json:"summary"`
		FixVersions []struct {
			Name string `json:"name"`
		} `json:"fixVersions"`
	} `json:"fields"`
}

type jiraError struct {
	ErrorMessages []string `json:"errorMessages"`
}

func (p *Provider) Fetch(ctx context.Context, ref issue.IssueRef, c creds.Credentials) (issue.Issue, error) {
	base := strings.TrimRight(c.BaseURL, "/")
	url := fmt.Sprintf("%s/rest/api/3/issue/%s?fields=summary,fixVersions", base, ref.Key)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return issue.Issue{}, err
	}
	req.SetBasicAuth(c.Email, c.Token)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return issue.Issue{}, fmt.Errorf("could not reach %s: %w", base, err)
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	switch resp.StatusCode {
	case http.StatusOK:
		// fall through to parse below
	case http.StatusUnauthorized, http.StatusForbidden:
		return issue.Issue{}, fmt.Errorf("auth to Jira failed (HTTP %d) — check email/token", resp.StatusCode)
	case http.StatusNotFound:
		return issue.Issue{}, fmt.Errorf("issue %s not found (HTTP 404)", ref.Key)
	default:
		var je jiraError
		_ = dec.Decode(&je)
		if len(je.ErrorMessages) > 0 {
			return issue.Issue{}, fmt.Errorf("unexpected response from Jira (HTTP %d): %s", resp.StatusCode, strings.Join(je.ErrorMessages, "; "))
		}
		return issue.Issue{}, fmt.Errorf("unexpected response from Jira (HTTP %d)", resp.StatusCode)
	}

	var ji jiraIssue
	if err := dec.Decode(&ji); err != nil {
		return issue.Issue{}, fmt.Errorf("decoding Jira response: %w", err)
	}
	if ji.Fields.Summary == "" {
		return issue.Issue{}, fmt.Errorf("issue %s has no summary", ref.Key)
	}

	out := issue.Issue{
		Provider: "jira",
		Key:      ref.Key,
		Title:    ji.Fields.Summary,
		URL:      fmt.Sprintf("%s/browse/%s", base, ref.Key),
	}
	// Per the plan mapping table, the normalized version is the first
	// fixVersion that parses to a MAJOR.MINOR; non-semver names are skipped.
	for _, fv := range ji.Fields.FixVersions {
		if v := issue.ParseVersion(fv.Name); v.OK() {
			vv := v
			out.Version = &vv
			break
		}
	}
	return out, nil
}
