package main

import (
	"testing"
)

func TestShellEscape(t *testing.T) {
	tests := []struct {
		Arg    string
		Quoted string
	}{
		{``, `''`},
		{`hello`, `'hello'`},
		{`hello\`, `'hello\'`},
		{`it's okay`, `'it'"'"'s okay'`},
	}

	for _, test := range tests {
		if out, want := shellEscape(test.Arg), test.Quoted; out != want {
			t.Errorf("shellEscape(%q) = %q; want %q", test.Arg, out, want)
		}
	}
}
