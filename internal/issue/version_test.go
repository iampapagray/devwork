package issue

import "testing"

func TestParseVersion(t *testing.T) {
	tests := []struct {
		in         string
		wantOK     bool
		major      int
		minor      int
		majorMinor string
	}{
		{"7.3", true, 7, 3, "7.3"},
		{"Release v7.3.0", true, 7, 3, "7.3"},
		{"v7.10.2-rc1", true, 7, 10, "7.10"},
		{"1.0", true, 1, 0, "1.0"},
		{"no version here", false, 0, 0, ""},
		{"", false, 0, 0, ""},
		{"sprint-42", false, 0, 0, ""},
	}
	for _, tc := range tests {
		v := ParseVersion(tc.in)
		if v.OK() != tc.wantOK {
			t.Errorf("ParseVersion(%q).OK() = %v, want %v", tc.in, v.OK(), tc.wantOK)
			continue
		}
		if !tc.wantOK {
			continue
		}
		if v.Major != tc.major || v.Minor != tc.minor {
			t.Errorf("ParseVersion(%q) = {%d,%d}, want {%d,%d}", tc.in, v.Major, v.Minor, tc.major, tc.minor)
		}
		if v.MajorMinor() != tc.majorMinor {
			t.Errorf("ParseVersion(%q).MajorMinor() = %q, want %q", tc.in, v.MajorMinor(), tc.majorMinor)
		}
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"7.3", "7.3", 0},
		{"7.10", "7.9", 1}, // integer minor: 10 > 9
		{"7.9", "7.10", -1},
		{"8.0", "7.99", 1},
		{"6.5", "7.0", -1},
	}
	for _, tc := range tests {
		got := Compare(ParseVersion(tc.a), ParseVersion(tc.b))
		if got != tc.want {
			t.Errorf("Compare(%s, %s) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}
