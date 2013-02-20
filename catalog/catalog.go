// Package catalog provides access to GLaDOS catalogs.
package catalog

import (
	"errors"
	"strconv"
	"strings"
)

// A Catalog is a database of projects.
type Catalog interface {
	// List returns a list of all the project short names in the catalog.
	List() ([]string, error)

	// GetProject fetches the project record with the given short name.
	GetProject(shortName string) (*Project, error)

	// PutProject stores a project record.
	PutProject(project *Project) error

	// DelProject removes a project record from the catalog.
	DelProject(shortName string) error

	// ShortName returns the short name for the given ID.
	// If the ID is not in the catalog, this method returns an empty string with no error.
	ShortName(id ID) (string, error)
}

// Project is the metadata associated with a project.
type Project struct {
	ID          ID       `json:"id"`
	ShortName   string   `json:"shortname"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Homepage    string   `json:"homepage,omitempty"`

	VCS     *VCSInfo             `json:"vcs,omitempty"`
	PerHost map[string]*HostInfo `json:"per_host,omitempty"`
}

// VCSInfo holds the version control information in a project record.
type VCSInfo struct {
	Type string `json:"type"`
	URL  string `json:"url,omitempty"`
}

// VCS types (for VCSInfo)
const (
	CVS        = "cvs"
	Git        = "git"
	Bazaar     = "bzr"
	Darcs      = "darcs"
	Subversion = "svn"
	Mercurial  = "hg"
)

// HostInfo holds the per-host project information.
type HostInfo struct {
	Path string `json:"path"`
}

// Errors
var (
	ErrLocked = errors.New("catalog is locked")
)

// VersionError is returned when opening a catalog from an incompatible version
// of GLaDOS.
type VersionError int

func (e VersionError) Error() string {
	return "incompatible catalog (version " + strconv.Itoa(int(e)) + ")"
}

func isValidShortName(shortName string) bool {
	if len(shortName) == 0 {
		return false
	}
	return strings.IndexFunc(shortName, func(r rune) bool {
		return !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_')
	}) == -1
}

// shortNameError is returned when a short name passed to a catalog is invalid.
type shortNameError string

func (e shortNameError) Error() string {
	return `bad project short name: "` + string(e) + `"`
}

// projectError is returned when an error occurs for a particular project.
type projectError struct {
	ShortName string
	Op        string
	Err       error
}

func (e *projectError) Error() string {
	return "catalog: " + e.Op + " project " + e.ShortName + ": " + e.Err.Error()
}
