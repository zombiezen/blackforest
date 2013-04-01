package search

import (
	"testing"
)

func TestLexQuery(t *testing.T) {
	tests := []struct {
		Query string
		Items []item
	}{
		{"", []item{{eofItem, ""}}},
		{" ", []item{{eofItem, ""}}},
		{"hello", []item{{tokenItem, "hello"}, {eofItem, ""}}},
		{"tag:hello", []item{{tagItem, "tag:"}, {tokenItem, "hello"}, {eofItem, ""}}},
		{"-hello", []item{{notItem, "-"}, {tokenItem, "hello"}, {eofItem, ""}}},
		{"hello world", []item{{tokenItem, "hello"}, {tokenItem, "world"}, {eofItem, ""}}},
		{"hello OR world", []item{{tokenItem, "hello"}, {orItem, "OR"}, {tokenItem, "world"}, {eofItem, ""}}},
	}
	for _, test := range tests {
		items := lexQuery(test.Query)
		if !itemSliceEqual(items, test.Items) {
			t.Errorf("lexQuery(%q) = %v; want %v", test.Query, items, test.Items)
		}
	}
}

func itemSliceEqual(a, b []item) bool {
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
