package vcs

import (
	"encoding/xml"
	"errors"
	"os"
	"path/filepath"
	"strconv"
)

const svnDir = ".svn"

type subversionError struct {
	Op   string
	Path string
	Err  error
}

func (e *subversionError) Error() string {
	return "svn: " + e.Op + " " + e.Path + ": " + e.Err.Error()
}

// Subversion implements the VCS interface for interacting with Subversion.
type Subversion struct {
	// Program is the path of the Subversion executable.
	Program string

	commander commander
}

var _ VCS = new(Subversion)

// cmd creates a command for the given arguments.
func (svn *Subversion) cmd(args ...string) command {
	prog := svn.Program
	if prog == "" {
		prog = "svn"
	}
	commander := svn.commander
	if commander == nil {
		commander = execCommander{}
	}
	return commander.command(prog, args...)
}

func (svn *Subversion) IsWorkingCopy(path string) (bool, error) {
	fi, err := os.Stat(filepath.Join(path, svnDir))
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return fi.IsDir(), nil
}

func (svn *Subversion) WorkingCopy(path string) (WorkingCopy, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	ok, err := svn.IsWorkingCopy(path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, &os.PathError{Op: "working copy", Path: path, Err: errNotWC}
	}
	return &subversionWC{svn: svn, path: path}, nil
}

func (svn *Subversion) Checkout(url, path string) (WorkingCopy, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := svn.checkout(url, path); err != nil {
		return nil, err
	}
	return &subversionWC{svn: svn, path: path}, nil
}

func (svn *Subversion) checkout(url, path string) error {
	if err := svn.cmd("checkout", "--", url, path).Run(); err != nil {
		return &subversionError{Op: "checkout", Path: path, Err: err}
	}
	return nil
}

type subversionWC struct {
	svn  *Subversion
	path string
}

func (wc *subversionWC) cmd(args ...string) command {
	c := wc.svn.cmd(args...)
	c.SetDir(wc.path)
	return c
}

func (wc *subversionWC) info(v interface{}, args ...string) error {
	const op = "info"
	c := wc.cmd(append([]string{"info", "--xml"}, args...)...)
	r, err := c.StdoutPipe()
	if err != nil {
		return &subversionError{Op: op, Path: wc.path, Err: err}
	}
	defer r.Close()
	if err := c.Start(); err != nil {
		return &subversionError{Op: op, Path: wc.path, Err: err}
	}
	xmlErr := xml.NewDecoder(r).Decode(v)
	cmdErr := c.Wait()
	if cmdErr != nil {
		return &subversionError{Op: op, Path: wc.path, Err: cmdErr}
	} else if xmlErr != nil {
		return &subversionError{Op: op, Path: wc.path, Err: xmlErr}
	}
	return nil
}

func (wc *subversionWC) VCS() VCS {
	return wc.svn
}

func (wc *subversionWC) Path() string {
	return wc.path
}

func (wc *subversionWC) Current() (Rev, error) {
	var v struct {
		Entry struct {
			Revision int `xml:"revision,attr"`
		} `xml:"entry"`
	}
	if err := wc.info(&v); err != nil {
		return nil, err
	}
	return subversionRev(v.Entry.Revision), nil
}

func (wc *subversionWC) Add(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	args := make([]string, len(paths)+2)
	args[0] = "add"
	args[1] = "--"
	for i, p := range paths {
		args[i+2] = p
	}
	if err := wc.cmd(args...).Run(); err != nil {
		return &subversionError{Op: "add", Path: wc.path, Err: err}
	}
	return nil
}

func (wc *subversionWC) Remove(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	args := make([]string, len(paths)+2)
	args[0] = "delete"
	args[1] = "--"
	for i, p := range paths {
		args[i+2] = p
	}
	if err := wc.cmd(args...).Run(); err != nil {
		return &subversionError{Op: "delete", Path: wc.path, Err: err}
	}
	return nil
}

func (wc *subversionWC) Rename(src, dst string) error {
	// TODO(light): find a safe way to perform renames afterward
	return &subversionError{Op: "rename", Path: wc.path, Err: errors.New("rename not supported")}
}

func (wc *subversionWC) Commit(message string, files []string) error {
	var args []string
	if files == nil {
		args = []string{"commit", "-m", message}
	} else {
		if len(files) == 0 {
			return &subversionError{Op: "commit", Path: wc.path, Err: errors.New("empty commit")}
		}
		args = make([]string, 0, len(files)+4)
		args = append(args, "commit", "-m", message, "--")
		for _, f := range files {
			args = append(args, f)
		}
	}
	if err := wc.cmd(args...).Run(); err != nil {
		return &subversionError{Op: "commit", Path: wc.path, Err: err}
	}
	return nil
}

func (wc *subversionWC) Update(rev Rev) error {
	var c command
	if rev == nil {
		c = wc.cmd("update")
	} else {
		// TODO: check if rev is type subversionRev?
		c = wc.cmd("update", "-r", rev.Rev())
	}
	if err := c.Run(); err != nil {
		return &subversionError{Op: "update", Path: wc.path, Err: err}
	}
	return nil
}

func (wc *subversionWC) ParseRev(s string) (Rev, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}
	return subversionRev(n), nil
}

type subversionRev int

func (r subversionRev) Rev() string {
	return strconv.Itoa(int(r))
}

func (r subversionRev) String() string {
	return r.Rev()
}
