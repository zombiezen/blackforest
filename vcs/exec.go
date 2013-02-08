package vcs

import (
	"io"
	"os/exec"
)

type commander interface {
	command(program string, args ...string) command
}

type command interface {
	SetDir(dir string)

	CombinedOutput() ([]byte, error)
	Output() ([]byte, error)
	Run() error
	Start() error
	StderrPipe() (io.ReadCloser, error)
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.ReadCloser, error)
	Wait() error
}

type execCommander struct{}

func (execCommander) command(program string, args ...string) command {
	return execCmd{exec.Command(program, args...)}
}

type execCmd struct {
	*exec.Cmd
}

func (e execCmd) SetDir(dir string) {
	e.Cmd.Dir = dir
}
