package vcs

import (
	"bytes"
	"errors"
	"io"
)

var errMockCmd = errors.New("mock command")

type mockCommander []mockCommand

func (mc *mockCommander) command(program string, args ...string) command {
	var c *mockCommand
	c, *mc = &(*mc)[0], (*mc)[1:]
	c.Args = append([]string{program}, args...)
	return c
}

type mockCommand struct {
	Out bytes.Buffer

	Args []string
	Dir  string
}

func (mc *mockCommand) SetDir(dir string) {
	mc.Dir = dir
}

func (mc *mockCommand) CombinedOutput() ([]byte, error) {
	// TODO: stderr
	return mc.Output()
}

func (mc *mockCommand) Output() ([]byte, error) {
	b := make([]byte, mc.Out.Len())
	copy(b, mc.Out.Bytes())
	return b, nil
}

func (mc *mockCommand) Run() error {
	mc.Out.Truncate(0)
	return nil
}

func (mc *mockCommand) Start() error {
	return errMockCmd
}

func (mc *mockCommand) StderrPipe() (io.ReadCloser, error) {
	return nil, errMockCmd
}

func (mc *mockCommand) StdinPipe() (io.WriteCloser, error) {
	return nil, errMockCmd
}

func (mc *mockCommand) StdoutPipe() (io.ReadCloser, error) {
	return nil, errMockCmd
}

func (mc *mockCommand) Wait() error {
	return errMockCmd
}
