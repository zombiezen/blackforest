package search

import (
	"testing"
	"unicode"
)

func TestFold(t *testing.T) {
	const asciiPunctuation = "\x00\x01\x02\x03\x04\x05\x06\x07\x08\t\n\x0b\x0c\r\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f !\"#$%&'()*+,-./0123456789:;<=>?@[\\]^_`{|}~\x7f"
	tests := []struct {
		s string
		f string
	}{
		{"", ""},
		{"abcdefghijklmnopqrstuvwxyz", "ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{"ABCDEFGHIJKLMNOPQRSTUVWXYZ", "ABCDEFGHIJKLMNOPQRSTUVWXYZ"},
		{asciiPunctuation, asciiPunctuation},
	}
	for _, test := range tests {
		f := foldString(test.s)
		if f != test.f {
			t.Errorf("foldString(%q) = %q; want %q", test.s, f, test.f)
		}
	}
}

func BenchmarkFoldASCII(b *testing.B) {
	const n = 65536
	b.StopTimer()
	r := make([]rune, 0, n)
	for i := 0; i < n; i++ {
		r = append(r, rune(i%128))
	}
	buf := make([]rune, len(r))
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		copy(buf, r)
		fold(buf)
	}
	b.SetBytes(n)
}

func BenchmarkFold(b *testing.B) {
	const n = 65536
	b.StopTimer()
	r := make([]rune, 0, n)
	for i := 0; i < n; i++ {
		r = append(r, rune(i))
	}
	buf := make([]rune, len(r))
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		copy(buf, r)
		fold(buf)
	}
	b.SetBytes(n)
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		s string
		a []string
	}{
		{"", []string{}},
		{" ", []string{}},
		{" \t ", []string{}},
		{" abc ", []string{"abc"}},
		{"1 2 3 4", []string{"1", "2", "3", "4"}},
		{"1. 2. 3. -4.", []string{"1", "2", "3", "4"}},
	}
	for _, test := range tests {
		a := makeStringArray(tokenize([]rune(test.s)))
		if !strarrEq(a, test.a) {
			t.Errorf("tokenize(%q) = %v; want %v", test.s, a, test.a)
		}
	}
}

func makeStringArray(a [][]rune) []string {
	s := make([]string, len(a))
	for i := range a {
		s[i] = string(a[i])
	}
	return s
}

func strarrEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestIsTokenSep(t *testing.T) {
	allowed := []*unicode.RangeTable{unicode.Letter, unicode.Number}
	for r := rune(0); r <= unicode.MaxRune; r++ {
		if mine, actual := isTokenSep(r), !unicode.IsOneOf(allowed, r); mine != actual {
			t.Errorf("isTokenSep(%q) = %t; want %t", r, mine, actual)
		}
	}
}

func BenchmarkIsTokenSep(b *testing.B) {
	for i := 0; i < b.N; i++ {
		isTokenSep(0x10ffff)
	}
}

func TestStripTokenSep(t *testing.T) {
	tests := []struct {
		r string
		s string
	}{
		{"", ""},
		{"A", "A"},
		{"a", "a"},
		{"a.", "a"},
	}
	for _, test := range tests {
		s := string(stripTokenSep([]rune(test.r)))
		if s != test.s {
			t.Errorf("stripTokenSep(%q) = %q; want %q", test.r, s, test.s)
		}
	}
}
