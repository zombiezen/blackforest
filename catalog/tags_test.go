package catalog

import (
	"reflect"
	"testing"
)

func TestTagSetAdd(t *testing.T) {
	tests := []struct {
		Input []string
		Tag   string

		Ok     bool
		Output []string
	}{
		{[]string{}, "foo", true, []string{"foo"}},
		{[]string{"foo"}, "foo", false, []string{"foo"}},
		{[]string{"foo"}, "bar", true, []string{"foo", "bar"}},
		{[]string{"foo", "bar"}, "bar", false, []string{"foo", "bar"}},
	}
	for _, test := range tests {
		ts := make(TagSet, len(test.Input))
		copy(ts, test.Input)
		ok := ts.Add(test.Tag)
		if ok != test.Ok {
			t.Errorf("TagSet(%q).Add(%q) = %t; want %t", test.Input, test.Tag, ok, test.Ok)
		}
		if !reflect.DeepEqual([]string(ts), test.Output) {
			t.Errorf("TagSet(%q).Add(%q) is %q; want %q", test.Input, test.Tag, []string(ts), test.Output)
		}
	}
}

func TestTagSetRemove(t *testing.T) {
	tests := []struct {
		Input []string
		Tag   string

		Ok     bool
		Output []string
	}{
		{[]string{}, "foo", false, []string{}},
		{[]string{"foo"}, "foo", true, []string{}},
		{[]string{"foo"}, "bar", false, []string{"foo"}},
		{[]string{"foo", "foo"}, "foo", true, []string{}},
		{[]string{"foo", "bar"}, "foo", true, []string{"bar"}},
		{[]string{"foo", "bar"}, "bar", true, []string{"foo"}},
	}
	for _, test := range tests {
		ts := make(TagSet, len(test.Input))
		copy(ts, test.Input)
		ok := ts.Remove(test.Tag)
		if ok != test.Ok {
			t.Errorf("TagSet(%q).Remove(%q) = %t; want %t", test.Input, test.Tag, ok, test.Ok)
		}
		if !reflect.DeepEqual([]string(ts), test.Output) {
			t.Errorf("TagSet(%q).Remove(%q) is %q; want %q", test.Input, test.Tag, []string(ts), test.Output)
		}
	}
}

func TestTagSetUnique(t *testing.T) {
	tests := []struct {
		Input  []string
		Output []string
	}{
		{[]string{}, []string{}},
		{[]string{"foo"}, []string{"foo"}},
		{[]string{"foo", "foo"}, []string{"foo"}},
		{[]string{"foo", "bar", "foo"}, []string{"foo", "bar"}},
	}
	for _, test := range tests {
		ts := make(TagSet, len(test.Input))
		copy(ts, test.Input)
		ts.Unique()
		if !reflect.DeepEqual([]string(ts), test.Output) {
			t.Errorf("TagSet(%q).Unique() is %q; want %q", test.Input, []string(ts), test.Output)
		}
	}
}
