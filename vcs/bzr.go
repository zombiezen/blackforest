package vcs

import (
	"bytes"
	"errors"
)

// Bazaar implements the VCS interface for interacting with Bazaar.
type Bazaar struct {
	// Program is the path of the Bazaar executable.
	Program string

	commander commander
	c         commandVCS
}

var _ VCS = new(Bazaar)

func (bzr *Bazaar) init() {
	if bzr.c.vcs == nil {
		bzr.c.init(bzr, "bzr", bzr.Program, bzr.commander)
		bzr.c.specialDir = ".bzr"
		bzr.c.checkout = "branch"
		bzr.c.remove = "remove"
		bzr.c.rename = "mv"
		bzr.c.renameFlags = []string{"--after"}
		bzr.c.current = func(wc *commandWC) (Rev, error) {
			return bzrVersionInfo(wc)
		}
		bzr.c.parseRev = func(wc *commandWC, s string) (Rev, error) {
			return bzrVersionInfo(wc, "-r", s)
		}
	}
}

func (bzr *Bazaar) IsWorkingCopy(path string) (bool, error) {
	bzr.init()
	return bzr.c.IsWorkingCopy(path)
}

func (bzr *Bazaar) WorkingCopy(path string) (WorkingCopy, error) {
	bzr.init()
	return bzr.c.WorkingCopy(path)
}

func (bzr *Bazaar) Checkout(url, path string) (WorkingCopy, error) {
	bzr.init()
	return bzr.c.Checkout(url, path)
}

func bzrVersionInfo(wc *commandWC, args ...string) (Rev, error) {
	const op = "version-info"
	args = append([]string{"version-info", "--custom", "--template={revno}\n{revision_id}"}, args...)
	out, err := wc.cmd(args...).Output()
	if err != nil {
		return nil, &vcsError{Name: wc.c.name, Op: op, Path: wc.path, Err: err}
	}
	rev, err := parseBzrVersionInfo(out)
	if err != nil {
		return nil, &vcsError{Name: wc.c.name, Op: op, Path: wc.path, Err: err}
	}
	return rev, nil
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
