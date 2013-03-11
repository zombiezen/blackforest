package main

import (
	"path/filepath"
	"time"

	"bitbucket.org/zombiezen/glados/catalog"
	"bitbucket.org/zombiezen/subcmd"
)

func cmdUpdate(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	form := make(map[string][]string)
	fset := cmd.FlagSet(set)
	addFormFlag(fset, form, "name", "human-readable name of project")
	addFormFlag(fset, form, "tags", "set the project's tags, separated by commas. Can't be used with -addtags or -deltags.")
	addFormFlag(fset, form, "addtags", "add tags to the project, separated by commas. Can't be used with -tags.")
	addFormFlag(fset, form, "deltags", "delete tags from the project, separated by commas. Can't be used with -tags.")
	addFormFlag(fset, form, "path", "path of working copy")
	addFormFlag(fset, form, "created", "project creation date, formatted as RFC3339 ("+rfc3339example+")")
	addFormFlag(fset, form, "url", "project homepage")
	addFormFlag(fset, form, "vcs", "type of VCS for project")
	addFormFlag(fset, form, "vcsurl", "project VCS URL")
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
	if err := updateForm(proj, form, host); err != nil {
		return err
	}
	if err := cat.PutProject(proj); err != nil {
		return err
	}
	return nil
}

type projectForm struct {
	Name        string         `schema:"name"`
	ShortName   string         `schema:"shortname"`
	Tags        catalog.TagSet `schema:"tags"`
	AddTags     catalog.TagSet `schema:"addtags"`
	DelTags     catalog.TagSet `schema:"deltags"`
	Description string         `schema:"description"`
	Path        string         `schema:"path"`
	CreateTime  time.Time      `schema:"created"`
	Homepage    string         `schema:"url"`
	VCSType     string         `schema:"vcs"`
	VCSURL      string         `schema:"vcsurl"`
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

func createForm(form map[string][]string, host string) (*catalog.Project, error) {
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

	reqErr := make(formError)
	if isFormValueEmpty(form, "shortname") {
		reqErr["shortname"] = errRequiredField
	}
	if isFormValueEmpty(form, "name") {
		reqErr["name"] = errRequiredField
	}
	if len(reqErr) > 0 {
		return nil, reqErr
	}

	err = updateForm(proj, form, host)
	return proj, err
}

func updateForm(proj *catalog.Project, form map[string][]string, host string) error {
	var f projectForm
	if err := formDecode(&f, form); err != nil {
		return err
	}

	if hasFormField(form, &f, &f.Tags) && (hasFormField(form, &f, &f.AddTags) || hasFormField(form, &f, &f.DelTags)) {
		return formError{formFieldKey(&f, &f.Tags): errTagsMutexFlags}
	}

	ferr := make(formError)
	if f.Name != "" {
		proj.Name = f.Name
	}
	if f.ShortName != "" {
		proj.ShortName = f.ShortName
	}
	if hasFormField(form, &f, &f.Tags) {
		proj.Tags = f.Tags
		proj.Tags.Unique()
	}
	for _, tag := range f.AddTags {
		proj.Tags.Add(tag)
	}
	for _, tag := range f.DelTags {
		proj.Tags.Remove(tag)
	}
	if hasFormField(form, &f, &f.Description) {
		proj.Description = f.Description
	}
	if host != "" && hasFormField(form, &f, &f.Path) {
		if f.Path == "" {
			proj.SetPath(host, "")
		} else {
			if p, err := filepath.Abs(filepath.Clean(f.Path)); err == nil {
				proj.SetPath(host, p)
			} else {
				ferr[formFieldKey(&f, &f.Path)] = err
			}
		}
	}
	if hasFormField(form, &f, &f.CreateTime) {
		proj.CreateTime = f.CreateTime
	}
	if hasFormField(form, &f, &f.Homepage) {
		proj.Homepage = f.Homepage
	}
	if hasFormField(form, &f, &f.VCSType) {
		switch {
		case f.VCSType == "":
			proj.VCS = nil
		case isValidVCSType(f.VCSType):
			if proj.VCS == nil {
				proj.VCS = new(catalog.VCSInfo)
			}
			proj.VCS.Type = f.VCSType
		default:
			ferr[formFieldKey(&f, &f.VCSType)] = badVCSError(f.VCSType)
		}
	}
	if hasFormField(form, &f, &f.VCSURL) {
		if proj.VCS == nil {
			ferr[formFieldKey(&f, &f.VCSURL)] = errDanglingVCSURL
		} else {
			proj.VCS.URL = f.VCSURL
		}
	}
	if len(ferr) > 0 {
		return ferr
	}
	return nil
}
