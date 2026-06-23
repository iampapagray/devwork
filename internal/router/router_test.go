package router

import (
	"context"
	"testing"

	"github.com/iampapagray/devwork/internal/creds"
	"github.com/iampapagray/devwork/internal/issue"
	"github.com/iampapagray/devwork/internal/provider"
)

// stub is a configurable fake provider for router tests.
type stub struct {
	name  string
	match func(string) provider.MatchResult
}

func (s stub) Name() string                         { return s.name }
func (s stub) Match(in string) provider.MatchResult { return s.match(in) }
func (s stub) Resolve(context.Context, string, provider.RepoContext) (issue.IssueRef, error) {
	return issue.IssueRef{Provider: s.name}, nil
}

func (s stub) Fetch(context.Context, issue.IssueRef, creds.Credentials) (issue.Issue, error) {
	return issue.Issue{}, nil
}

func registerStubs() {
	provider.Register(stub{name: "jira", match: func(in string) provider.MatchResult {
		if contains(in, "atlassian.net") || contains(in, "/browse/") || matchesKey(in) {
			return provider.MatchResult{Confidence: provider.Strong}
		}
		return provider.MatchResult{Confidence: provider.NoMatch}
	}})
	provider.Register(stub{name: "github", match: func(in string) provider.MatchResult {
		if contains(in, "github.com") {
			return provider.MatchResult{Confidence: provider.Strong}
		}
		if isNumeric(in) {
			return provider.MatchResult{Confidence: provider.Weak}
		}
		return provider.MatchResult{Confidence: provider.NoMatch}
	}})
	provider.Register(stub{name: "gitlab", match: func(in string) provider.MatchResult {
		if contains(in, "gitlab.com") {
			return provider.MatchResult{Confidence: provider.Strong}
		}
		if isNumeric(in) {
			return provider.MatchResult{Confidence: provider.Weak}
		}
		return provider.MatchResult{Confidence: provider.NoMatch}
	}})
}

func contains(s, sub string) bool { return len(s) >= len(sub) && indexOf(s, sub) >= 0 }
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func matchesKey(s string) bool {
	dash := indexOf(s, "-")
	return dash > 0 && dash < len(s)-1 && s[0] >= 'A' && s[0] <= 'Z'
}

func isNumeric(s string) bool {
	s = trimHash(s)
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func trimHash(s string) string {
	if len(s) > 0 && s[0] == '#' {
		return s[1:]
	}
	return s
}

func TestSelect(t *testing.T) {
	registerStubs()

	tests := []struct {
		name  string
		input string
		in    Inputs
		want  string
		err   bool
	}{
		{"flag wins", "anything", Inputs{FlagProvider: "github"}, "github", false},
		{"unknown flag", "x", Inputs{FlagProvider: "bogus"}, "", true},
		{"jira url", "https://acme.atlassian.net/browse/QA-1", Inputs{}, "jira", false},
		{"github url", "https://github.com/o/r/issues/5", Inputs{}, "github", false},
		{"configured host", "https://git.acme.com/team/app/-/issues/3", Inputs{Hosts: map[string]string{"gitlab": "git.acme.com"}}, "gitlab", false},
		{"unknown host", "https://example.com/x", Inputs{}, "", true},
		{"jira key shape", "QA-2840", Inputs{}, "jira", false},
		{"bare default", "123", Inputs{DefaultProvider: "github"}, "github", false},
		{"bare ambiguous", "123", Inputs{}, "", true},
		{"bare narrowed by config", "123", Inputs{Configured: []string{"gitlab"}}, "gitlab", false},
		{"unresolvable", "totally-unknown!", Inputs{}, "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := Select(tc.input, tc.in)
			if tc.err {
				if err == nil {
					t.Fatalf("expected error, got %v", p)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Name() != tc.want {
				t.Errorf("got %q, want %q", p.Name(), tc.want)
			}
		})
	}
}
