package main

import (
	"reflect"
	"testing"
	"time"

	"bitbucket.org/zombiezen/glados/catalog"
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

func TestUpdateForm(t *testing.T) {
	magicTime := time.Date(2013, 2, 7, 10, 51, 13, 0, time.FixedZone("PST", int(-8*time.Hour/time.Second)))

	proj := new(catalog.Project)
	err := updateProjectForm(proj, map[string][]string{
		"name":        {"Hello, World!"},
		"shortname":   {"hello"},
		"tags":        {"foo,bar,foo"},
		"description": {"Greetings, Program!"},
		"path":        {"/usr/src/hello"},
		"created":     {"2013-02-07T10:51:13-08:00"},
		"url":         {"http://example.com/"},
		"vcs":         {"svn"},
		"vcsurl":      {"http://example.com/svn/trunk/"},
	}, "foo")
	if err != nil {
		t.Error("error:", err)
	}

	if want := "Hello, World!"; proj.Name != want {
		t.Errorf("proj.Name = %q; want %q", proj.Name, want)
	}
	if want := "hello"; proj.ShortName != want {
		t.Errorf("proj.ShortName = %q; want %q", proj.ShortName, want)
	}
	if want := (catalog.TagSet{"foo", "bar"}); !reflect.DeepEqual(proj.Tags, want) {
		t.Errorf("proj.Tags = %q; want %q", []string(proj.Tags), []string(want))
	}
	if want := "Greetings, Program!"; proj.Description != want {
		t.Errorf("proj.Description = %q; want %q", proj.Description, want)
	}
	if host, want := "foo", "/usr/src/hello"; proj.Path(host) != want {
		t.Errorf("proj.Path(%q) = %q; want %q", host, proj.Path(host), want)
	}
	if !proj.CreateTime.Equal(magicTime) {
		t.Errorf("proj.CreateTime = %v; want %v", proj.CreateTime, magicTime)
	}
	if want := "http://example.com/"; proj.Homepage != want {
		t.Errorf("proj.Homepage = %q; want %q", proj.Homepage, want)
	}
	if proj.VCS == nil {
		t.Error("proj.VCS == nil")
	} else {
		if want := "svn"; proj.VCS.Type != want {
			t.Errorf("proj.VCS.Type = %q; want %q", proj.VCS.Type, want)
		}
		if want := "http://example.com/svn/trunk/"; proj.VCS.URL != want {
			t.Errorf("proj.VCS.URL = %q; want %q", proj.VCS.URL, want)
		}
	}
}
