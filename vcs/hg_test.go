package vcs

import (
	"bytes"
	"testing"
)

const desiredHgPath = "/wc"

var magicHgRev = mercurialRev{0x0d, 0x9c, 0x2b, 0x3c, 0x7b, 0xce, 0x68, 0xef, 0x99, 0x50, 0xd2, 0x37, 0xea, 0xc5, 0xff, 0x67, 0xf1, 0x17, 0xbf, 0xf5}

func newIsolatedMercurialWC(path string, c mockCommander) *mercurialWC {
	return &mercurialWC{
		hg:   &Mercurial{Program: "hg", commander: &c},
		path: path,
	}
}

func TestMercurialCheckout(t *testing.T) {
	const (
		wcPath   = "baz"
		cloneURL = "http://example.com/foo/bar"
	)
	mc := mockCommander{
		{
			Out:        *bytes.NewBufferString(""),
			ExpectArgs: []string{"hg", "clone", "--", cloneURL, wcPath},
		},
	}
	c := mc
	hg := &Mercurial{Program: "hg", commander: &c}
	err := hg.checkout(cloneURL, wcPath)
	mc.check(t)
	if err != nil {
		t.Error("hg.checkout(%q, %q) error:", cloneURL, wcPath, err)
	}
}

func TestMercurialCurrent(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBufferString("0d9c2b3c7bce68ef9950d237eac5ff67f117bff5\n"),
			ExpectDir:  desiredHgPath,
			ExpectArgs: []string{"hg", "identify", "--debug", "-i"},
		},
	}
	wc := newIsolatedMercurialWC(desiredHgPath, mc)
	rev, err := wc.Current()
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Current() error: %v", err)
	}
	if r := magicHgRev; rev != r {
		t.Errorf("wc.Current() = %v; want %v", rev, r)
	}
}

func TestMercurialAdd(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBuffer([]byte{}),
			ExpectDir:  desiredHgPath,
			ExpectArgs: []string{"hg", "add", "path:foo", "path:bar"},
		},
	}
	wc := newIsolatedMercurialWC(desiredHgPath, mc)
	files := []string{"foo", "bar"}
	err := wc.Add(files)
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Add(%q) error: %v", files, err)
	}
}

func TestMercurialRemove(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBuffer([]byte{}),
			ExpectDir:  desiredHgPath,
			ExpectArgs: []string{"hg", "remove", "path:foo", "path:bar"},
		},
	}
	wc := newIsolatedMercurialWC(desiredHgPath, mc)
	files := []string{"foo", "bar"}
	err := wc.Remove(files)
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Remove(%q) error: %v", files, err)
	}
}

func TestMercurialRename(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBuffer([]byte{}),
			ExpectDir:  desiredHgPath,
			ExpectArgs: []string{"hg", "rename", "--after", "--", "foo", "bar"},
		},
	}
	wc := newIsolatedMercurialWC(desiredHgPath, mc)
	err := wc.Rename("foo", "bar")
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Rename(%q, %q) error: %v", "foo", "bar", err)
	}
}

func TestMercurialCommit(t *testing.T) {
	const commitMessage = "Hello, World!"

	// files==nil test
	{
		mc := mockCommander{
			{
				Out:        *bytes.NewBuffer([]byte{}),
				ExpectDir:  desiredHgPath,
				ExpectArgs: []string{"hg", "commit", "-m", commitMessage},
			},
		}
		wc := newIsolatedMercurialWC(desiredHgPath, mc)
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
				ExpectDir:  desiredHgPath,
				ExpectArgs: []string{"hg", "commit", "-m", commitMessage, "path:foo", "path:bar"},
			},
		}
		wc := newIsolatedMercurialWC(desiredHgPath, mc)
		files := []string{"foo", "bar"}
		err := wc.Commit("Hello, World!", files)
		mc.check(t)
		if err != nil {
			t.Errorf("wc.Commit(%q, %q) error: %v", commitMessage, files, err)
		}
	}
}

func TestMercurialUpdate(t *testing.T) {
	{
		mc := mockCommander{
			{
				Out:        *bytes.NewBuffer([]byte{}),
				ExpectDir:  desiredHgPath,
				ExpectArgs: []string{"hg", "update"},
			},
		}
		wc := newIsolatedMercurialWC(desiredHgPath, mc)
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
				ExpectDir:  desiredHgPath,
				ExpectArgs: []string{"hg", "update", "-r", magicHgRev.Rev()},
			},
		}
		wc := newIsolatedMercurialWC(desiredHgPath, mc)
		err := wc.Update(magicHgRev)
		mc.check(t)
		if err != nil {
			t.Errorf("wc.Update(%v) error: %v", magicHgRev, err)
		}
	}
}

func TestParseIdentifyOutput(t *testing.T) {
	tests := []struct {
		Arg   string
		Rev   mercurialRev
		Error bool
	}{
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff5", magicHgRev, false},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff5\n", magicHgRev, false},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff5+", magicHgRev, false},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff5+\n", magicHgRev, false},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff", mercurialRev{}, true},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff\n", mercurialRev{}, true},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff50", mercurialRev{}, true},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff50\n", mercurialRev{}, true},
	}
	for _, test := range tests {
		rev, err := parseIdentifyOutput([]byte(test.Arg))
		if err != nil && !test.Error {
			t.Errorf("parseIdentifyOutput(%q) error: %v", test.Arg, err)
		} else if err == nil && test.Error {
			t.Errorf("parseIdentifyOutput(%q) expected an error", test.Arg)
		}
		if rev != test.Rev {
			t.Errorf("parseIdentifyOutput(%q) = %v; want %v", test.Arg, rev.Rev(), test.Rev.Rev())
		}
	}
}
