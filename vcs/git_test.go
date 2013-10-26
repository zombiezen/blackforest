package vcs

import (
	"bytes"
	"testing"
)

const desiredGitPath = "/wc"

var magicGitRev = gitRev{0x0d, 0x9c, 0x2b, 0x3c, 0x7b, 0xce, 0x68, 0xef, 0x99, 0x50, 0xd2, 0x37, 0xea, 0xc5, 0xff, 0x67, 0xf1, 0x17, 0xbf, 0xf5}

func newIsolatedGitWC(path string, c mockCommander) gitWC {
	git := &Git{Program: "git"}
	git.init()
	git.c.commander = &c
	return gitWC{&commandWC{c: &git.c, path: path}}
}

func TestGitInit(t *testing.T) {
	wc := newIsolatedGitWC(desiredGitPath, mockCommander{})
	if want := ".git"; wc.c.specialDir != want {
		t.Errorf("wc.c.specialDir = %q; want %q", wc.c.specialDir, want)
	}
	if wc.c.checkout != "clone" {
		t.Errorf("wc.c.checkout = %q; want %q", wc.c.checkout, "clone")
	}
	if wc.c.remove != "rm" {
		t.Errorf("wc.c.remove = %q; want %q", wc.c.remove, "remove")
	}
}

func TestGitCurrent(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBufferString("0d9c2b3c7bce68ef9950d237eac5ff67f117bff5\n"),
			ExpectDir:  desiredGitPath,
			ExpectArgs: []string{"git", "rev-parse", "HEAD"},
		},
	}
	wc := newIsolatedGitWC(desiredGitPath, mc)
	rev, err := wc.Current()
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Current() error: %v", err)
	}
	if r := magicGitRev; rev != r {
		t.Errorf("wc.Current() = %v; want %v", rev, r)
	}
}

func TestGitUpdate(t *testing.T) {
	{
		mc := mockCommander{
			{
				Out:        *bytes.NewBuffer([]byte{}),
				ExpectDir:  desiredGitPath,
				ExpectArgs: []string{"git", "checkout", "master"},
			},
		}
		wc := newIsolatedGitWC(desiredGitPath, mc)
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
				ExpectDir:  desiredGitPath,
				ExpectArgs: []string{"git", "checkout", magicGitRev.Rev()},
			},
		}
		wc := newIsolatedGitWC(desiredGitPath, mc)
		err := wc.Update(magicGitRev)
		mc.check(t)
		if err != nil {
			t.Errorf("wc.Update(%v) error: %v", magicGitRev, err)
		}
	}
}

func TestGitAdd(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBuffer([]byte{}),
			ExpectDir:  desiredGitPath,
			ExpectArgs: []string{"git", "add", "--", "foo", "bar"},
		},
	}
	wc := newIsolatedGitWC(desiredGitPath, mc)
	files := []string{"foo", "bar"}
	err := wc.Add(files)
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Add(%q) error: %v", files, err)
	}
}

func TestGitRemove(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBuffer([]byte{}),
			ExpectDir:  desiredGitPath,
			ExpectArgs: []string{"git", "rm", "--", "foo", "bar"},
		},
	}
	wc := newIsolatedGitWC(desiredGitPath, mc)
	files := []string{"foo", "bar"}
	err := wc.Remove(files)
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Remove(%q) error: %v", files, err)
	}
}

func TestGitRename(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBuffer([]byte{}),
			ExpectDir:  desiredGitPath,
			ExpectArgs: []string{"git", "add", "--", "bar"},
		},
		{
			Out:        *bytes.NewBuffer([]byte{}),
			ExpectDir:  desiredGitPath,
			ExpectArgs: []string{"git", "rm", "--", "foo"},
		},
	}
	wc := newIsolatedGitWC(desiredGitPath, mc)
	from := "foo"
	to := "bar"
	err := wc.Rename(from, to)
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Rename(%s, %s) error: %v", from, to, err)
	}
}

func TestGitCommit(t *testing.T) {
	const commitMessage = "Hello, World!"

	// files==nil test
	{
		mc := mockCommander{
			{
				Out:        *bytes.NewBuffer([]byte{}),
				ExpectDir:  desiredGitPath,
				ExpectArgs: []string{"git", "add", "--update"},
			},
			{
				Out:        *bytes.NewBuffer([]byte{}),
				ExpectDir:  desiredGitPath,
				ExpectArgs: []string{"git", "commit", "-m", commitMessage},
			},
		}
		wc := newIsolatedGitWC(desiredGitPath, mc)
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
				ExpectDir:  desiredGitPath,
				ExpectArgs: []string{"git", "commit", "-m", commitMessage, "--", "foo", "bar"},
			},
		}
		wc := newIsolatedGitWC(desiredGitPath, mc)
		files := []string{"foo", "bar"}
		err := wc.Commit("Hello, World!", files)
		mc.check(t)
		if err != nil {
			t.Errorf("wc.Commit(%q, %q) error: %v", commitMessage, files, err)
		}
	}

	// first command fails test
	{
		mc := mockCommander{
			{
				Out:        *bytes.NewBuffer([]byte{}),
				ExpectDir:  desiredGitPath,
				ExpectArgs: []string{"git", "add", "--update"},
				Fail:       true,
			},
		}
		wc := newIsolatedGitWC(desiredGitPath, mc)
		err := wc.Commit("Hello, World!", nil)
		mc.check(t)
		if err != errFailCmd {
			t.Errorf("wc.Commit(%q, nil) expected error errFailCmd but passed", commitMessage)
		}
	}
}

func TestParseRevParseOutput(t *testing.T) {
	tests := []struct {
		Arg   string
		Rev   gitRev
		Error bool
	}{
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff5", magicGitRev, false},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff5\n", magicGitRev, false},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff5aaa", gitRev{}, true},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff", gitRev{}, true},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff\n", gitRev{}, true},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff50", gitRev{}, true},
		{"0d9c2b3c7bce68ef9950d237eac5ff67f117bff50\n", gitRev{}, true},
		{"z0d9c2b3c7bce68ef9950d237eac5ff67f117bff5\n", gitRev{}, true},
	}
	for _, test := range tests {
		rev, err := parseRevParseOutput([]byte(test.Arg))
		if err != nil && !test.Error {
			t.Errorf("parseRevParseOutput(%q) error: %v", test.Arg, err)
		} else if err == nil && test.Error {
			t.Errorf("parseRevParseOutput(%q) expected an error", test.Arg)
		}
		if rev != test.Rev {
			t.Errorf("parseRevParseOutput(%q) = %v; want %v", test.Arg, rev.Rev(), test.Rev.Rev())
		}
	}
}
