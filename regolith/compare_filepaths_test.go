package regolith

import "testing"

func TestCompareFilePaths(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"text/text.txt", "text.txt", -1},
		{"text.txt", "text/text.txt", 1},
		{"abc/a.txt", "abc/b.txt", -1},
		{"abc/b.txt", "abc/a.txt", 1},
		{"abc\\b.txt", "abc/b.txt", 0},
		{"same/path.txt", "same/path.txt", 0},
	}
	for _, tt := range tests {
		if got := compareFilePaths(tt.a, tt.b); got != tt.expected {
			t.Errorf("compareFilePaths(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.expected)
		}
	}
}
