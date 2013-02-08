package vcs

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"
)

var (
	errMockCmd     = errors.New("mock command")
	errTooManyCmds = errors.New("too many commands")
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
	return nil
}

func (mc *mockCommand) Start() error {
	if mc.Bad {
		return errTooManyCmds
	}
	return errMockCmd
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
	return nil, errMockCmd
}

func (mc *mockCommand) Wait() error {
	if mc.Bad {
		return errTooManyCmds
	}
	return errMockCmd
}
