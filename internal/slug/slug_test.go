package slug

import "testing"

func TestSlugify(t *testing.T) {
	s := New(DefaultMaxWords, DefaultMaxChars, nil)
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"basic", "Fix login redirect", "fix-login-redirect"},
		{"drops stopwords", "Update the alert and close the modal", "update-alert-close-modal"},
		{"non-alnum to boundary", "Alert: follow-ups & close (modal)", "alert-follow-ups-close-modal"},
		{"caps to max words", "one two three four five six seven eight", "one-two-three-four-five-six"},
		{"empty fallback", "the a an of to", "task"},
		{"blank fallback", "", "task"},
		{"unicode stripped", "café ☕ update", "caf-update"},
		{"numbers kept", "bump v2 api 123", "bump-v2-api-123"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := s.Slugify(tc.in); got != tc.want {
				t.Errorf("Slugify(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSlugifyMaxChars(t *testing.T) {
	s := New(20, 10, nil)
	got := s.Slugify("alpha bravo charlie delta")
	if len(got) > 10 {
		t.Fatalf("slug %q exceeds max chars 10", got)
	}
	// must not end on a dangling hyphen
	if got[len(got)-1] == '-' {
		t.Errorf("slug %q ends with a hyphen", got)
	}
}

func TestNewDefaults(t *testing.T) {
	s := New(0, -1, nil)
	if s.MaxWords != DefaultMaxWords || s.MaxChars != DefaultMaxChars {
		t.Errorf("got MaxWords=%d MaxChars=%d, want defaults", s.MaxWords, s.MaxChars)
	}
}
