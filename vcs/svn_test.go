package vcs

import (
	"bytes"
	"testing"
)

const desiredSvnPath = "/wc"

var magicSvnRev = subversionRev(1302)

func newIsolatedSubversionWC(path string, c mockCommander) subversionWC {
	svn := Subversion{Program: "svn", commander: &c}
	svn.init()
	return subversionWC{&commandWC{c: &svn.c, path: path}}
}

func TestSubversionInit(t *testing.T) {
	wc := newIsolatedSubversionWC("/wc", mockCommander{})
	if wc.c.checkout != "checkout" {
		t.Errorf("wc.c.checkout = %q; want %q", wc.c.checkout, "checkout")
	}
	if wc.c.remove != "delete" {
		t.Errorf("wc.c.remove = %q; want %q", wc.c.remove, "delete")
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

func TestSubversionRename(t *testing.T) {
	mc := mockCommander{}
	wc := newIsolatedSubversionWC(desiredSvnPath, mc)
	err := wc.Rename("foo", "bar")
	mc.check(t)
	if err == nil {
		t.Errorf("wc.Rename(%q, %q) expected an error", "foo", "bar")
	}
}
