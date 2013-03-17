package main

import (
	"testing"
)

func TestFold(t *testing.T) {
	tests := []struct {
		S      string
		Folded string
	}{
		{"", ""},
		{"A", "A"},
		{"a", "A"},
		{"Hello, World!", "HELLO, WORLD!"},
		{"hello, world!", "HELLO, WORLD!"},
	}
	for _, test := range tests {
		f := fold(test.S)
		if f != test.Folded {
			t.Errorf("fold(%q) = %q; want %q", test.S, f, test.Folded)
		}
	}
}
