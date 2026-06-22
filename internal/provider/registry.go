package provider

import (
	"fmt"
	"sort"
	"sync"
)

var (
	mu       sync.RWMutex
	registry = map[string]Provider{}
)

// Register adds a provider to the registry. It panics on a duplicate name,
// which can only happen at init time from a programming error.
func Register(p Provider) {
	mu.Lock()
	defer mu.Unlock()
	name := p.Name()
	if _, dup := registry[name]; dup {
		panic(fmt.Sprintf("provider %q registered twice", name))
	}
	registry[name] = p
}

// Get returns the provider with the given name, or false if absent.
func Get(name string) (Provider, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[name]
	return p, ok
}

// All returns the registered providers sorted by name (stable ordering for the
// router's heuristic pass and for help output).
func All() []Provider {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]Provider, 0, len(names))
	for _, n := range names {
		out = append(out, registry[n])
	}
	return out
}

// Names returns the sorted list of registered provider names.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
