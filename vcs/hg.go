package vcs

import (
	"encoding/hex"
	"errors"
)

// Mercurial implements the VCS interface for interacting with Mercurial.
type Mercurial struct {
	// Program is the path of the Mercurial executable.
	Program string

	c commandVCS
}

func (hg *Mercurial) init() {
	hg.c = commandVCS{
		vcs:         hg,
		name:        "hg",
		program:     "hg",
		specialDir:  ".hg",
		checkout:    "clone",
		remove:      "remove",
		rename:      "rename",
		renameFlags: []string{"--after"},
		current: func(wc *commandWC) (Rev, error) {
			return hgIdentify(wc)
		},
		parseRev: func(wc *commandWC, s string) (Rev, error) {
			if len(s) == hex.EncodedLen(mercurialRevSize) {
				var rev mercurialRev
				if _, err := hex.Decode(rev[:], []byte(s)); err == nil {
					return rev, nil
				}
			}
			return hgIdentify(wc, "-r", s)
		},
	}
	hg.c.init(hg.Program)
}

func (hg *Mercurial) IsWorkingCopy(path string) (bool, error) {
	hg.init()
	return hg.c.IsWorkingCopy(path)
}

func (hg *Mercurial) WorkingCopy(path string) (WorkingCopy, error) {
	hg.init()
	wc, err := hg.c.WorkingCopy(path)
	if wc != nil {
		wc = mercurialWC{wc.(*commandWC)}
	}
	return wc, err
}

func (hg *Mercurial) Checkout(url, path string) (WorkingCopy, error) {
	hg.init()
	wc, err := hg.c.Checkout(url, path)
	if wc != nil {
		wc = mercurialWC{wc.(*commandWC)}
	}
	return wc, err
}

func hgIdentify(wc *commandWC, args ...string) (Rev, error) {
	const op = "identify"
	out, err := wc.cmd(append([]string{"identify", "--debug", "-i"}, args...)...).Output()
	if err != nil {
		return nil, &vcsError{Name: wc.c.name, Op: op, Path: wc.path, Err: err}
	}
	rev, err := parseHgIdentifyOutput(out)
	if err != nil {
		return nil, &vcsError{Name: wc.c.name, Op: op, Path: wc.path, Err: err}
	}
	return rev, nil
}

type mercurialWC struct {
	*commandWC
}

func (wc mercurialWC) Add(paths []string) error {
	if len(paths) == 0 {
		return wc.commandWC.Add(paths)
	}
	p := make([]string, len(paths))
	for i := range paths {
		p[i] = "path:" + paths[i]
	}
	return wc.commandWC.Add(p)
}

func (wc mercurialWC) Remove(paths []string) error {
	if len(paths) == 0 {
		return wc.commandWC.Remove(paths)
	}
	p := make([]string, len(paths))
	for i := range paths {
		p[i] = "path:" + paths[i]
	}
	return wc.commandWC.Remove(p)
}

func (wc mercurialWC) Commit(message string, files []string) error {
	if len(files) == 0 {
		return wc.commandWC.Commit(message, files)
	}
	f := make([]string, len(files))
	for i := range files {
		f[i] = "path:" + files[i]
	}
	return wc.commandWC.Commit(message, f)
}

const mercurialRevSize = 20

type mercurialRev [mercurialRevSize]byte

func (r mercurialRev) Rev() string {
	return hex.EncodeToString(r[:])
}

func (r mercurialRev) String() string {
	return hex.EncodeToString(r[:6])
}

func parseHgIdentifyOutput(out []byte) (mercurialRev, error) {
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
