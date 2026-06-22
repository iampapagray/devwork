// Package router implements provider selection (plan §2): an explicit
// --provider flag wins, then full-URL host detection, then bare-id resolution
// via the per-repo default and shape heuristics.
package router

import (
	"fmt"
	"sort"
	"strings"

	"github.com/iampapagray/devwork/internal/provider"
)

// Inputs carries everything Select needs beyond the raw input string.
type Inputs struct {
	FlagProvider    string            // --provider (highest precedence)
	DefaultProvider string            // per-repo default for bare ids
	Hosts           map[string]string // provider name -> configured host (e.g. "acme.atlassian.net")
	Configured      []string          // provider names with a config section present
}

// Select returns the provider that should handle input, or an actionable error.
func Select(input string, in Inputs) (provider.Provider, error) {
	// (1) explicit flag wins.
	if in.FlagProvider != "" {
		p, ok := provider.Get(in.FlagProvider)
		if !ok {
			return nil, fmt.Errorf("unknown provider %q (configured: %s)", in.FlagProvider, strings.Join(provider.Names(), ", "))
		}
		return p, nil
	}

	// (2) full URL -> host detection.
	if host, ok := urlHost(input); ok {
		if p := byConfiguredHost(host, in.Hosts); p != nil {
			return p, nil
		}
		if p := byStrongMatch(input); p != nil {
			return p, nil
		}
		return nil, fmt.Errorf("could not match URL host %q to a provider; pass --provider", host)
	}

	// (3) bare id -> per-repo default, else shape heuristic.
	if in.DefaultProvider != "" {
		if p, ok := provider.Get(in.DefaultProvider); ok {
			return p, nil
		}
		return nil, fmt.Errorf("repo default provider %q is not registered", in.DefaultProvider)
	}

	strong, weak := classify(input)
	if len(strong) == 1 {
		return strong[0], nil
	}
	if len(strong) > 1 {
		return nil, ambiguous(strong)
	}
	switch len(weak) {
	case 1:
		return weak[0], nil
	case 0:
		return nil, fmt.Errorf("could not determine a provider for %q; pass --provider", input)
	default:
		// Narrow ambiguous weak matches to configured providers if that resolves it.
		if cfg := intersect(weak, in.Configured); len(cfg) == 1 {
			return cfg[0], nil
		}
		return nil, ambiguous(weak)
	}
}

func classify(input string) (strong, weak []provider.Provider) {
	for _, p := range provider.All() {
		switch p.Match(input).Confidence {
		case provider.Strong:
			strong = append(strong, p)
		case provider.Weak:
			weak = append(weak, p)
		}
	}
	return strong, weak
}

func byStrongMatch(input string) provider.Provider {
	for _, p := range provider.All() {
		if p.Match(input).Confidence == provider.Strong {
			return p
		}
	}
	return nil
}

func byConfiguredHost(host string, hosts map[string]string) provider.Provider {
	for name, h := range hosts {
		if h == "" {
			continue
		}
		if strings.EqualFold(host, h) {
			if p, ok := provider.Get(name); ok {
				return p
			}
		}
	}
	return nil
}

func intersect(ps []provider.Provider, names []string) []provider.Provider {
	set := map[string]bool{}
	for _, n := range names {
		set[n] = true
	}
	var out []provider.Provider
	for _, p := range ps {
		if set[p.Name()] {
			out = append(out, p)
		}
	}
	return out
}

func ambiguous(ps []provider.Provider) error {
	names := make([]string, len(ps))
	for i, p := range ps {
		names[i] = p.Name()
	}
	sort.Strings(names)
	return fmt.Errorf("input is ambiguous between %s; pass --provider", strings.Join(names, ", "))
}

// urlHost extracts the host from a full URL, returning ok=false for non-URLs.
func urlHost(input string) (string, bool) {
	i := strings.Index(input, "://")
	if i < 0 {
		return "", false
	}
	rest := input[i+3:]
	if at := strings.Index(rest, "@"); at >= 0 {
		rest = rest[at+1:]
	}
	end := strings.IndexAny(rest, "/:")
	if end >= 0 {
		rest = rest[:end]
	}
	if rest == "" {
		return "", false
	}
	return rest, true
}
