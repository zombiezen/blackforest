package vcs

import (
	"encoding/hex"
	"errors"
	"os"
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

// Mercurial implements the VCS interface for interacting with Mercurial.
type Mercurial struct {
	// Program is the path of the Mercurial executable.
	Program string

	commander commander
}

var _ VCS = new(Mercurial)

// cmd creates a command for the given arguments.
func (hg *Mercurial) cmd(args ...string) command {
	prog := hg.Program
	if prog == "" {
		prog = "hg"
	}
	commander := hg.commander
	if commander == nil {
		commander = execCommander{}
	}
	return commander.command(prog, args...)
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

func (wc *mercurialWC) cmd(args ...string) command {
	c := wc.hg.cmd(args...)
	c.SetDir(wc.path)
	return c
}

func (wc *mercurialWC) identify(args ...string) (Rev, error) {
	const op = "identify"
	out, err := wc.cmd(append([]string{"identify", "--debug", "-i"}, args...)...).Output()
	if err != nil {
		return nil, &mercurialError{Op: op, Path: wc.path, Err: err}
	}
	rev, err := parseIdentifyOutput(out)
	if err != nil {
		return nil, &mercurialError{Op: op, Path: wc.path, Err: err}
	}
	return rev, nil
}

func (wc *mercurialWC) VCS() VCS {
	return wc.hg
}

func (wc *mercurialWC) Path() string {
	return wc.path
}

func (wc *mercurialWC) Current() (Rev, error) {
	return wc.identify()
}

func (wc *mercurialWC) Add(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	args := make([]string, len(paths)+1)
	args[0] = "add"
	for i, p := range paths {
		args[i+1] = "path:" + p
	}
	if err := wc.cmd(args...).Run(); err != nil {
		return &mercurialError{Op: "add", Path: wc.path, Err: err}
	}
	return nil
}

func (wc *mercurialWC) Commit(message string, files []string) error {
	var args []string
	if files == nil {
		args = []string{"commit", "-m", message}
	} else {
		if len(files) == 0 {
			return &mercurialError{Op: "commit", Path: wc.path, Err: errors.New("empty commit")}
		}
		args = make([]string, 0, len(files)+3)
		args = append(args, "commit", "-m", message)
		for _, f := range files {
			args = append(args, "path:"+f)
		}
	}
	if err := wc.cmd(args...).Run(); err != nil {
		return &mercurialError{Op: "commit", Path: wc.path, Err: err}
	}
	return nil
}

func (wc *mercurialWC) Update(rev Rev) error {
	var c command
	if rev == nil {
		c = wc.cmd("update")
	} else {
		// TODO: check if rev is type mercurialRev?
		c = wc.cmd("update", "-r", rev.Rev())
	}
	if err := c.Run(); err != nil {
		return &mercurialError{Op: "update", Path: wc.path, Err: err}
	}
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
	return wc.identify("-r", s)
}

type mercurialRev [mercurialRevSize]byte

func (r mercurialRev) Rev() string {
	return hex.EncodeToString(r[:])
}

func (r mercurialRev) String() string {
	return hex.EncodeToString(r[:6])
}

func parseIdentifyOutput(out []byte) (mercurialRev, error) {
	for i := len(out) - 1; i >= 0; i-- {
		if c := out[i]; c != '\n' && c != '+' {
			out = out[:i+1]
			break
		}
	}
	if len(out) != hex.EncodedLen(mercurialRevSize) {
		return mercurialRev{}, errors.New("wrong rev size")
	}
	var rev mercurialRev
	if _, err := hex.Decode(rev[:], out); err != nil {
		return mercurialRev{}, err
	}
	return rev, nil
}
