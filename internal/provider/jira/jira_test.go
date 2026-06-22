package jira

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/iampapagray/devwork/internal/creds"
	"github.com/iampapagray/devwork/internal/issue"
	"github.com/iampapagray/devwork/internal/provider"
)

func TestExtractKey(t *testing.T) {
	tests := []struct {
		in   string
		want string
		ok   bool
	}{
		{"QA-2840", "QA-2840", true},
		{"https://x.atlassian.net/browse/qa-2840", "QA-2840", true},
		{"-task PROJ-1", "PROJ-1", true},
		{"no key here", "", false},
		{"123", "", false},
	}
	for _, tc := range tests {
		got, ok := ExtractKey(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Errorf("ExtractKey(%q) = (%q,%v), want (%q,%v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestMatch(t *testing.T) {
	p := New()
	if p.Match("https://acme.atlassian.net/browse/QA-1").Confidence != provider.Strong {
		t.Error("expected Strong for atlassian URL")
	}
	if p.Match("QA-2840").Confidence != provider.Strong {
		t.Error("expected Strong for key shape")
	}
	if p.Match("123").Confidence != provider.NoMatch {
		t.Error("expected NoMatch for bare number")
	}
}

func TestFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/QA-2840" {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}
		if u, p, ok := r.BasicAuth(); !ok || u != "me@acme.com" || p != "tok" {
			http.Error(w, "no auth", http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"fields":{"summary":"Alert follow-ups and close modal update","fixVersions":[{"name":"Release v7.3.0"},{"name":"v7.10"}]}}`))
	}))
	defer srv.Close()

	p := New()
	ref := issue.IssueRef{Provider: "jira", Key: "QA-2840"}
	got, err := p.Fetch(context.Background(), ref, creds.Credentials{
		Email: "me@acme.com", Token: "tok", BaseURL: srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Alert follow-ups and close modal update" {
		t.Errorf("title = %q", got.Title)
	}
	if got.Version == nil || got.Version.MajorMinor() != "7.3" {
		t.Errorf("version = %v, want first fixVersion 7.3", got.Version)
	}
	if got.URL != srv.URL+"/browse/QA-2840" {
		t.Errorf("url = %q", got.URL)
	}
}

func TestFetchErrors(t *testing.T) {
	cases := []struct {
		code int
	}{{http.StatusUnauthorized}, {http.StatusForbidden}, {http.StatusNotFound}, {http.StatusInternalServerError}}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.code)
			_, _ = w.Write([]byte(`{"errorMessages":["boom"]}`))
		}))
		ref := issue.IssueRef{Provider: "jira", Key: "QA-1"}
		_, err := New().Fetch(context.Background(), ref, creds.Credentials{Email: "e", Token: "t", BaseURL: srv.URL})
		if err == nil {
			t.Errorf("HTTP %d: expected error", tc.code)
		}
		srv.Close()
	}
}
