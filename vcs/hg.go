package vcs

import (
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
)

const hgDir = ".hg"

type mercurialError struct {
	Op   string
	Path string
	Err  error
}

func (e *mercurialError) Error() string {
	return "hg: " + e.Op + " " + e.Path + ": " + e.Err.Error()
}

type Mercurial struct {
	// Program is the path of the Mercurial executable.
	Program string
}

var _ VCS = new(Mercurial)

// cmd creates an exec.Cmd for the given arguments.
func (hg *Mercurial) cmd(args ...string) *exec.Cmd {
	prog := hg.Program
	if prog == "" {
		prog = "hg"
	}
	return exec.Command(prog, args...)
}

func (hg *Mercurial) IsWorkingCopy(path string) (bool, error) {
	fi, err := os.Stat(filepath.Join(path, hgDir))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return fi.IsDir(), nil
}

func (hg *Mercurial) WorkingCopy(path string) (WorkingCopy, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	ok, err := hg.IsWorkingCopy(path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, &os.PathError{Op: "working copy", Path: path, Err: errNotWC}
	}
	return &mercurialWC{hg: hg, path: path}, nil
}

type mercurialWC struct {
	hg   *Mercurial
	path string
}

func (wc *mercurialWC) cmd(args ...string) *exec.Cmd {
	c := wc.hg.cmd(args...)
	c.Dir = wc.path
	return c
}

func (wc *mercurialWC) VCS() VCS {
	return wc.hg
}

func (wc *mercurialWC) Path() string {
	return wc.path
}

func (wc *mercurialWC) Current() (Rev, error) {
	// TODO
	return nil, nil
}

func (wc *mercurialWC) Add(path string) error {
	// TODO
	return nil
}

func (wc *mercurialWC) Commit(message string) (Rev, error) {
	// TODO
	return nil, nil
}

func (wc *mercurialWC) Update(rev Rev) error {
	// TODO
	return nil
}

const mercurialRevSize = 20

func (wc *mercurialWC) ParseRev(s string) (Rev, error) {
	if len(s) == hex.EncodedLen(mercurialRevSize) {
		var rev mercurialRev
		if _, err := hex.Decode(rev[:], []byte(s)); err == nil {
			return rev, nil
		}
	}

	out, err := wc.cmd("identify", "-i", "-r", s).Output()
	if err != nil {
		return nil, err
	}
	var rev mercurialRev
	if _, err := hex.Decode(rev[:], out); err != nil {
		return nil, err
	}
	return rev, nil
}

type mercurialRev [mercurialRevSize]byte

func (r mercurialRev) Rev() string {
	return hex.EncodeToString(r[:])
}

func (r mercurialRev) String() string {
	return hex.EncodeToString(r[:6])
}
