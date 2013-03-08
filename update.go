package main

import (
	"flag"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	"bitbucket.org/zombiezen/glados/catalog"
	"bitbucket.org/zombiezen/subcmd"
)

func cmdUpdate(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	var (
		name     string
		tags     catalog.TagSet
		addTags  catalog.TagSet
		delTags  catalog.TagSet
		path     string
		created  time.Time
		homepage string
		vcsType  string
		vcsURL   string
	)
	flagDesc := []struct {
		Name    string
		AddFlag func(*flag.FlagSet, string)
		Do      func(*catalog.Project) error
	}{
		{
			"name",
			stringUpdateFlag(&name, "human-readable name of project"),
			func(proj *catalog.Project) error {
				proj.Name = name
				return nil
			},
		},
		{
			"tags",
			tagSetUpdateFlag(&tags, "set the project's tags, separated by commas. Can't be used with -addtags or -deltags."),
			func(proj *catalog.Project) error {
				proj.Tags = tags
				proj.Tags.Unique()
				return nil
			},
		},
		{
			"addtags",
			tagSetUpdateFlag(&addTags, "add tags to the project, separated by commas. Can't be used with -tags."),
			func(proj *catalog.Project) error {
				for _, tag := range addTags {
					proj.Tags.Add(tag)
				}
				return nil
			},
		},
		{
			"deltags",
			tagSetUpdateFlag(&delTags, "delete tags from the project, separated by commas. Can't be used with -tags."),
			func(proj *catalog.Project) error {
				for _, tag := range delTags {
					proj.Tags.Remove(tag)
				}
				return nil
			},
		},
		{
			"path",
			stringUpdateFlag(&path, "path of working copy"),
			func(proj *catalog.Project) error {
				if host == "" {
					return errHostNotSetPathGiven
				}
				if path != "" {
					var err error
					if path, err = filepath.Abs(filepath.Clean(path)); err != nil {
						return err
					}
				}
				proj.SetPath(host, path)
				return nil
			},
		},
		{
			"created",
			timeUpdateFlag(&created, "project creation date, formatted as RFC3339 ("+rfc3339example+")"),
			func(proj *catalog.Project) error {
				proj.CreateTime = created
				return nil
			},
		},
		{
			"url",
			stringUpdateFlag(&homepage, "project homepage"),
			func(proj *catalog.Project) error {
				proj.Homepage = homepage
				return nil
			},
		},
		{
			"vcs",
			stringUpdateFlag(&vcsType, "type of VCS for project"),
			func(proj *catalog.Project) error {
				switch {
				case vcsType == "":
					proj.VCS = nil
				case isValidVCSType(vcsType):
					if proj.VCS == nil {
						proj.VCS = new(catalog.VCSInfo)
					}
					proj.VCS.Type = vcsType
				default:
					return badVCSError(vcsType)
				}
				return nil
			},
		},
		{
			"vcsurl",
			stringUpdateFlag(&vcsURL, "project VCS URL"),
			func(proj *catalog.Project) error {
				if proj.VCS == nil {
					// This check happens after the -vcs flag processing, so
					// it's okay to check for nil.
					return errDanglingVCSURL
				}
				proj.VCS.URL = vcsURL
				return nil
			},
		},
	}

	fset := cmd.FlagSet(set)
	for i := range flagDesc {
		fd := &flagDesc[i]
		fd.AddFlag(fset, fd.Name)
	}
	parseFlags(fset, args)
	if fset.NArg() != 1 {
		cmd.PrintSynopsis(set)
		return exitError(exitUsage)
	}
	flags := make(map[string]*flag.Flag)
	fset.Visit(func(f *flag.Flag) {
		flags[f.Name] = f
	})
	if flags["tags"] != nil && (flags["addtags"] != nil || flags["deltags"] != nil) {
		return errTagsMutexFlags
	}
	cat := requireCatalog()

	shortName := fset.Arg(0)
	proj, err := cat.GetProject(shortName)
	if err != nil {
		return err
	}

	for i := range flagDesc {
		fd := &flagDesc[i]
		if flags[fd.Name] != nil {
			if err := fd.Do(proj); err != nil {
				return err
			}
		}
	}

	if err := cat.PutProject(proj); err != nil {
		return err
	}
	return nil
}

func stringUpdateFlag(s *string, help string) func(*flag.FlagSet, string) {
	return func(f *flag.FlagSet, n string) {
		f.StringVar(s, n, *s, help)
	}
}

func tagSetUpdateFlag(ts *catalog.TagSet, help string) func(*flag.FlagSet, string) {
	return func(f *flag.FlagSet, n string) {
		f.Var((*tagSetFlag)(ts), n, help)
	}
}

func timeUpdateFlag(t *time.Time, help string) func(*flag.FlagSet, string) {
	return func(f *flag.FlagSet, n string) {
		f.Var((*timeFlag)(t), n, help)
	}
}

const (
	projectShortNameParam   = "shortname"
	projectNameParam        = "name"
	projectTagsParam        = "tags"
	projectDescriptionParam = "description"
	projectCreateTimeParam  = "created"
	projectHomepageParam    = "url"
	projectVCSTypeParam     = "vcs"
	projectVCSURLParam      = "vcsurl"
)

func updateForm(proj *catalog.Project, form url.Values) error {
	ferr := make(formError)
	if v := form.Get(projectShortNameParam); v != "" {
		proj.ShortName = v
	}
	if v := form.Get(projectNameParam); v != "" {
		proj.Name = v
	}
	if v := form[projectTagsParam]; len(v) > 0 {
		proj.Tags = catalog.ParseTagSet(v[0])
	}
	if v := form[projectDescriptionParam]; len(v) > 0 {
		proj.Description = v[0]
	}
	if v := form.Get(projectCreateTimeParam); v != "" {
		// TODO
	}
	if v := form[projectHomepageParam]; len(v) > 0 {
		proj.Homepage = v[0]
	}
	if v := form[projectVCSTypeParam]; len(v) > 0 {
		v := v[0]
		switch {
		case v == "":
			proj.VCS = nil
		case isValidVCSType(v):
			if proj.VCS == nil {
				proj.VCS = new(catalog.VCSInfo)
			}
			proj.VCS.Type = v
		default:
			ferr[projectVCSTypeParam] = badVCSError(v)
		}
	}
	if v := form[projectVCSURLParam]; len(v) > 0 {
		v := v[0]
		if proj.VCS == nil {
			ferr[projectVCSURLParam] = errDanglingVCSURL
		} else {
			proj.VCS.URL = v
		}
	}
	if len(ferr) > 0 {
		return ferr
	}
	return nil
}

type formError map[string]error

func (e formError) Error() string {
	msg, n := "", 0
	for _, err := range e {
		if err != nil {
			if n == 0 {
				msg = err.Error()
			}
			n++
		}
	}
	switch n {
	case 0:
		return "0 errors"
	case 1:
		return msg
	case 2:
		return msg + " (and 1 other error)"
	}
	return fmt.Sprintf("%s (and %d other errors)", msg, n-1)
}
