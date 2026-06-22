package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iampapagray/devwork/internal/creds"
	"github.com/iampapagray/devwork/internal/git"
	"github.com/iampapagray/devwork/internal/issue"
	"github.com/iampapagray/devwork/internal/provider"
)

func TestResolve(t *testing.T) {
	p := New()
	ctx := context.Background()
	remoteCtx := provider.RepoContext{HasRemote: true, Remote: git.Remote{Host: "github.com", Owner: "acme", Repo: "widget"}}

	tests := []struct {
		name             string
		input            string
		ctx              provider.RepoContext
		owner, repo, key string
		wantErr          bool
	}{
		{"url", "https://github.com/o/r/issues/42", provider.RepoContext{}, "o", "r", "42", false},
		{"pull url", "https://github.com/o/r/pull/7", provider.RepoContext{}, "o", "r", "7", false},
		{"owner/repo#n", "acme/widget#9", provider.RepoContext{}, "acme", "widget", "9", false},
		{"bare with remote", "#123", remoteCtx, "acme", "widget", "123", false},
		{"bare number with remote", "123", remoteCtx, "acme", "widget", "123", false},
		{"bare no remote", "123", provider.RepoContext{}, "", "", "", true},
		{"non-github remote", "5", provider.RepoContext{HasRemote: true, Remote: git.Remote{Host: "gitlab.com", Owner: "a", Repo: "b"}}, "", "", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ref, err := p.Resolve(ctx, tc.input, tc.ctx)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", ref)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if ref.Owner != tc.owner || ref.Repo != tc.repo || ref.Key != tc.key {
				t.Errorf("got %+v, want %s/%s#%s", ref, tc.owner, tc.repo, tc.key)
			}
		})
	}
}

func TestFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/widget/issues/9" {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "no auth", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"title":"Fix the thing","html_url":"https://github.com/acme/widget/issues/9","milestone":{"title":"v2.1"}}`))
	}))
	defer srv.Close()

	p := New()
	p.apiBase = srv.URL
	got, err := p.Fetch(context.Background(), issue.IssueRef{Provider: "github", Owner: "acme", Repo: "widget", Key: "9"}, creds.Credentials{Token: "tok"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Fix the thing" || got.Key != "9" {
		t.Errorf("got %+v", got)
	}
	if got.Version == nil || got.Version.MajorMinor() != "2.1" {
		t.Errorf("milestone version = %v", got.Version)
	}
}
