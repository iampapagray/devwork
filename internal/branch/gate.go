package branch

import (
	"fmt"

	"github.com/iampapagray/devwork/internal/issue"
)

// Status is the outcome class of comparing the issue version to the repo version.
type Status int

const (
	// StatusEqual: issue and repo versions match.
	StatusEqual Status = iota
	// StatusAhead: the issue version is ahead of the repo (post-release transition).
	StatusAhead
	// StatusBehind: the issue version is behind/incompatible with the repo.
	StatusBehind
)

// GateResult describes the gate decision for an active version gate.
type GateResult struct {
	Status       Status
	Version      string // normalized MAJOR.MINOR used as the branch prefix (issue version wins)
	RepoVersion  string // normalized repo version, for display
	NeedsConfirm bool   // ahead, or behind in non-strict mode
	Abort        bool   // behind in strict mode
	Message      string // human note for the confirmation UI
}

// Gate classifies the issue version against the repo version per decision C.
// Both versions must be present and parseable; callers handle the
// "unresolvable -> hard error" cases before calling Gate.
//
// strict turns a behind/incompatible issue version into a hard abort instead of
// a warn+confirm.
func Gate(issueVer, repoVer issue.Version, strict bool) GateResult {
	res := GateResult{
		Version:     issueVer.MajorMinor(),
		RepoVersion: repoVer.MajorMinor(),
	}
	switch issue.Compare(issueVer, repoVer) {
	case 0:
		res.Status = StatusEqual
		res.Message = fmt.Sprintf("matches repo version %s", res.RepoVersion)
	case 1:
		res.Status = StatusAhead
		res.NeedsConfirm = true
		res.Message = fmt.Sprintf("post-release transition: repo version is %s", res.RepoVersion)
	default: // -1
		res.Status = StatusBehind
		res.Message = fmt.Sprintf("issue version %s is behind repo version %s — wrong repo/branch?", res.Version, res.RepoVersion)
		if strict {
			res.Abort = true
		} else {
			res.NeedsConfirm = true
		}
	}
	return res
}
