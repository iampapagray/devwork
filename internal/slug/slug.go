// Package slug turns a human issue title into a branch-safe slug.
//
// It is a direct port of the bash reference's slugify(): lowercase, replace
// every non-alphanumeric run with a word boundary, drop stopwords, join with
// hyphens, then cap to MaxWords / MaxChars with a fallback when everything is
// stripped.
package slug

import "strings"

// DefaultStopwords mirrors the reference script's stopword list.
var DefaultStopwords = []string{
	"a", "an", "the", "and", "or", "but", "of", "to", "in", "on", "for",
	"with", "at", "by", "from", "up", "out", "as", "is", "are", "be",
	"been", "being", "this", "that", "these", "those", "it", "its", "into",
	"over", "under", "after", "before", "when", "while", "which", "who",
	"whom", "whose", "will", "would", "should", "could", "can", "may",
	"might", "do", "does", "did", "has", "have", "had", "not", "no",
}

// Default tuning, matching the reference (MAX_WORDS=6, MAX_CHARS=50).
const (
	DefaultMaxWords = 6
	DefaultMaxChars = 50
)

// Slugger holds slug-generation rules. The zero value is not useful; use New.
type Slugger struct {
	MaxWords  int
	MaxChars  int
	stopwords map[string]struct{}
}

// New builds a Slugger. Non-positive maxWords/maxChars fall back to the
// defaults; a nil stopwords slice falls back to DefaultStopwords.
func New(maxWords, maxChars int, stopwords []string) *Slugger {
	if maxWords <= 0 {
		maxWords = DefaultMaxWords
	}
	if maxChars <= 0 {
		maxChars = DefaultMaxChars
	}
	if stopwords == nil {
		stopwords = DefaultStopwords
	}
	set := make(map[string]struct{}, len(stopwords))
	for _, w := range stopwords {
		set[strings.ToLower(w)] = struct{}{}
	}
	return &Slugger{MaxWords: maxWords, MaxChars: maxChars, stopwords: set}
}

// Slugify converts raw text into a branch-safe slug. It never returns an empty
// string: when nothing survives filtering it returns "task".
func (s *Slugger) Slugify(raw string) string {
	// Lowercase, then split on every run of non-alphanumeric characters.
	fields := strings.FieldsFunc(strings.ToLower(raw), func(r rune) bool {
		return !isAlnum(r)
	})

	var words []string
	for _, w := range fields {
		if _, stop := s.stopwords[w]; stop {
			continue
		}
		words = append(words, w)
		if len(words) >= s.MaxWords {
			break
		}
	}

	out := strings.Join(words, "-")
	if len(out) > s.MaxChars {
		out = out[:s.MaxChars]
	}
	// Trim a dangling hyphen left by truncation.
	out = strings.TrimRight(out, "-")
	if out == "" {
		return "task"
	}
	return out
}

func isAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}
