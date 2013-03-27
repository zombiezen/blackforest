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

func TestPageList(t *testing.T) {
	tests := []struct {
		Curr, N int
		Size    int

		Prev []int
		Next []int
	}{
		{1, 1, 5, []int{}, []int{}},
		{1, 3, 5, []int{}, []int{2, 3}},
		{2, 3, 5, []int{1}, []int{3}},
		{3, 3, 5, []int{1, 2}, []int{}},
		{1, 10, 5, []int{}, []int{2, 3, 4, 5}},
		{2, 10, 5, []int{1}, []int{3, 4, 5}},
		{3, 10, 5, []int{1, 2}, []int{4, 5}},
		{4, 10, 5, []int{2, 3}, []int{5, 6}},
		{7, 10, 5, []int{5, 6}, []int{8, 9}},
		{8, 10, 5, []int{6, 7}, []int{9, 10}},
		{9, 10, 5, []int{6, 7, 8}, []int{10}},
		{10, 10, 5, []int{6, 7, 8, 9}, []int{}},
	}

	for _, test := range tests {
		if prev := prevPageList(test.Curr, test.N, test.Size); !reflect.DeepEqual(prev, test.Prev) {
			t.Errorf("prevPageList(%d, %d, %d) = %v; want %v", test.Curr, test.N, test.Size, prev, test.Prev)
		}
	}
	for _, test := range tests {
		if next := nextPageList(test.Curr, test.N, test.Size); !reflect.DeepEqual(next, test.Next) {
			t.Errorf("nextPageList(%d, %d, %d) = %v; want %v", test.Curr, test.N, test.Size, next, test.Next)
		}
	}
}
