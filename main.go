// The glados command provides a front-end for a GLaDOS catalog.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"bitbucket.org/zombiezen/glados/catalog"
	"bitbucket.org/zombiezen/glados/vcs"
	"bitbucket.org/zombiezen/subcmd"
)

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
