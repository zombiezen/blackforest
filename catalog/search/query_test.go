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
