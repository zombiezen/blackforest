// The glados command provides a front-end for a GLaDOS catalog.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"bitbucket.org/zombiezen/glados/catalog"
	"bitbucket.org/zombiezen/glados/vcs"
	"bitbucket.org/zombiezen/subcmd"
)

// Environment variable names
const (
	CatalogPathEnv = "GLADOS_PATH"
	HostEnv        = "GLADOS_HOST"
	EditorEnv      = "GLADOS_EDITOR"

	globalEditorEnv = "EDITOR"
)

// Global flags
var (
	catalogPath string = os.Getenv(CatalogPathEnv)
	host        string = os.Getenv(HostEnv)
	editor      string = "vi"
)

func init() {
	if e := os.Getenv(EditorEnv); e != "" {
		editor = e
	} else if e := os.Getenv(globalEditorEnv); e != "" {
		editor = e
	}
}

func main() {
	if err := commandSet.Do(os.Args[1:]); err == nil {
		os.Exit(exitSuccess)
	} else if code, ok := err.(exitError); ok {
		os.Exit(int(code))
	} else if err == flag.ErrHelp {
		os.Exit(exitUsage)
	} else if _, ok := err.(subcmd.CommandError); ok {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitUsage)
	} else if _, ok := err.(usageError); ok {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitUsage)
	} else {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitFailure)
	}
}

func requireCatalog() catalog.Catalog {
	if catalogPath == "" {
		panic(errCatalogPathNotSet)
	}

	var v vcs.VCS
	wc, err := vcs.OpenWorkingCopy(catalogPath)
	if wc != nil {
		v = wc.VCS()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "catalog VCS warning:", err)
	}

	cat, err := catalog.Open(catalogPath, v)
	if err != nil {
		panic(err)
	}
	return cat
}

var (
	errEmptyName           = errors.New("empty name")
	errDanglingVCSURL      = errors.New("-vcsurl given, but project has no VCS")
	errCatalogPathNotSet   = errors.New(CatalogPathEnv + " not set")
	errHostNotSet          = errors.New(HostEnv + " not set")
	errHostNotSetPathGiven = errors.New("-path given and " + HostEnv + " not set")

	errFailed         error = exitError(exitFailure)
	errTagsMutexFlags error = usageError("cannot use -tags flag with -addtags/-deltags")
)

type projectHasPathError struct {
	ShortName string
	Path      string
}

func (e *projectHasPathError) Error() string {
	return string(e.ShortName) + " already has path: " + e.Path + "\n(use -overwritepath to force)"
}

type noVCSURLError string

func (e noVCSURLError) Error() string {
	return "project " + string(e) + " has no VCS URL"
}

type badVCSError string

func (e badVCSError) Error() string {
	return string(e) + " is not a valid VCS name\nvalid choices are: " + validVCSText
}

type exitError int

func (e exitError) Error() string {
	return fmt.Sprint("exit code", int(e))
}

type usageError string

func (e usageError) Error() string {
	return string(e)
}

// exit codes
const (
	exitSuccess = 0
	exitFailure = 1
	exitUsage   = 64
)

func globalFlags(fset *flag.FlagSet) {
	fset.StringVar(&catalogPath, "catalog", catalogPath, "path to catalog directory (overrides the "+CatalogPathEnv+" environment variable)")
	fset.StringVar(&host, "host", host, "key for this host (overrides the "+HostEnv+" environment variable)")
	fset.StringVar(&editor, "editor", editor, "text editor (overrides the "+EditorEnv+" environment variable)")
}

func parseFlags(fset *flag.FlagSet, args []string) {
	if err := fset.Parse(args[1:]); err == flag.ErrHelp {
		panic(err)
	} else if err != nil {
		panic(exitError(exitUsage))
	}
}

var knownVCS = []struct {
	Name string
	Impl vcs.VCS
}{
	{catalog.CVS, nil},
	{catalog.Subversion, new(vcs.Subversion)},
	{catalog.Mercurial, new(vcs.Mercurial)},
	{catalog.Git, nil},
	{catalog.Bazaar, new(vcs.Bazaar)},
	{catalog.Darcs, nil},
}

var validVCSText string

func init() {
	names := make([]string, len(knownVCS))
	for i := range names {
		names[i] = knownVCS[i].Name
	}
	validVCSText = strings.Join(names, ", ")
}

func vcsImpl(t string) vcs.VCS {
	for _, v := range knownVCS {
		if t == v.Name {
			return v.Impl
		}
	}
	return nil
}

func isValidVCSType(t string) bool {
	for _, v := range knownVCS {
		if t == v.Name {
			return true
		}
	}
	return false
}
