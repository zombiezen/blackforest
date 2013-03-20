package vcs

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
)

const bzrDir = ".bzr"

type bazaarError struct {
	Op   string
	Path string
	Err  error
}

func (e *bazaarError) Error() string {
	return "bzr: " + e.Op + " " + e.Path + ": " + e.Err.Error()
}

// Bazaar implements the VCS interface for interacting with Bazaar.
type Bazaar struct {
	// Program is the path of the Bazaar executable.
	Program string

	commander commander
}

var _ VCS = new(Bazaar)

// cmd creates a command for the given arguments.
func (bzr *Bazaar) cmd(args ...string) command {
	prog := bzr.Program
	if prog == "" {
		prog = "bzr"
	}
	commander := bzr.commander
	if commander == nil {
		commander = execCommander{}
	}
	return commander.command(prog, args...)
}

func (bzr *Bazaar) IsWorkingCopy(path string) (bool, error) {
	fi, err := os.Stat(filepath.Join(path, bzrDir))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return fi.IsDir(), nil
}

func (bzr *Bazaar) WorkingCopy(path string) (WorkingCopy, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	ok, err := bzr.IsWorkingCopy(path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, &os.PathError{Op: "working copy", Path: path, Err: errNotWC}
	}
	return &bazaarWC{bzr: bzr, path: path}, nil
}

func (bzr *Bazaar) Checkout(url, path string) (WorkingCopy, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := bzr.checkout(url, path); err != nil {
		return nil, err
	}
	return &bazaarWC{bzr: bzr, path: path}, nil
}

func (bzr *Bazaar) checkout(url, path string) error {
	if err := bzr.cmd("branch", "--", url, path).Run(); err != nil {
		return &bazaarError{Op: "branch", Path: path, Err: err}
	}
	return nil
}

type bazaarWC struct {
	bzr  *Bazaar
	path string
}

func (wc *bazaarWC) cmd(args ...string) command {
	c := wc.bzr.cmd(args...)
	c.SetDir(wc.path)
	return c
}

func (wc *bazaarWC) versionInfo(args ...string) (Rev, error) {
	const op = "version-info"
	args = append([]string{"version-info", "--custom", "--template={revno}\n{revision_id}"}, args...)
	out, err := wc.cmd(args...).Output()
	if err != nil {
		return nil, &bazaarError{Op: op, Path: wc.path, Err: err}
	}
	rev, err := parseBzrVersionInfo(out)
	if err != nil {
		return nil, &bazaarError{Op: op, Path: wc.path, Err: err}
	}
	return rev, nil
}

func (wc *bazaarWC) VCS() VCS {
	return wc.bzr
}

func (wc *bazaarWC) Path() string {
	return wc.path
}

func (wc *bazaarWC) Current() (Rev, error) {
	return wc.versionInfo()
}

func (wc *bazaarWC) Add(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	args := make([]string, 0, len(paths)+2)
	args = append(args, "add", "--")
	args = append(args, paths...)
	if err := wc.cmd(args...).Run(); err != nil {
		return &bazaarError{Op: "add", Path: wc.path, Err: err}
	}
	return nil
}

func (wc *bazaarWC) Remove(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	args := make([]string, 0, len(paths)+2)
	args = append(args, "remove", "--")
	args = append(args, paths...)
	if err := wc.cmd(args...).Run(); err != nil {
		return &bazaarError{Op: "remove", Path: wc.path, Err: err}
	}
	return nil
}

func (wc *bazaarWC) Rename(src, dst string) error {
	if err := wc.cmd("mv", "--after", "--", src, dst).Run(); err != nil {
		return &bazaarError{Op: "rename", Path: wc.path, Err: err}
	}
	return nil
}

func (wc *bazaarWC) Commit(message string, files []string) error {
	var args []string
	if files == nil {
		args = []string{"commit", "-m", message}
	} else {
		if len(files) == 0 {
			return &bazaarError{Op: "commit", Path: wc.path, Err: errors.New("empty commit")}
		}
		args = make([]string, 0, len(files)+4)
		args = append(args, "commit", "-m", message, "--")
		args = append(args, files...)
	}
	if err := wc.cmd(args...).Run(); err != nil {
		return &bazaarError{Op: "commit", Path: wc.path, Err: err}
	}
	return nil
}

func (wc *bazaarWC) Update(rev Rev) error {
	var c command
	if rev == nil {
		c = wc.cmd("update")
	} else {
		// TODO: check if rev is type bazaarRev?
		c = wc.cmd("update", "-r", rev.Rev())
	}
	if err := c.Run(); err != nil {
		return &bazaarError{Op: "update", Path: wc.path, Err: err}
	}
	return nil
}

func (wc *bazaarWC) ParseRev(s string) (Rev, error) {
	return wc.versionInfo("-r", s)
}

type bazaarRev struct {
	ID  string
	Num string
}

func (r bazaarRev) Rev() string {
	return r.ID
}

func (r bazaarRev) String() string {
	return r.Num
}

func parseBzrVersionInfo(out []byte) (bazaarRev, error) {
	i := bytes.IndexByte(out, '\n')
	if i == -1 {
		return bazaarRev{}, errors.New("no newline in output")
	}
	return bazaarRev{
		Num: string(out[:i]),
		ID:  string(out[i+1:]),
	}, nil
}
