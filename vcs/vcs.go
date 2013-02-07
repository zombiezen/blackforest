// Package vcs provides an abstract interface for interacting with version control systems.
package vcs

import (
	"errors"
)

var errNotWC = errors.New("not a working copy")

// VCS is a version control system connector.
type VCS interface {
	IsWorkingCopy(path string) (bool, error)
	WorkingCopy(path string) (WorkingCopy, error)
}

// WorkingCopy is a filesystem directory that mirrors a version control repository.  Any paths given to this interface are filesystem paths relative to the directory (unless otherwise specified).
type WorkingCopy interface {
	// VCS returns the working copy's version control system.
	VCS() VCS

	// Path returns the absolute path to the root of the working copy.
	Path() string

	// Current returns the Rev the working copy is currently updated to.
	Current() (Rev, error)

	// Add adds files to the next commit.
	Add(paths []string) error

	// Commit creates a new changeset.
	// If files is nil, all dirty files will be committed.
	Commit(message string, files []string) error

	// Update updates the working copy to a specific changeset.
	// If the Rev is nil, then the most recent changeset is used.
	Update(rev Rev) error

	// ParseRev parses a Rev from a string.
	// A working copy implementation may request additional information from the
	// VCS to disambiguate changesets.
	ParseRev(s string) (Rev, error)
}

// A Rev is a unique identifier for a changeset.
// The Rev method should return a string that uniquely identifies a changeset
// across working copies.
//
// The String method can return the same string as Rev, but in some VCSs
// (like Git or Mercurial), the full identifier is not ideal for display.
// String should return a more user-friendly string, which might not uniquely
// identify a changeset.
type Rev interface {
	Rev() string
	String() string
}
