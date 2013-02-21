package main

import (
	"bitbucket.org/zombiezen/glados/catalog"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// environment variable names
const (
	CatalogPathEnv = "GLADOS_PATH"
	HostEnv        = "GLADOS_HOST"
)

// global flags
var (
	catalogPath string = os.Getenv(CatalogPathEnv)
	host        string = os.Getenv(HostEnv)
)

type subcmd struct {
	Func        func(*subcmd, []string)
	Name        string
	Aliases     []string
	Synopsis    string
	Description string
}

func (cmd *subcmd) NewFlagSet() *flag.FlagSet {
	fset := flag.NewFlagSet(cmd.Name, flag.ContinueOnError)
	globalFlags(fset)
	fset.Usage = func() {
		cmd.PrintUsage(fset)
	}
	fset.SetOutput(os.Stdout)
	return fset
}

// Matches returns whether name is one of the recognized forms of cmd.
func (cmd *subcmd) Matches(name string) bool {
	if name == cmd.Name {
		return true
	}
	for _, a := range cmd.Aliases {
		if name == a {
			return true
		}
	}
	return false
}

func (cmd *subcmd) PrintUsage(fset *flag.FlagSet) {
	if cmd.Synopsis != "" {
		fmt.Printf("glados %s\n\n", cmd.Synopsis)
	} else {
		fmt.Printf("Usage of %s:\n\n", cmd.Name)
	}
	if cmd.Description != "" {
		fmt.Print(cmd.Description + "\n\n")
	}
	fmt.Print("options:\n\n")
	fset.PrintDefaults()
}

func (cmd *subcmd) ExitSynopsis() {
	fmt.Fprintln(os.Stderr, "usage: glados", cmd.Synopsis)
	os.Exit(exitUsage)
}

func globalFlags(fset *flag.FlagSet) {
	fset.StringVar(&catalogPath, "catalog", catalogPath, "path to catalog directory (overrides the "+CatalogPathEnv+" environment variable)")
	fset.StringVar(&host, "host", host, "key for this host (overrides the "+HostEnv+" environment variable)")
}

func fail(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(exitFailure)
}

func failf(f string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, f, args...)
	os.Exit(exitFailure)
}

func parseFlags(fset *flag.FlagSet, args []string) {
	if err := fset.Parse(args); err == flag.ErrHelp {
		os.Exit(exitSuccess)
	} else if err != nil {
		os.Exit(exitUsage)
	}
}

// exit codes
const (
	exitSuccess = 0
	exitFailure = 1
	exitUsage   = 2
)

type optStringFlag struct {
	s       string
	present bool
}

func (f *optStringFlag) String() string {
	if !f.present {
		return `""`
	}
	return `"` + f.s + `"`
}

func (f *optStringFlag) Set(val string) error {
	f.s = val
	f.present = true
	return nil
}

type timeFlag time.Time

func (t *timeFlag) String() string {
	return `"` + (*time.Time)(t).Format(time.RFC3339) + `"`
}

func (t *timeFlag) Set(s string) error {
	tt, err := time.Parse(time.RFC3339, s)
	*t = timeFlag(tt)
	return err
}

type optTimeFlag struct {
	t       timeFlag
	present bool
}

func (f *optTimeFlag) String() string {
	if !f.present || time.Time(f.t).IsZero() {
		return `""`
	}
	return f.t.String()
}

func (f *optTimeFlag) Set(s string) error {
	f.present = true
	return f.t.Set(s)
}

type tagSetFlag catalog.TagSet

func (f *tagSetFlag) String() string {
	return `"` + catalog.TagSet(*f).String() + `"`
}

func (f *tagSetFlag) Set(val string) error {
	*f = tagSetFlag(catalog.ParseTagSet(val))
	return nil
}

type optTagSetFlag struct {
	ts      tagSetFlag
	present bool
}

func (f *optTagSetFlag) String() string {
	if !f.present {
		return `""`
	}
	return f.ts.String()
}

func (f *optTagSetFlag) Set(s string) error {
	f.present = true
	return f.ts.Set(s)
}

var validVCSTypes = []string{
	catalog.CVS,
	catalog.Subversion,
	catalog.Mercurial,
	catalog.Git,
	catalog.Bazaar,
	catalog.Darcs,
}

var validVCSText = strings.Join(validVCSTypes, ", ")

func isValidVCSType(t string) bool {
	for _, v := range validVCSTypes {
		if t == v {
			return true
		}
	}
	return false
}

func sanitizeName(name string) string {
	sn := make([]rune, 0, len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z' || r == '-' || r == '_':
			sn = append(sn, r)
		case r >= 'A' && r <= 'Z':
			sn = append(sn, r-'A'+'a')
		case r == ' ':
			sn = append(sn, '-')
		default:
			sn = append(sn, '_')
		}
	}
	return string(sn)
}
