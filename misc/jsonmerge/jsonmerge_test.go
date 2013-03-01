package main

import (
	"reflect"
	"testing"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		Old, A, B interface{}
		Result    interface{}
	}{
		{nil, nil, nil, nil},
		{"", "", "", ""},
		{false, false, false, false},
		{false, true, true, true},
		{true, false, false, false},
		{true, true, true, true},
		{1.0, 1.0, 1.0, 1.0},
		{1.0, 2.0, 2.0, 2.0},
		{1.0, 2.0, 1.0, 2.0},
		{1.0, 1.0, 2.0, 2.0},
		{1.0, 2.0, 3.0, mergeConflict{2.0, 3.0}},
		{1.0, 3.0, 2.0, mergeConflict{3.0, 2.0}},
		{"a", "a", "a", "a"},
		{"a", "b", "b", "b"},
		{"a", "b", "a", "b"},
		{"a", "a", "b", "b"},
		{"a", "b", "c", mergeConflict{"b", "c"}},
		{"a", "c", "b", mergeConflict{"c", "b"}},
		{"foo", "foo", 42.0, 42.0},
		{"foo", 42.0, "foo", 42.0},
		{"foo", "bar", 42.0, mergeConflict{"bar", 42.0}},
		{"foo", 42.0, "bar", mergeConflict{42.0, "bar"}},
		{
			map[string]interface{}{},
			map[string]interface{}{},
			map[string]interface{}{},
			map[string]interface{}{},
		},
		{
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 0.0},
		},
		{
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 1.0},
			map[string]interface{}{"a": 1.0},
			map[string]interface{}{"a": 1.0},
		},
		{
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 1.0},
			map[string]interface{}{"a": 2.0},
			map[string]interface{}{"a": mergeConflict{1.0, 2.0}},
		},
		{
			map[string]interface{}{},
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{"a": 0.0},
		},
		{
			map[string]interface{}{},
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{},
			map[string]interface{}{"a": 0.0},
		},
		{
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{},
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{},
		},
		{
			map[string]interface{}{"a": 0.0},
			map[string]interface{}{},
			map[string]interface{}{"a": 1.0},
			map[string]interface{}{"a": mergeConflict{nil, 1.0}},
		},
	}
	for _, test := range tests {
		m := merge(test.Old, test.A, test.B)
		if !reflect.DeepEqual(m, test.Result) {
			t.Errorf("merge(%v, %v, %v) = %v; want %v", test.Old, test.A, test.B, m, test.Result)
		}
	}
}
