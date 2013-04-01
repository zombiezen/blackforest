package vcs

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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

type commandVCS struct {
	vcs        VCS
	name       string
	program    string
	commander  commander
	specialDir string

	// Command names
	checkout    string
	remove      string
	rename      string
	renameFlags []string

	current  func(*commandWC) (Rev, error)
	parseRev func(*commandWC, string) (Rev, error)
}

func (c *commandVCS) init(program string) {
	if program != "" {
		c.program = program
	}
	if c.commander == nil {
		c.commander = execCommander{}
	}
}

// cmd creates a command for the given arguments.
func (c *commandVCS) cmd(args ...string) command {
	return c.commander.command(c.program, args...)
}

func (c *commandVCS) IsWorkingCopy(path string) (bool, error) {
	fi, err := os.Stat(filepath.Join(path, c.specialDir))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return fi.IsDir(), nil
}

func (c *commandVCS) WorkingCopy(path string) (WorkingCopy, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	ok, err := c.IsWorkingCopy(path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, &vcsError{Name: c.name, Op: "working copy", Path: path, Err: errNotWC}
	}
	return &commandWC{c: c, path: path}, nil
}

func (c *commandVCS) Checkout(url, path string) (WorkingCopy, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := c.runCheckout(url, path); err != nil {
		return nil, &vcsError{Name: c.name, Op: c.checkout, Path: path, Err: err}
	}
	return &commandWC{c: c, path: path}, nil
}

func (c *commandVCS) runCheckout(url, path string) error {
	return c.cmd(c.checkout, "--", url, path).Run()
}

type commandWC struct {
	c    *commandVCS
	path string
}

func (wc *commandWC) cmd(args ...string) command {
	c := wc.c.cmd(args...)
	c.SetDir(wc.path)
	return c
}

func (wc *commandWC) VCS() VCS {
	return wc.c.vcs
}

func (wc *commandWC) Path() string {
	return wc.path
}

func (wc *commandWC) Current() (Rev, error) {
	return wc.c.current(wc)
}

func (wc *commandWC) Add(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	args := make([]string, 0, len(paths)+2)
	args = append(args, "add", "--")
	args = append(args, paths...)
	if err := wc.cmd(args...).Run(); err != nil {
		return &vcsError{Name: wc.c.name, Op: "add", Path: wc.path, Err: err}
	}
	return nil
}

func (wc *commandWC) Remove(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	args := make([]string, 0, len(paths)+2)
	args = append(args, wc.c.remove, "--")
	args = append(args, paths...)
	if err := wc.cmd(args...).Run(); err != nil {
		return &vcsError{Name: wc.c.name, Op: wc.c.remove, Path: wc.path, Err: err}
	}
	return nil
}

func (wc *commandWC) Rename(src, dst string) error {
	args := make([]string, 0, 1+len(wc.c.renameFlags)+3)
	args = append(args, wc.c.rename)
	args = append(args, wc.c.renameFlags...)
	args = append(args, "--", src, dst)
	if err := wc.cmd(args...).Run(); err != nil {
		return &vcsError{Name: wc.c.name, Op: wc.c.rename, Path: wc.path, Err: err}
	}
	return nil
}

func (wc *commandWC) Commit(message string, files []string) error {
	var args []string
	if files == nil {
		args = []string{"commit", "-m", message}
	} else {
		if len(files) == 0 {
			return &vcsError{Name: wc.c.name, Op: "commit", Path: wc.path, Err: errors.New("empty commit")}
		}
		args = make([]string, 0, len(files)+4)
		args = append(args, "commit", "-m", message, "--")
		args = append(args, files...)
	}
	if err := wc.cmd(args...).Run(); err != nil {
		return &vcsError{Name: wc.c.name, Op: "commit", Path: wc.path, Err: err}
	}
	return nil
}

func (wc *commandWC) Update(rev Rev) error {
	var c command
	if rev == nil {
		c = wc.cmd("update")
	} else {
		c = wc.cmd("update", "-r", rev.Rev())
	}
	if err := c.Run(); err != nil {
		return &vcsError{Name: wc.c.name, Op: "update", Path: wc.path, Err: err}
	}
	return nil
}

func (wc *commandWC) ParseRev(s string) (Rev, error) {
	return wc.c.parseRev(wc, s)
}

type vcsError struct {
	Name string
	Op   string
	Path string
	Err  error
}

func (e *vcsError) Error() string {
	return e.Name + ": " + e.Op + " " + e.Path + ": " + e.Err.Error()
}
