package vcs

import (
	"bytes"
	"reflect"
	"testing"
)

const desiredBzrPath = "/wc"

var magicBzrRev = bazaarRev{"john@example.com-20100303001707-e0f5uz51ddzrlag0", "42.1.1"}

func newIsolatedBazaarWC(path string, c mockCommander) *commandWC {
	bzr := Bazaar{Program: "bzr", commander: &c}
	bzr.init()
	return &commandWC{c: &bzr.c, path: path}
}

func TestBazaarInit(t *testing.T) {
	wc := newIsolatedBazaarWC("/wc", mockCommander{})
	if want := ".bzr"; wc.c.specialDir != want {
		t.Errorf("wc.c.specialDir = %q; want %q", wc.c.specialDir, want)
	}
	if wc.c.checkout != "branch" {
		t.Errorf("wc.c.checkout = %q; want %q", wc.c.checkout, "branch")
	}
	if wc.c.remove != "remove" {
		t.Errorf("wc.c.remove = %q; want %q", wc.c.remove, "remove")
	}
	if wc.c.rename != "mv" {
		t.Errorf("wc.c.rename = %q; want %q", wc.c.rename, "mv")
	}
	if want := ([]string{"--after"}); !reflect.DeepEqual(wc.c.renameFlags, want) {
		t.Errorf("wc.c.rename = %q; want %q", wc.c.renameFlags, want)
	}
}

func TestBazaarCurrent(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBufferString("42.1.1\njohn@example.com-20100303001707-e0f5uz51ddzrlag0"),
			ExpectDir:  desiredBzrPath,
			ExpectArgs: []string{"bzr", "version-info", "--custom", "--template={revno}\n{revision_id}"},
		},
	}
	wc := newIsolatedBazaarWC(desiredBzrPath, mc)
	rev, err := wc.Current()
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Current() error: %v", err)
	}
	if r := magicBzrRev; rev != r {
		t.Errorf("wc.Current() = %#v; want %#v", rev, r)
	}
}

func TestParseBzrVersionInfo(t *testing.T) {
	tests := []struct {
		Arg   string
		Rev   bazaarRev
		Error bool
	}{
		{"42.1.1\njohn@example.com-20100303001707-e0f5uz51ddzrlag0", magicBzrRev, false},
		{"42.1.1john@example.com-20100303001707-e0f5uz51ddzrlag0", bazaarRev{}, true},
	}
	for _, test := range tests {
		rev, err := parseBzrVersionInfo([]byte(test.Arg))
		if err != nil && !test.Error {
			t.Errorf("parseBzrVersionInfo(%q) error: %v", test.Arg, err)
		} else if err == nil && test.Error {
			t.Errorf("parseBzrVersionInfo(%q) expected an error", test.Arg)
		}
		if rev != test.Rev {
			t.Errorf("parseBzrVersionInfo(%q) = %#v; want %#v", test.Arg, rev, test.Rev)
		}
	}
}
