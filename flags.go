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

func newFlagSet(name string, synopsis string) *flag.FlagSet {
	fset := flag.NewFlagSet(name, flag.ContinueOnError)
	fset.StringVar(&catalogPath, "catalog", catalogPath, "path to catalog directory (overrides the "+CatalogPathEnv+" environment variable)")
	fset.StringVar(&host, "host", host, "key for this host (overrides the "+HostEnv+" environment variable)")
	fset.Usage = func() {
		printUsage(fset, name, synopsis)
	}
	fset.SetOutput(os.Stdout)
	return fset
}

func printUsage(fset *flag.FlagSet, name string, synopsis string) {
	fmt.Printf("Usage of %s:\n", name)
	if synopsis != "" {
		fmt.Printf("  %s\n\n", synopsis)
	}
	fset.PrintDefaults()
}

func exitSynopsis(synopsis string) {
	fmt.Fprintln(os.Stderr, "usage: glados", synopsis)
	os.Exit(exitUsage)
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
	return f.s
}

func (f *optStringFlag) Set(val string) error {
	f.s = val
	f.present = true
	return nil
}

type timeFlag time.Time

func (t *timeFlag) String() string {
	return (*time.Time)(t).Format(time.RFC3339)
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
		return ""
	}
	return f.t.String()
}

func (f *optTimeFlag) Set(s string) error {
	f.present = true
	return f.t.Set(s)
}

type tagsList []string

func (tl tagsList) String() string {
	return strings.Join([]string(tl), ",")
}

func (tl *tagsList) Set(val string) error {
	tags := strings.Split(val, ",")
	for i := range tags {
		tags[i] = strings.TrimSpace(tags[i])
	}
	for i := 0; i < len(tags); {
		if tags[i] == "" {
			tags = append(tags[:i], tags[i+1:]...)
		} else {
			i++
		}
	}
	*tl = tags
	return nil
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
