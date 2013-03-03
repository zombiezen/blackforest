package vcs

import (
	"bytes"
	"testing"
)

const desiredSvnPath = "/wc"

var magicSvnRev = subversionRev(1302)

func newIsolatedSubversionWC(path string, c mockCommander) *subversionWC {
	return &subversionWC{
		svn:  &Subversion{Program: "svn", commander: &c},
		path: path,
	}
}

func TestSubversionCheckout(t *testing.T) {
	const (
		wcPath   = "baz"
		cloneURL = "http://example.com/foo/bar"
	)
	mc := mockCommander{
		{
			Out:        *bytes.NewBufferString(""),
			ExpectArgs: []string{"svn", "checkout", "--", cloneURL, wcPath},
		},
	}
	c := mc
	svn := &Subversion{Program: "svn", commander: &c}
	err := svn.checkout(cloneURL, wcPath)
	mc.check(t)
	if err != nil {
		t.Error("svn.checkout(%q, %q) error:", cloneURL, wcPath, err)
	}
}

func TestSubversionCurrent(t *testing.T) {
	mc := mockCommander{
		{
			Out: *bytes.NewBufferString(`<?xml version="1.0" encoding="UTF-8"?>
<info>
<entry
   kind="dir"
   path="."
   revision="1302">
</entry>
</info>
`),
			ExpectDir:  desiredSvnPath,
			ExpectArgs: []string{"svn", "info", "--xml"},
		},
	}
	wc := newIsolatedSubversionWC(desiredSvnPath, mc)
	rev, err := wc.Current()
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Current() error: %v", err)
	}
	if r := subversionRev(magicSvnRev); rev != r {
		t.Errorf("wc.Current() = %v; want %v", rev, r)
	}
}

func TestSubversionAdd(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBuffer([]byte{}),
			ExpectDir:  desiredSvnPath,
			ExpectArgs: []string{"svn", "add", "--", "foo", "bar"},
		},
	}
	wc := newIsolatedSubversionWC(desiredSvnPath, mc)
	files := []string{"foo", "bar"}
	err := wc.Add(files)
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Add(%q) error: %v", files, err)
	}
}

func TestSubversionRemove(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBuffer([]byte{}),
			ExpectDir:  desiredSvnPath,
			ExpectArgs: []string{"svn", "delete", "--", "foo", "bar"},
		},
	}
	wc := newIsolatedSubversionWC(desiredSvnPath, mc)
	files := []string{"foo", "bar"}
	err := wc.Remove(files)
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Remove(%q) error: %v", files, err)
	}
}

func TestSubversionRename(t *testing.T) {
	mc := mockCommander{}
	wc := newIsolatedSubversionWC(desiredSvnPath, mc)
	err := wc.Rename("foo", "bar")
	mc.check(t)
	if err == nil {
		t.Errorf("wc.Rename(%q, %q) expected an error", "foo", "bar")
	}
}

func TestSubversionCommit(t *testing.T) {
	const commitMessage = "Hello, World!"

	// files==nil test
	{
		mc := mockCommander{
			{
				Out:        *bytes.NewBuffer([]byte{}),
				ExpectDir:  desiredSvnPath,
				ExpectArgs: []string{"svn", "commit", "-m", commitMessage},
			},
		}
		wc := newIsolatedSubversionWC(desiredSvnPath, mc)
		err := wc.Commit("Hello, World!", nil)
		mc.check(t)
		if err != nil {
			t.Errorf("wc.Commit(%q, nil) error: %v", commitMessage, err)
		}
	}

	// files!=nil test
	{
		mc := mockCommander{
			{
				Out:        *bytes.NewBuffer([]byte{}),
				ExpectDir:  desiredSvnPath,
				ExpectArgs: []string{"svn", "commit", "-m", commitMessage, "--", "foo", "bar"},
			},
		}
		wc := newIsolatedSubversionWC(desiredSvnPath, mc)
		files := []string{"foo", "bar"}
		err := wc.Commit("Hello, World!", files)
		mc.check(t)
		if err != nil {
			t.Errorf("wc.Commit(%q, %q) error: %v", commitMessage, files, err)
		}
	}
}

func TestSubversionUpdate(t *testing.T) {
	{
		mc := mockCommander{
			{
				Out:        *bytes.NewBuffer([]byte{}),
				ExpectDir:  desiredSvnPath,
				ExpectArgs: []string{"svn", "update"},
			},
		}
		wc := newIsolatedSubversionWC(desiredSvnPath, mc)
		err := wc.Update(nil)
		mc.check(t)
		if err != nil {
			t.Errorf("wc.Update(nil) error: %v", err)
		}
	}

	{
		mc := mockCommander{
			{
				Out:        *bytes.NewBuffer([]byte{}),
				ExpectDir:  desiredSvnPath,
				ExpectArgs: []string{"svn", "update", "-r", "1302"},
			},
		}
		wc := newIsolatedSubversionWC(desiredSvnPath, mc)
		err := wc.Update(magicSvnRev)
		mc.check(t)
		if err != nil {
			t.Errorf("wc.Update(%v) error: %v", magicSvnRev, err)
		}
	}
}
