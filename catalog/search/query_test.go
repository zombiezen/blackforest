package search

import (
	"reflect"
	"testing"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		Query  string
		Expect queryAST
	}{
		{"", nil},
		{"hello", token("hello")},
		{"tag:hello", tagAtom("hello")},
		{"-hello", queryNot{token("hello")}},
		{"hello world", queryAnd{token("hello"), token("world")}},
		{"hello OR world", queryOr{token("hello"), token("world")}},
	}
	for _, test := range tests {
		ast, err := parseQuery(test.Query)
		if err != nil {
			t.Errorf("parseQuery(%q) error: %v", test.Query, err)
		}
		if !reflect.DeepEqual(ast, test.Expect) {
			t.Errorf("parseQuery(%q) = %v; want %v", test.Query, ast, test.Expect)
		}
	}
}

func TestFindTerms(t *testing.T) {
	tests := []struct {
		Query string
		Text  string
		Pairs []int
	}{
		{"", "", []int{}},
		{"hello", "hello", []int{0, 5}},
		{"hello", "world", []int{}},
		{"HELLO", "hello", []int{0, 5}},
		{"hello", "  hello  ", []int{2, 7}},
		{"hello", "hello hello", []int{0, 5, 6, 11}},
	}
	for _, test := range tests {
		pairs := FindTerms(test.Query, test.Text)
		if !intSliceEq(pairs, test.Pairs) {
			t.Errorf("FindTerms(%q, %q) = %v; want %v", test.Query, test.Text, pairs, test.Pairs)
		}
	}
}

func intSliceEq(a, b []int) bool {
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
