package main

import (
	"reflect"
	"testing"
)

func TestOrganizeTags(t *testing.T) {
	tests := []struct {
		Finder tagFinder
		Groups []tagGroup
	}{
		{
			Finder: mockTagFinder{},
			Groups: []tagGroup{{"", []tagInfo{}}},
		},
		{
			Finder: mockTagFinder{
				"a": {"b", "c"},
			},
			Groups: []tagGroup{
				{"", []tagInfo{{"a", 2}}},
			},
		},
		{
			Finder: mockTagFinder{
				"lang-go": {"foo", "bar"},
			},
			Groups: []tagGroup{
				{"", []tagInfo{{"lang-go", 2}}},
			},
		},
		{
			Finder: mockTagFinder{
				"lang-go":  {"foo", "bar"},
				"lang-c++": {"baz"},
			},
			Groups: []tagGroup{
				{"lang", []tagInfo{{"lang-c++", 1}, {"lang-go", 2}}},
				{"", []tagInfo{}},
			},
		},
		{
			Finder: mockTagFinder{
				"lang-go":  {"a"},
				"lang-c++": {"b", "c"},
				"opengl":   {"b"},
			},
			Groups: []tagGroup{
				{"lang", []tagInfo{{"lang-c++", 2}, {"lang-go", 1}}},
				{"", []tagInfo{{"opengl", 1}}},
			},
		},
	}

	for _, test := range tests {
		groups := organizeTags(test.Finder)
		if !reflect.DeepEqual(groups, test.Groups) {
			t.Errorf("organizeTags(%v) = %+v; want %+v", test.Finder, groups, test.Groups)
		}
	}
}

type mockTagFinder map[string][]string

func (tf mockTagFinder) Tags() []string {
	t := make([]string, 0, len(tf))
	for k := range tf {
		t = append(t, k)
	}
	return t
}

func (tf mockTagFinder) FindTag(tag string) []string {
	return tf[tag]
}
