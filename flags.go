package main

import (
	"flag"
	"os"
	"strings"
	"time"

	"bitbucket.org/zombiezen/glados/catalog"
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

type timeFlag time.Time

func (t *timeFlag) String() string {
	if (*time.Time)(t).IsZero() {
		return `""`
	}
	return `"` + (*time.Time)(t).Format(time.RFC3339) + `"`
}

func (t *timeFlag) Set(s string) error {
	tt, err := time.Parse(time.RFC3339, s)
	*t = timeFlag(tt)
	return err
}

type tagSetFlag catalog.TagSet

func (f *tagSetFlag) String() string {
	return `"` + catalog.TagSet(*f).String() + `"`
}

func (f *tagSetFlag) Set(val string) error {
	*f = tagSetFlag(catalog.ParseTagSet(val))
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
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_':
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
