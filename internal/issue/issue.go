// Package issue holds the normalized cross-provider issue model and the
// version comparison used by the branch-prefix gate.
package issue

// Issue is the normalized view of a tracker item, independent of provider.
type Issue struct {
	Provider string   // "jira" | "github" | "gitlab"
	Key      string   // canonical id used in the branch: "PROJ-123", "123"
	Title    string   // raw summary/title, fed to the slugger
	Version  *Version // optional fixVersion / milestone; nil when none
	URL      string   // canonical web URL, for confirmation display
}

// IssueRef is the result of parsing input: enough to fetch, with no network
// performed yet. Owner/Repo/Project are populated for GitHub/GitLab.
type IssueRef struct {
	Provider string
	Key      string // "PROJ-123" for Jira, "123" for GitHub/GitLab
	Owner    string // GitHub owner / GitLab namespace (optional)
	Repo     string // GitHub repo (optional)
	Project  string // GitLab project id or path (optional)
	BaseURL  string // resolved host base, when derived from a full URL
}
