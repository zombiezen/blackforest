package main

import (
	"path/filepath"
	"strings"
	"time"

	"bitbucket.org/zombiezen/glados/catalog"
	"bitbucket.org/zombiezen/subcmd"
	"github.com/zombiezen/schema"
)

const rfc3339example = "2006-01-02T15:04:05-07:00"

func cmdCreate(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	form := make(map[string][]string)
	fset := cmd.FlagSet(set)
	addFormFlag(fset, form, projectFormShortNameKey, "identifier for project (default is lowercased full name)")
	addFormFlag(fset, form, projectFormTagsKey, "comma-separated tags to assign to the new project")
	addFormFlag(fset, form, projectFormPathKey, "path of working copy")
	addFormFlag(fset, form, projectFormCreateTimeKey, "project creation date, formatted as RFC3339 ("+rfc3339example+")")
	addFormFlag(fset, form, projectFormHomepageKey, "project homepage")
	addFormFlag(fset, form, projectFormVCSTypeKey, "type of VCS for project")
	addFormFlag(fset, form, projectFormVCSURLKey, "project VCS URL")
	parseFlags(fset, args)
	if fset.NArg() != 1 {
		cmd.PrintSynopsis(set)
		return exitError(exitUsage)
	}
	cat := requireCatalog()

	name := strings.TrimSpace(fset.Arg(0))
	if len(name) == 0 {
		return errEmptyName
	}
	form[projectFormNameKey] = []string{name}
	if isFormValueEmpty(form, projectFormShortNameKey) {
		form[projectFormShortNameKey] = []string{sanitizeName(name)}
	}

	proj, err := createProjectForm(form, host)
	if err != nil {
		return err
	}
	return cat.PutProject(proj)
}

func cmdUpdate(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	form := make(map[string][]string)
	fset := cmd.FlagSet(set)
	addFormFlag(fset, form, projectFormNameKey, "human-readable name of project")
	addFormFlag(fset, form, projectFormTagsKey, "set the project's tags, separated by commas. Can't be used with -addtags or -deltags.")
	addFormFlag(fset, form, projectFormAddTagsKey, "add tags to the project, separated by commas. Can't be used with -tags.")
	addFormFlag(fset, form, projectFormDelTagsKey, "delete tags from the project, separated by commas. Can't be used with -tags.")
	addFormFlag(fset, form, projectFormPathKey, "path of working copy")
	addFormFlag(fset, form, projectFormCreateTimeKey, "project creation date, formatted as RFC3339 ("+rfc3339example+")")
	addFormFlag(fset, form, projectFormHomepageKey, "project homepage")
	addFormFlag(fset, form, projectFormVCSTypeKey, "type of VCS for project")
	addFormFlag(fset, form, projectFormVCSURLKey, "project VCS URL")
	parseFlags(fset, args)
	if fset.NArg() != 1 {
		cmd.PrintSynopsis(set)
		return exitError(exitUsage)
	}
	cat := requireCatalog()

	shortName := fset.Arg(0)
	proj, err := cat.GetProject(shortName)
	if err != nil {
		return err
	}
	if err := updateProjectForm(proj, form, host); err != nil {
		return err
	}
	if err := cat.PutProject(proj); err != nil {
		return err
	}
	return nil
}

const (
	projectFormNameKey        = "name"
	projectFormShortNameKey   = "shortname"
	projectFormTagsKey        = "tags"
	projectFormAddTagsKey     = "addtags"
	projectFormDelTagsKey     = "deltags"
	projectFormDescriptionKey = "description"
	projectFormPathKey        = "path"
	projectFormCreateTimeKey  = "created"
	projectFormHomepageKey    = "url"
	projectFormVCSTypeKey     = "vcs"
	projectFormVCSURLKey      = "vcsurl"
)

type projectForm struct {
	Name        string         `schema:"name"`
	ShortName   string         `schema:"shortname"`
	Tags        catalog.TagSet `schema:"tags"`
	AddTags     catalog.TagSet `schema:"addtags"`
	DelTags     catalog.TagSet `schema:"deltags"`
	Description nullString     `schema:"description"`
	Path        nullString     `schema:"path"`
	CreateTime  *time.Time     `schema:"created"`
	Homepage    nullString     `schema:"url"`
	VCSType     nullString     `schema:"vcs"`
	VCSURL      nullString     `schema:"vcsurl"`
}

func (f *projectForm) Update(proj *catalog.Project, host string) error {
	if f.Tags != nil && (f.AddTags != nil || f.DelTags != nil) {
		return schema.MultiError{projectFormTagsKey: errTagsMutexFlags}
	}

	ferr := make(schema.MultiError)
	if f.Name != "" {
		proj.Name = f.Name
	}
	if f.ShortName != "" {
		proj.ShortName = f.ShortName
	}
	if f.Tags != nil {
		proj.Tags = f.Tags
		proj.Tags.Unique()
	}
	for _, tag := range f.AddTags {
		proj.Tags.Add(tag)
	}
	for _, tag := range f.DelTags {
		proj.Tags.Remove(tag)
	}
	if f.Description.Valid {
		proj.Description = f.Description.String
	}
	if f.Path.Valid {
		path := f.Path.String
		if host == "" {
			ferr[projectFormPathKey] = errHostNotSetPathGiven
		} else if path == "" {
			proj.SetPath(host, "")
		} else if p, err := filepath.Abs(filepath.Clean(path)); err == nil {
			proj.SetPath(host, p)
		} else {
			ferr[projectFormPathKey] = err
		}
	}
	if f.CreateTime != nil {
		proj.CreateTime = *f.CreateTime
	}
	if f.Homepage.Valid {
		proj.Homepage = f.Homepage.String
	}
	if f.VCSType.Valid {
		vt := f.VCSType.String
		switch {
		case vt == "":
			proj.VCS = nil
		case isValidVCSType(vt):
			if proj.VCS == nil {
				proj.VCS = new(catalog.VCSInfo)
			}
			proj.VCS.Type = vt
		default:
			ferr[projectFormVCSTypeKey] = badVCSError(vt)
		}
	}
	if f.VCSURL.Valid {
		if proj.VCS == nil {
			ferr[projectFormVCSURLKey] = errDanglingVCSURL
		} else {
			proj.VCS.URL = f.VCSURL.String
		}
	}
	if len(ferr) > 0 {
		return ferr
	}
	return nil
}

func createProjectForm(form map[string][]string, host string) (*catalog.Project, error) {
	now := time.Now()
	id, err := catalog.GenerateID()
	if err != nil {
		return nil, err
	}
	proj := &catalog.Project{
		ID:          id,
		CreateTime:  now,
		CatalogTime: now,
	}

	reqErr := make(schema.MultiError)
	if isFormValueEmpty(form, projectFormShortNameKey) {
		reqErr[projectFormShortNameKey] = errRequiredField
	}
	if isFormValueEmpty(form, projectFormNameKey) {
		reqErr[projectFormNameKey] = errRequiredField
	}
	if len(reqErr) > 0 {
		return nil, reqErr
	}

	delete(form, projectFormAddTagsKey)
	delete(form, projectFormDelTagsKey)
	err = updateProjectForm(proj, form, host)
	return proj, err
}

func updateProjectForm(proj *catalog.Project, form map[string][]string, host string) error {
	var f projectForm
	if err := decoder.Decode(&f, form); err != nil {
		return err
	}
	return f.Update(proj, host)
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
