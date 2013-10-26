package vcs

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"reflect"
	"testing"
)

var (
	errMockCmd     = errors.New("mock command")
	errTooManyCmds = errors.New("too many commands")
	errFailCmd     = errors.New("command failed")
)

type mockCommander []mockCommand

func (mc *mockCommander) command(program string, args ...string) command {
	if len(*mc) == 0 {
		return &mockCommand{Bad: true}
	}
	var c *mockCommand
	c, *mc = &(*mc)[0], (*mc)[1:]
	c.Args = append([]string{program}, args...)
	return c
}

func (mc mockCommander) check(t *testing.T) {
	for i := range mc {
		c := &mc[i]
		if c.Dir != c.ExpectDir {
			t.Errorf("command[%d]: cd = %v; want %v", i, c.Dir, c.ExpectDir)
		}
		if !reflect.DeepEqual(c.Args, c.ExpectArgs) {
			t.Errorf("command[%d]: args = %q; want %q", i, c.Args, c.ExpectArgs)
		}
	}
}

type mockCommand struct {
	Out        bytes.Buffer
	ExpectArgs []string
	ExpectDir  string

	Args []string
	Dir  string

	Bad bool

	// If Fail is true, then Run or Wait will return errFailCmd.
	Fail bool
}

func (mc *mockCommand) SetDir(dir string) {
	mc.Dir = dir
}

func (mc *mockCommand) CombinedOutput() ([]byte, error) {
	if mc.Bad {
		return nil, errTooManyCmds
	}
	// TODO: stderr
	return mc.Output()
}

func (mc *mockCommand) Output() ([]byte, error) {
	if mc.Bad {
		return nil, errTooManyCmds
	}
	b := make([]byte, mc.Out.Len())
	copy(b, mc.Out.Bytes())
	return b, nil
}

func (mc *mockCommand) Run() error {
	if mc.Bad {
		return errTooManyCmds
	}
	mc.Out.Truncate(0)

	if mc.Fail {
		return errFailCmd
	}
	return nil
}

func (mc *mockCommand) Start() error {
	if mc.Bad {
		return errTooManyCmds
	}
	return nil
}

func (mc *mockCommand) StderrPipe() (io.ReadCloser, error) {
	if mc.Bad {
		return nil, errTooManyCmds
	}
	return nil, errMockCmd
}

func (mc *mockCommand) StdinPipe() (io.WriteCloser, error) {
	if mc.Bad {
		return nil, errTooManyCmds
	}
	return nil, errMockCmd
}

func (mc *mockCommand) StdoutPipe() (io.ReadCloser, error) {
	if mc.Bad {
		return nil, errTooManyCmds
	}
	return ioutil.NopCloser(&mc.Out), nil
}

func (mc *mockCommand) Wait() error {
	if mc.Bad {
		return errTooManyCmds
	}

	if mc.Fail {
		return errFailCmd
	}
	return nil
}

func newCommandWC(path string, mc mockCommander) *commandWC {
	return &commandWC{
		path: path,
		c: &commandVCS{
			name:       "commandVCS",
			specialDir: ".CMD",
			program:    "CMD",
			commander:  &mc,

			checkout:    "CMDCHECKOUT",
			remove:      "CMDREMOVE",
			rename:      "CMDRENAME",
			renameFlags: []string{"--foo"},
		},
	}
}

func TestCommandVCSCheckout(t *testing.T) {
	const (
		wcPath   = "baz"
		cloneURL = "http://example.com/foo/bar"
	)
	mc := mockCommander{
		{
			Out:        *bytes.NewBufferString(""),
			ExpectArgs: []string{"CMD", "CMDCHECKOUT", "--", cloneURL, wcPath},
		},
	}
	c := newCommandWC(wcPath, mc).c
	err := c.runCheckout(cloneURL, wcPath)
	mc.check(t)
	if err != nil {
		t.Errorf("commandVCS.checkout(%q, %q) error: %v", cloneURL, wcPath, err)
	}
}

func TestCommandWCAdd(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBuffer([]byte{}),
			ExpectDir:  "/wc",
			ExpectArgs: []string{"CMD", "add", "--", "foo", "bar"},
		},
	}
	wc := newCommandWC("/wc", mc)
	files := []string{"foo", "bar"}
	err := wc.Add(files)
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Add(%q) error: %v", files, err)
	}
}

func TestCommandWCRemove(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBuffer([]byte{}),
			ExpectDir:  "/wc",
			ExpectArgs: []string{"CMD", "CMDREMOVE", "--", "foo", "bar"},
		},
	}
	wc := newCommandWC("/wc", mc)
	files := []string{"foo", "bar"}
	err := wc.Remove(files)
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Add(%q) error: %v", files, err)
	}
}

func TestCommandWCRename(t *testing.T) {
	mc := mockCommander{
		{
			Out:        *bytes.NewBuffer([]byte{}),
			ExpectDir:  "/wc",
			ExpectArgs: []string{"CMD", "CMDRENAME", "--foo", "--", "foo", "bar"},
		},
	}
	wc := newCommandWC("/wc", mc)
	err := wc.Rename("foo", "bar")
	mc.check(t)
	if err != nil {
		t.Errorf("wc.Rename(%q, %q) error: %v", "foo", "bar", err)
	}
}

func TestCommandWCCommit(t *testing.T) {
	const commitMessage = "Hello, World!"

	// files==nil test
	{
		mc := mockCommander{
			{
				Out:        *bytes.NewBuffer([]byte{}),
				ExpectDir:  "/wc",
				ExpectArgs: []string{"CMD", "commit", "-m", commitMessage},
			},
		}
		wc := newCommandWC("/wc", mc)
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
				ExpectDir:  "/wc",
				ExpectArgs: []string{"CMD", "commit", "-m", commitMessage, "--", "foo", "bar"},
			},
		}
		wc := newCommandWC("/wc", mc)
		files := []string{"foo", "bar"}
		err := wc.Commit("Hello, World!", files)
		mc.check(t)
		if err != nil {
			t.Errorf("wc.Commit(%q, %q) error: %v", commitMessage, files, err)
		}
	}
}

type mockRev string

func (rev mockRev) Rev() string    { return string(rev) }
func (rev mockRev) String() string { return "STRING" + string(rev) }

func TestCommandWCUpdate(t *testing.T) {
	const magicRev mockRev = "xyzzy"

	{
		mc := mockCommander{
			{
				Out:        *bytes.NewBuffer([]byte{}),
				ExpectDir:  "/wc",
				ExpectArgs: []string{"CMD", "update"},
			},
		}
		wc := newCommandWC("/wc", mc)
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
				ExpectDir:  "/wc",
				ExpectArgs: []string{"CMD", "update", "-r", string(magicRev)},
			},
		}
		wc := newCommandWC("/wc", mc)
		err := wc.Update(magicRev)
		mc.check(t)
		if err != nil {
			t.Errorf("wc.Update(%v) error: %v", magicRev, err)
		}
	}
}
