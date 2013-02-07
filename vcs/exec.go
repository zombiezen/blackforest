package vcs

import (
	"io"
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
