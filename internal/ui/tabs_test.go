package ui

import "testing"

// TestTruncateLabel covers the rune-aware truncation used to fit model names
// into a fixed-width grid cell.
func TestTruncateLabel(t *testing.T) {
	cases := []struct {
		in   string
		w    int
		want string
	}{
		{"short", 10, "short"},
		{"exactfit!!", 10, "exactfit!!"},
		{"toolongforcell", 10, "toolongfo…"},
		{"abc", 1, "…"},
		{"abc", 0, ""},
		{"abc", -3, ""},
		{"héllo wörld", 6, "héllo…"},
	}
	for _, c := range cases {
		if got := truncateLabel(c.in, c.w); got != c.want {
			t.Errorf("truncateLabel(%q, %d) = %q, want %q", c.in, c.w, got, c.want)
		}
	}
}
