package main

import (
	"bitbucket.org/zombiezen/blackforest/vcs"
	"testing"
)

func TestVCSImpl(t *testing.T) {
	var vc vcs.VCS

	if vc = vcsImpl("cvs"); vc != nil {
		t.Errorf(`vcsImpl("cvs") is %T; want nil`, vc)
	}
	vc = vcsImpl("svn")
	if _, ok := vc.(*vcs.Subversion); !ok {
		t.Errorf(`vcsImpl("svn") is %T; want *vcs.Subversion`, vc)
	}
	vc = vcsImpl("hg")
	if _, ok := vc.(*vcs.Mercurial); !ok {
		t.Errorf(`vcsImpl("hg") is %T; want *vcs.Mercurial`, vc)
	}
	if vc = vcsImpl("git"); vc != nil {
		t.Errorf(`vcsImpl("git") is %T; want nil`, vc)
	}
	vc = vcsImpl("bzr")
	if _, ok := vc.(*vcs.Bazaar); !ok {
		t.Errorf(`vcsImpl("bzr") is %T; want *vcs.Bazaar`, vc)
	}
	if vc = vcsImpl("darcs"); vc != nil {
		t.Errorf(`vcsImpl("darcs") is %T; want nil`, vc)
	}

	if vc = vcsImpl("foo"); vc != nil {
		t.Errorf(`vcsImpl("foo") is %T; want nil`, vc)
	}
}

func TestValidVCSType(t *testing.T) {
	tests := []struct {
		Name  string
		Valid bool
	}{
		{"cvs", true},
		{"svn", true},
		{"hg", true},
		{"git", true},
		{"bzr", true},
		{"darcs", true},
		{"foo", false},
	}

	for _, test := range tests {
		valid := isValidVCSType(test.Name)
		if valid != test.Valid {
			t.Errorf("isValidVCSType(%q) = %t; want %t", test.Name, valid, test.Valid)
		}
	}
}
