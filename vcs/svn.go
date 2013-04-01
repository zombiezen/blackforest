package vcs

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strconv"
)

const svnDir = ".svn"

// Subversion implements the VCS interface for interacting with Subversion.
type Subversion struct {
	// Program is the path of the Subversion executable.
	Program string

	commander commander
	c         commandVCS
}

var _ VCS = new(Subversion)

func (svn *Subversion) init() {
	if svn.c.vcs == nil {
		svn.c.init(svn, "svn", svn.Program, svn.commander)
		svn.c.checkout = "checkout"
		svn.c.remove = "delete"
		svn.c.isWC = func(path string) (bool, error) {
			fi, err := os.Stat(filepath.Join(path, svnDir))
			if err != nil {
				if os.IsNotExist(err) {
					return false, nil
				}
				return false, err
			}
			return fi.IsDir(), nil
		}
		svn.c.current = func(wc *commandWC) (Rev, error) {
			var v struct {
				Entry struct {
					Revision int `xml:"revision,attr"`
				} `xml:"entry"`
			}
			if err := svnInfo(wc, &v); err != nil {
				return nil, err
			}
			return subversionRev(v.Entry.Revision), nil
		}
		svn.c.parseRev = func(wc *commandWC, s string) (Rev, error) {
			n, err := strconv.Atoi(s)
			if err != nil {
				return nil, err
			}
			return subversionRev(n), nil
		}
	}
}

func (svn *Subversion) IsWorkingCopy(path string) (bool, error) {
	svn.init()
	return svn.c.IsWorkingCopy(path)
}

func (svn *Subversion) WorkingCopy(path string) (WorkingCopy, error) {
	wc, err := svn.c.WorkingCopy(path)
	if wc != nil {
		wc = subversionWC{wc.(*commandWC)}
	}
	return wc, err
}

func (svn *Subversion) Checkout(url, path string) (WorkingCopy, error) {
	wc, err := svn.c.Checkout(url, path)
	if wc != nil {
		wc = subversionWC{wc.(*commandWC)}
	}
	return wc, err
}

func svnInfo(wc *commandWC, v interface{}, args ...string) error {
	const op = "info"
	c := wc.cmd(append([]string{"info", "--xml"}, args...)...)
	r, err := c.StdoutPipe()
	if err != nil {
		return &vcsError{Name: wc.c.name, Op: op, Path: wc.path, Err: err}
	}
	defer r.Close()
	if err := c.Start(); err != nil {
		return &vcsError{Name: wc.c.name, Op: op, Path: wc.path, Err: err}
	}
	xmlErr := xml.NewDecoder(r).Decode(v)
	cmdErr := c.Wait()
	if cmdErr != nil {
		return &vcsError{Name: wc.c.name, Op: op, Path: wc.path, Err: cmdErr}
	} else if xmlErr != nil {
		return &vcsError{Name: wc.c.name, Op: op, Path: wc.path, Err: xmlErr}
	}
	return nil
}

type subversionWC struct {
	*commandWC
}

func (wc subversionWC) Rename(src, dst string) error {
	// TODO(light): find a safe way to perform renames afterward
	return &vcsError{Name: wc.c.name, Op: "move", Path: wc.path, Err: errors.New("rename not supported")}
}

type subversionRev int

func (r subversionRev) Rev() string {
	return strconv.Itoa(int(r))
}

func (r subversionRev) String() string {
	return r.Rev()
}
