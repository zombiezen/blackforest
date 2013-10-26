package vcs

import (
	"encoding/hex"
	"errors"
)

// Git implements the VCS interface for interacting with Git.
type Git struct {
	// Program is the path of the Git executable.
	Program string

	c commandVCS
}

var _ VCS = new(Git)

func (git *Git) init() {
	git.c = commandVCS{
		vcs:        git,
		name:       "git",
		program:    "git",
		specialDir: ".git",
		checkout:   "clone",
		remove:     "rm",
		current: func(wc *commandWC) (Rev, error) {
			return gitCommitHash(wc, "HEAD")
		},
		parseRev: func(wc *commandWC, s string) (Rev, error) {
			if len(s) == hex.EncodedLen(gitRevSize) {
				var rev gitRev
				if _, err := hex.Decode(rev[:], []byte(s)); err == nil {
					return rev, nil
				}
			}
			return gitCommitHash(wc, s)
		},
	}
	git.c.init(git.Program)
}

func (git *Git) IsWorkingCopy(path string) (bool, error) {
	git.init()
	return git.c.IsWorkingCopy(path)
}

func (git *Git) WorkingCopy(path string) (WorkingCopy, error) {
	git.init()
	wc, err := git.c.WorkingCopy(path)
	if wc != nil {
		wc = gitWC{wc.(*commandWC)}
	}
	return wc, err
}

func (git *Git) Checkout(url, path string) (WorkingCopy, error) {
	git.init()
	wc, err := git.c.Checkout(url, path)
	if wc != nil {
		wc = gitWC{wc.(*commandWC)}
	}
	return wc, err
}

func gitCommitHash(wc *commandWC, arg string) (Rev, error) {
	const op = "rev-parse"

	out, err := wc.cmd([]string{"rev-parse", arg}...).Output()
	if err != nil {
		return nil, &vcsError{Name: wc.c.name, Op: op, Path: wc.path, Err: err}
	}

	rev, err := parseRevParseOutput(out)
	if err != nil {
		return nil, &vcsError{Name: wc.c.name, Op: op, Path: wc.path, Err: err}
	}
	return rev, nil
}

type gitWC struct {
	*commandWC
}

func (wc gitWC) Rename(src, dst string) error {
	if err := wc.Add([]string{dst}); err != nil {
		return err
	}
	if err = wc.Remove([]string{src}); err != nil {
		return err
	}
	return nil
}

func (wc gitWC) Commit(message string, files []string) error {
	if files == nil {
		// `git add --update` updates already tracked files
		if err := wc.cmd([]string{"add", "--update"}...).Run(); err != nil {
			return err
		}
	}

	return wc.commandWC.Commit(message, files)
}

func (wc gitWC) Update(rev Rev) error {
	if rev == nil {
		// TODO(adam): this may not always be correct, but Update isn't used yet.
		// Fix if it comes into use
		if err := wc.cmd([]string{"checkout", "master"}...).Run(); err != nil {
			return err
		}
	} else {
		if err := wc.cmd([]string{"checkout", rev.Rev()}...).Run(); err != nil {
			return err
		}
	}

	return nil
}

const gitRevSize = 20

type gitRev [gitRevSize]byte

func (r gitRev) Rev() string {
	return hex.EncodeToString(r[:])
}

func (r gitRev) String() string {
	return hex.EncodeToString(r[:4])[:7]
}

func parseRevParseOutput(out []byte) (gitRev, error) {
	for i := len(out) - 1; i >= 0; i-- {
		if c := out[i]; c != '\n' {
			out = out[:i+1]
			break
		}
	}
	if len(out) != hex.EncodedLen(gitRevSize) {
		return gitRev{}, errors.New("wrong rev size")
	}
	var rev gitRev
	if _, err := hex.Decode(rev[:], out); err != nil {
		return gitRev{}, err
	}
	return rev, nil
}
