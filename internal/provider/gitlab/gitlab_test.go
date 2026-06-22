package gitlab

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
	remoteCtx := provider.RepoContext{HasRemote: true, Remote: git.Remote{Host: "gitlab.com", Owner: "group/sub", Repo: "proj"}}

	tests := []struct {
		name      string
		input     string
		ctx       provider.RepoContext
		proj, key string
		wantErr   bool
	}{
		{"url", "https://gitlab.com/group/proj/-/issues/12", provider.RepoContext{}, "group/proj", "12", false},
		{"url subgroup", "https://gitlab.com/a/b/c/-/issues/3", provider.RepoContext{}, "a/b/c", "3", false},
		{"path#n", "group/proj#5", provider.RepoContext{}, "group/proj", "5", false},
		{"bare with remote", "#7", remoteCtx, "group/sub/proj", "7", false},
		{"bare no remote", "7", provider.RepoContext{}, "", "", true},
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
			if ref.Project != tc.proj || ref.Key != tc.key {
				t.Errorf("got %+v, want %s#%s", ref, tc.proj, tc.key)
			}
		})
	}
}

func TestFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// project path is URL-encoded: group%2Fproj
		if r.URL.EscapedPath() != "/api/v4/projects/group%2Fproj/issues/5" {
			http.Error(w, "bad path: "+r.URL.EscapedPath(), http.StatusNotFound)
			return
		}
		if r.Header.Get("PRIVATE-TOKEN") != "tok" {
			http.Error(w, "no auth", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"title":"Add dark mode","web_url":"https://gitlab.com/group/proj/-/issues/5","milestone":{"title":"Release 3.4"}}`))
	}))
	defer srv.Close()

	p := New()
	got, err := p.Fetch(context.Background(),
		issue.IssueRef{Provider: "gitlab", Project: "group/proj", Key: "5"},
		creds.Credentials{Token: "tok", BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Add dark mode" || got.Key != "5" {
		t.Errorf("got %+v", got)
	}
	if got.Version == nil || got.Version.MajorMinor() != "3.4" {
		t.Errorf("milestone version = %v", got.Version)
	}
}
