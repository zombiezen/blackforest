package main

import (
	"reflect"
	"testing"
)

func TestProjectFormConstants(t *testing.T) {
	tests := []struct {
		ConstantName string
		Constant     string
		FieldName    string
	}{
		{"projectFormNameKey", projectFormNameKey, "Name"},
		{"projectFormShortNameKey", projectFormShortNameKey, "ShortName"},
		{"projectFormTagsKey", projectFormTagsKey, "Tags"},
		{"projectFormAddTagsKey", projectFormAddTagsKey, "AddTags"},
		{"projectFormDelTagsKey", projectFormDelTagsKey, "DelTags"},
		{"projectFormDescriptionKey", projectFormDescriptionKey, "Description"},
		{"projectFormPathKey", projectFormPathKey, "Path"},
		{"projectFormCreateTimeKey", projectFormCreateTimeKey, "CreateTime"},
		{"projectFormHomepageKey", projectFormHomepageKey, "Homepage"},
		{"projectFormVCSTypeKey", projectFormVCSTypeKey, "VCSType"},
		{"projectFormVCSURLKey", projectFormVCSURLKey, "VCSURL"},
	}
	tp := reflect.TypeOf(projectForm{})
	for _, test := range tests {
		f, ok := tp.FieldByName(test.FieldName)
		if !ok {
			t.Error("no such field:", test.FieldName)
		} else if tag := f.Tag.Get("schema"); test.Constant != tag {
			t.Errorf("%s == %q; want %q", test.ConstantName, test.Constant, tag)
		}
	}
}
