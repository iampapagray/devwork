package branch

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/iampapagray/devwork/internal/issue"
)

// TagFunc returns the repo's latest tag (e.g. via git describe). It may return
// an error when there are no tags.
type TagFunc func() (string, error)

// ResolveRepoVersion reads the repo version from the configured source. "auto"
// tries package.json, then VERSION, then git-tag. It returns an error only on a
// malformed/missing explicit source; an unparseable version yields a Version
// with OK()==false so the caller can produce the decision-C hard error with
// full context.
func ResolveRepoVersion(repoRoot, source string, tag TagFunc) (issue.Version, error) {
	switch source {
	case "", "package.json":
		return fromPackageJSON(repoRoot)
	case "VERSION":
		return fromVersionFile(repoRoot)
	case "git-tag":
		return fromGitTag(tag)
	case "auto":
		for _, fn := range []func() (issue.Version, error){
			func() (issue.Version, error) { return fromPackageJSON(repoRoot) },
			func() (issue.Version, error) { return fromVersionFile(repoRoot) },
			func() (issue.Version, error) { return fromGitTag(tag) },
		} {
			if v, err := fn(); err == nil && v.OK() {
				return v, nil
			}
		}
		return issue.Version{}, fmt.Errorf("no version found via auto (tried package.json, VERSION, git tag)")
	default:
		return issue.Version{}, fmt.Errorf("unknown version source %q (want package.json, VERSION, git-tag, or auto)", source)
	}
}

func fromPackageJSON(repoRoot string) (issue.Version, error) {
	path := filepath.Join(repoRoot, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return issue.Version{}, fmt.Errorf("reading %s: %w", path, err)
	}
	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return issue.Version{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	if pkg.Version == "" {
		return issue.Version{}, fmt.Errorf("%s has no \"version\" field", path)
	}
	return issue.ParseVersion(pkg.Version), nil
}

func fromVersionFile(repoRoot string) (issue.Version, error) {
	path := filepath.Join(repoRoot, "VERSION")
	data, err := os.ReadFile(path)
	if err != nil {
		return issue.Version{}, fmt.Errorf("reading %s: %w", path, err)
	}
	return issue.ParseVersion(strings.TrimSpace(string(data))), nil
}

func fromGitTag(tag TagFunc) (issue.Version, error) {
	if tag == nil {
		return issue.Version{}, fmt.Errorf("git tag source unavailable")
	}
	t, err := tag()
	if err != nil {
		return issue.Version{}, fmt.Errorf("git tag: %w", err)
	}
	return issue.ParseVersion(strings.TrimPrefix(strings.TrimSpace(t), "v")), nil
}
