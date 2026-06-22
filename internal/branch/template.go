// Package branch renders branch names from a template and runs the version
// gate described in decision C of the plan.
package branch

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// tokenRe matches "{token}" placeholders.
var tokenRe = regexp.MustCompile(`\{([a-z]+)\}`)

// knownTokens are the template tokens devwork can fill in v1.
var knownTokens = map[string]bool{
	"key":      true,
	"slug":     true,
	"version":  true,
	"provider": true,
}

// HasVersionToken reports whether the template activates the version gate.
func HasVersionToken(template string) bool {
	return strings.Contains(template, "{version}")
}

// ValidateTemplate returns an error if the template references unknown tokens.
func ValidateTemplate(template string) error {
	var unknown []string
	for _, m := range tokenRe.FindAllStringSubmatch(template, -1) {
		if !knownTokens[m[1]] {
			unknown = append(unknown, "{"+m[1]+"}")
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf("unknown template token(s): %s (valid: {key} {slug} {version} {provider})",
			strings.Join(dedup(unknown), " "))
	}
	return nil
}

// Render fills the template from fields. It validates tokens first; a {version}
// token with an empty value is a programming error (the gate should have
// errored earlier) and is reported as such.
func Render(template string, fields map[string]string) (string, error) {
	if err := ValidateTemplate(template); err != nil {
		return "", err
	}
	var rerr error
	out := tokenRe.ReplaceAllStringFunc(template, func(tok string) string {
		name := tok[1 : len(tok)-1]
		v, ok := fields[name]
		if !ok || v == "" {
			rerr = fmt.Errorf("template token %s has no value", tok)
			return tok
		}
		return v
	})
	if rerr != nil {
		return "", rerr
	}
	return out, nil
}

func dedup(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
