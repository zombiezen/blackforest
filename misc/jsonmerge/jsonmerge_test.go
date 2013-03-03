package main

import (
	"reflect"
	"testing"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		Old, A, B interface{}
		Result    interface{}
		Conflicts bool
	}{
		{nil, nil, nil, nil, false},
		{nil, "a", "b", &mergeConflict{"a", "b"}, true},
		{nil, "a", nil, "a", false},
		{"a", nil, "a", nil, false},
		{"a", nil, nil, nil, false},
		{"", "", "", "", false},
		{false, false, false, false, false},
		{false, true, true, true, false},
		{true, false, false, false, false},
		{true, true, true, true, false},
		{1.0, 1.0, 1.0, 1.0, false},
		{1.0, 2.0, 2.0, 2.0, false},
		{1.0, 2.0, 1.0, 2.0, false},
		{1.0, 1.0, 2.0, 2.0, false},
		{1.0, 2.0, 3.0, &mergeConflict{2.0, 3.0}, true},
		{1.0, 3.0, 2.0, &mergeConflict{3.0, 2.0}, true},
		{"a", "a", "a", "a", false},
		{"a", "b", "b", "b", false},
		{"a", "b", "a", "b", false},
		{"a", "a", "b", "b", false},
		{"a", "b", "c", &mergeConflict{"b", "c"}, true},
		{"a", "c", "b", &mergeConflict{"c", "b"}, true},
		{"foo", "foo", 42.0, 42.0, false},
		{"foo", 42.0, "foo", 42.0, false},
		{"foo", "bar", 42.0, &mergeConflict{"bar", 42.0}, true},
		{"foo", 42.0, "bar", &mergeConflict{42.0, "bar"}, true},
		{
			map[string]interface{}{},
			map[string]interface{}{},
			map[string]interface{}{},
			map[string]interface{}{},
			false,
		},
		{
			nil,
			map[string]interface{}{},
			map[string]interface{}{},
			map[string]interface{}{},
			false,
		},
		{
			map[string]interface{}{},
			map[string]interface{}{"a": 1.0},
			map[string]interface{}{"b": 2.0},
			map[string]interface{}{"a": 1.0, "b": 2.0},
			false,
		},
		{
			nil,
			map[string]interface{}{"a": 1.0},
			map[string]interface{}{"b": 2.0},
			map[string]interface{}{"a": 1.0, "b": 2.0},
			false,
		},
		{
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 0.0},
			false,
		},
		{
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 1.0},
			map[string]interface{}{"a": 1.0},
			map[string]interface{}{"a": 1.0},
			false,
		},
		{
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 1.0},
			map[string]interface{}{"a": 2.0},
			map[string]interface{}{"a": &mergeConflict{1.0, 2.0}},
			true,
		},
		{
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 2.0},
			map[string]interface{}{"a": 1.0},
			map[string]interface{}{"a": &mergeConflict{2.0, 1.0}},
			true,
		},
		{
			map[string]interface{}{},
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 0.0},
			false,
		},
		{
			map[string]interface{}{},
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{},
			map[string]interface{}{"a": 0.0},
			false,
		},
		{
			map[string]interface{}{},
			map[string]interface{}{},
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 0.0},
			false,
		},
		{
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{},
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{},
			false,
		},
		{
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{},
			map[string]interface{}{},
			false,
		},
		{
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{},
			map[string]interface{}{"a": 1.0},
			map[string]interface{}{"a": &mergeConflict{nil, 1.0}},
			true,
		},
		{
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 1.0},
			map[string]interface{}{},
			map[string]interface{}{"a": &mergeConflict{1.0, nil}},
			true,
		},
		{
			map[string]interface{}{},
			map[string]interface{}{"a": 1.0},
			map[string]interface{}{"a": 2.0},
			map[string]interface{}{"a": &mergeConflict{1.0, 2.0}},
			true,
		},
		{
			map[string]interface{}{
				"a": 47.0,
				"b": 3.0,
			},
			map[string]interface{}{
				"a": 47.0,
				"b": 3.0,
				"c": "hi",
				"e": "wut",
			},
			map[string]interface{}{
				"a": 47.0,
				"c": "hi",
				"d": "hey",
				"e": "now",
			},
			map[string]interface{}{
				"a": 47.0,
				"c": "hi",
				"d": "hey",
				"e": &mergeConflict{"wut", "now"},
			},
			true,
		},
	}
	for _, test := range tests {
		m, conflicts := merge(test.Old, test.A, test.B)
		if !reflect.DeepEqual(m, test.Result) {
			t.Errorf("merge(%v, %v, %v) = %v; want %v", test.Old, test.A, test.B, m, test.Result)
		}
		if conflicts != test.Conflicts {
			t.Errorf("merge(%v, %v, %v) conflicts = %t; want %t", test.Old, test.A, test.B, conflicts, test.Conflicts)
		}
	}
}

func BenchmarkNil(b *testing.B) {
	for i := 0; i < b.N; i++ {
		merge(nil, nil, nil)
	}
}

func BenchmarkSimple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		merge(1.0, 2.0, 2.0)
	}
}

func BenchmarkEmptyMap(b *testing.B) {
	old := map[string]interface{}{}
	obja := map[string]interface{}{}
	objb := map[string]interface{}{}
	for i := 0; i < b.N; i++ {
		merge(old, obja, objb)
	}
}

func BenchmarkMap(b *testing.B) {
	old := map[string]interface{}{
		"a": 47.0,
		"b": 3.0,
	}
	obja := map[string]interface{}{
		"a": 47.0,
		"b": 3.0,
		"c": "hi",
		"e": "wut",
	}
	objb := map[string]interface{}{
		"a": 47.0,
		"c": "hi",
		"d": "hey",
		"e": "now",
	}
	for i := 0; i < b.N; i++ {
		merge(old, obja, objb)
	}
}
