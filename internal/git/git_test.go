package git

import "testing"

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name              string
		in                string
		host, owner, repo string
		wantErr           bool
	}{
		{"ssh github", "git@github.com:iampapagray/devwork.git", "github.com", "iampapagray", "devwork", false},
		{"ssh no suffix", "git@github.com:acme/widget", "github.com", "acme", "widget", false},
		{"https github", "https://github.com/iampapagray/devwork.git", "github.com", "iampapagray", "devwork", false},
		{"https with user", "https://user@gitlab.com/group/proj.git", "gitlab.com", "group", "proj", false},
		{"gitlab subgroup", "git@gitlab.com:group/sub/proj.git", "gitlab.com", "group/sub", "proj", false},
		{"self-hosted port", "https://gitlab.acme.com:8443/team/app.git", "gitlab.acme.com", "team", "app", false},
		{"ssh scheme", "ssh://git@github.com/acme/widget.git", "github.com", "acme", "widget", false},
		{"empty", "", "", "", "", true},
		{"garbage", "not-a-url", "", "", "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r, err := ParseRemoteURL(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if r.Host != tc.host || r.Owner != tc.owner || r.Repo != tc.repo {
				t.Errorf("got %+v, want host=%s owner=%s repo=%s", r, tc.host, tc.owner, tc.repo)
			}
		})
	}
}
