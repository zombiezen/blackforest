package main

import (
	"testing"
)

func TestMergeString(t *testing.T) {
	tests := []struct {
		Old, A, B string
		Result    string
		Ok        bool
	}{
		{"", "", "", "", true},
		{"a", "a", "a", "a", true},
		{"a", "b", "b", "b", true},
		{"a", "b", "a", "b", true},
		{"a", "a", "b", "b", true},
		{"a", "b", "c", "a", false},
		{"a", "c", "b", "a", false},
	}
	for _, test := range tests {
		m, ok := mergeString(test.Old, test.A, test.B)
		if m != test.Result {
			t.Errorf("mergeString(%q, %q, %q) = %q; want %q", test.Old, test.A, test.B, m, test.Result)
		}
		if ok != test.Ok {
			t.Errorf("mergeString(%q, %q, %q) ok = %t; want %t", test.Old, test.A, test.B, ok, test.Ok)
		}
	}
}
