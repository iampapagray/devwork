package issue

import (
	"regexp"
	"strconv"
)

// Version is a MAJOR.MINOR view of a version-ish string. Minor is an integer so
// 7.10 > 7.9 (unlike string comparison). Patch and pre-release are ignored for
// the branch-prefix gate, matching the bash reference's norm_version.
type Version struct {
	Raw   string
	Major int
	Minor int
	ok    bool
}

// OK reports whether the version parsed into a usable MAJOR.MINOR.
func (v Version) OK() bool { return v.ok }

// MajorMinor renders the normalized "MAJOR.MINOR" form used as a branch prefix.
func (v Version) MajorMinor() string {
	if !v.ok {
		return ""
	}
	return strconv.Itoa(v.Major) + "." + strconv.Itoa(v.Minor)
}

var verRe = regexp.MustCompile(`([0-9]+)\.([0-9]+)`)

// ParseVersion extracts the first MAJOR.MINOR from any version-ish string
// (e.g. "Release v7.3.0" -> {7,3}). When none is present it returns a Version
// with ok == false.
func ParseVersion(raw string) Version {
	m := verRe.FindStringSubmatch(raw)
	if m == nil {
		return Version{Raw: raw}
	}
	// Submatches are \d+ so Atoi cannot fail here.
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	return Version{Raw: raw, Major: major, Minor: minor, ok: true}
}

// Compare returns 1 if a > b, -1 if a < b, 0 if equal, comparing MAJOR then
// MINOR numerically. Both versions are assumed parsed (ok == true).
func Compare(a, b Version) int {
	if a.Major != b.Major {
		if a.Major > b.Major {
			return 1
		}
		return -1
	}
	if a.Minor != b.Minor {
		if a.Minor > b.Minor {
			return 1
		}
		return -1
	}
	return 0
}
