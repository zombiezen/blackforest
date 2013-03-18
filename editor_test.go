package main

import (
	"testing"
)

func TestShellEscape(t *testing.T) {
	tests := []struct {
		Arg           string
		UnixQuoted    string
		WindowsQuoted string
	}{
		{``, `''`, `""`},
		{`hello`, `'hello'`, `"hello"`},
		{`hello\`, `'hello\'`, `"hello\"`},
		{`it's okay`, `'it'"'"'s okay'`, `"it's okay"`},
		{`the "best" fit`, `'the "best" fit'`, `"the ""best"" fit"`},
	}

	for _, test := range tests {
		if out, want := shellEscape(test.Arg), test.UnixQuoted; out != want {
			t.Errorf("shellEscape(%q) = %q; want %q", test.Arg, out, want)
		}
		if out, want := windowsEscape(test.Arg), test.WindowsQuoted; out != want {
			t.Errorf("windowsEscape(%q) = %q; want %q", test.Arg, out, want)
		}
	}
}
