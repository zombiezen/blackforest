package vcs

import (
	"bytes"
	"testing"
)

const desiredBzrPath = "/wc"

var magicBzrRev = bazaarRev{"john@example.com-20100303001707-e0f5uz51ddzrlag0", "42.1.1"}

func newIsolatedBazaarWC(path string, c mockCommander) *bazaarWC {
	return &bazaarWC{
		bzr:  &Bazaar{Program: "bzr", commander: &c},
		path: path,
	}
}

func TestBazaarCheckout(t *testing.T) {
	const (
		wcPath   = "baz"
		cloneURL = "http://example.com/foo/bar"
	)
	mc := mockCommander{
		{
			Out:        *bytes.NewBufferString(""),
			ExpectArgs: []string{"bzr", "branch", "--", cloneURL, wcPath},
		},
	}
	c := mc
	bzr := &Bazaar{Program: "bzr", commander: &c}
	err := bzr.checkout(cloneURL, wcPath)
	mc.check(t)
	if err != nil {
		t.Errorf("bzr.checkout(%q, %q) error: %v", cloneURL, wcPath, err)
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

func TestBazaarAdd(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBuffer([]byte{}),
			ExpectDir:  desiredBzrPath,
			ExpectArgs: []string{"bzr", "add", "--", "foo", "bar"},
		},
	}
	wc := newIsolatedBazaarWC(desiredBzrPath, mc)
	files := []string{"foo", "bar"}
	err := wc.Add(files)
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Add(%q) error: %v", files, err)
	}
}

func TestBazaarRemove(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBuffer([]byte{}),
			ExpectDir:  desiredBzrPath,
			ExpectArgs: []string{"bzr", "remove", "--", "foo", "bar"},
		},
	}
	wc := newIsolatedBazaarWC(desiredBzrPath, mc)
	files := []string{"foo", "bar"}
	err := wc.Remove(files)
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Remove(%q) error: %v", files, err)
	}
}

func TestBazaarRename(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBuffer([]byte{}),
			ExpectDir:  desiredBzrPath,
			ExpectArgs: []string{"bzr", "mv", "--after", "--", "foo", "bar"},
		},
	}
	wc := newIsolatedBazaarWC(desiredBzrPath, mc)
	err := wc.Rename("foo", "bar")
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Rename(%q, %q) error: %v", "foo", "bar", err)
	}
}

func TestBazaarCommit(t *testing.T) {
	const commitMessage = "Hello, World!"

	// files==nil test
	{
		mc := mockCommander{
			{
				Out:        *bytes.NewBuffer([]byte{}),
				ExpectDir:  desiredBzrPath,
				ExpectArgs: []string{"bzr", "commit", "-m", commitMessage},
			},
		}
		wc := newIsolatedBazaarWC(desiredBzrPath, mc)
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
				ExpectDir:  desiredBzrPath,
				ExpectArgs: []string{"bzr", "commit", "-m", commitMessage, "--", "foo", "bar"},
			},
		}
		wc := newIsolatedBazaarWC(desiredBzrPath, mc)
		files := []string{"foo", "bar"}
		err := wc.Commit("Hello, World!", files)
		mc.check(t)
		if err != nil {
			t.Errorf("wc.Commit(%q, %q) error: %v", commitMessage, files, err)
		}
	}
}

func TestBazaarUpdate(t *testing.T) {
	{
		mc := mockCommander{
			{
				Out:        *bytes.NewBuffer([]byte{}),
				ExpectDir:  desiredBzrPath,
				ExpectArgs: []string{"bzr", "update"},
			},
		}
		wc := newIsolatedBazaarWC(desiredBzrPath, mc)
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
				ExpectDir:  desiredBzrPath,
				ExpectArgs: []string{"bzr", "update", "-r", magicBzrRev.Rev()},
			},
		}
		wc := newIsolatedBazaarWC(desiredBzrPath, mc)
		err := wc.Update(magicBzrRev)
		mc.check(t)
		if err != nil {
			t.Errorf("wc.Update(%v) error: %v", magicBzrRev, err)
		}
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
