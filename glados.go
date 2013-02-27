// The glados command provides a front-end for a GLaDOS catalog.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"bitbucket.org/zombiezen/glados/catalog"
	"bitbucket.org/zombiezen/glados/vcs"
	"bitbucket.org/zombiezen/subcmd"
)

func main() {
	if err := commandSet.Do(os.Args[1:]); err == nil {
		os.Exit(exitSuccess)
	} else if err == flag.ErrHelp {
		os.Exit(exitUsage)
	} else if _, ok := err.(subcmd.CommandError); ok {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitUsage)
	} else {
		fail(err)
	}
}

var commandSet = subcmd.Set{
	Name:        "glados",
	GlobalFlags: globalFlags,
	Commands: []subcmd.Command{
		{
			Func:        cmdInit,
			Name:        "init",
			Aliases:     []string{},
			Synopsis:    "init -catalog=PATH",
			Description: "create a catalog",
		},
		{
			Func:        cmdList,
			Name:        "list",
			Aliases:     []string{"ls"},
			Synopsis:    "list",
			Description: "list project short names",
		},
		{
			Func:        cmdPath,
			Name:        "path",
			Aliases:     []string{},
			Synopsis:    "path PROJECT",
			Description: "print a project's local path",
		},
		{
			Func:        cmdShow,
			Name:        "show",
			Aliases:     []string{"info"},
			Synopsis:    "show PROJECT [...]",
			Description: "print projects",
		},
		{
			Func:        cmdCreate,
			Name:        "create",
			Aliases:     []string{},
			Synopsis:    "create [options] NAME",
			Description: "create a project",
		},
		{
			Func:        cmdUpdate,
			Name:        "update",
			Aliases:     []string{"up"},
			Synopsis:    "update [options] PROJECT",
			Description: "change project fields",
		},
		{
			Func:        cmdRename,
			Name:        "rename",
			Aliases:     []string{"mv"},
			Synopsis:    "rename SRC DST",
			Description: "change a project's short name",
		},
		{
			Func:        cmdDelete,
			Name:        "delete",
			Aliases:     []string{"del", "rm"},
			Synopsis:    "delete PROJECT [...]",
			Description: "delete projects",
		},
		{
			Func:        cmdImport,
			Name:        "import",
			Aliases:     []string{},
			Synopsis:    "import [PATH [...]]",
			Description: "import project(s) from JSON",
		},
		{
			Func:        cmdCheckout,
			Name:        "checkout",
			Aliases:     []string{"co"},
			Synopsis:    "checkout PROJECT [PATH]",
			Description: "check out project from version control",
		},
	},
}

func cmdInit(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	fset := cmd.FlagSet(set)
	parseFlags(fset, args)
	if fset.NArg() != 0 {
		exitSynopsis(set, cmd)
	}

	if catalogPath == "" {
		fail(CatalogPathEnv + " not set")
	}
	if _, err := catalog.Create(catalogPath); err != nil {
		fail(err)
	}
	return nil
}

func cmdList(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	fset := cmd.FlagSet(set)
	parseFlags(fset, args)
	cat := requireCatalog()
	if fset.NArg() != 0 {
		exitSynopsis(set, cmd)
	}

	list, err := cat.List()
	if err != nil {
		fail(err)
	}
	sort.Strings(list)
	for _, name := range list {
		fmt.Println(name)
	}
	return nil
}

func cmdPath(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	fset := cmd.FlagSet(set)
	parseFlags(fset, args)
	if fset.NArg() != 1 {
		exitSynopsis(set, cmd)
	}
	cat := requireCatalog()

	if host == "" {
		fail(HostEnv + " not set")
	}
	proj, err := cat.GetProject(fset.Arg(0))
	if err != nil {
		fail(err)
	}
	p := proj.Path(host)
	if p == "" {
		os.Exit(exitFailure)
	}
	fmt.Println(p)
	return nil
}

func cmdShow(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	fset := cmd.FlagSet(set)
	jsonFormat := fset.Bool("json", false, "print project as JSON")
	rfc3339Time := fset.Bool("rfc3339", false, "print dates as RFC3339")
	parseFlags(fset, args)
	if fset.NArg() == 0 {
		exitSynopsis(set, cmd)
	}
	cat := requireCatalog()

	if *jsonFormat {
		projects := make([]*catalog.Project, 0, fset.NArg())
		for _, shortName := range fset.Args() {
			proj, err := cat.GetProject(shortName)
			if err != nil {
				fail(err)
			}
			projects = append(projects, proj)
		}
		if err := json.NewEncoder(os.Stdout).Encode(projects); err != nil {
			fail(err)
		}
	} else {
		fmtTime := fmtSimpleTime
		if *rfc3339Time {
			fmtTime = fmtRFC3339Time
		}
		failed := false
		for i, shortName := range fset.Args() {
			if i > 0 {
				fmt.Println()
			}
			if proj, err := cat.GetProject(shortName); err == nil {
				showProject(proj, fmtTime)
			} else {
				fmt.Fprintln(os.Stderr, err)
				failed = true
			}
		}
		if failed {
			os.Exit(exitFailure)
		}
	}
	return nil
}

func showProject(proj *catalog.Project, fmtTime func(time.Time) string) {
	fmt.Println(proj.Name)
	showField("ID", proj.ID)
	if p := proj.Path(host); p != "" {
		showField("Path", p)
	}
	if len(proj.Tags) != 0 {
		sort.Strings(proj.Tags)
		showField("Tags", strings.Join(proj.Tags, ", "))
	}
	showField("Created", fmtTime(proj.CreateTime))
	showField("Added On", fmtTime(proj.CatalogTime))
	if proj.Homepage != "" {
		showField("URL", proj.Homepage)
	}
	if vcsInfo := proj.VCS; vcsInfo != nil {
		if vcsInfo.URL != "" {
			showField("VCS", vcsInfo.Type, vcsInfo.URL)
		} else {
			showField("VCS", vcsInfo.Type)
		}
	}
	if proj.Description != "" {
		fmt.Println("\n" + proj.Description)
	}
}

func showField(label string, args ...interface{}) {
	fmt.Printf("%-9s %s", label+":", fmt.Sprintln(args...))
}

func fmtSimpleTime(t time.Time) string {
	const layout = "Jan 2 2006 3:04PM"
	return t.Local().Format(layout)
}

func fmtRFC3339Time(t time.Time) string {
	return t.Format(time.RFC3339)
}

func cmdImport(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	fset := cmd.FlagSet(set)
	parseFlags(fset, args)
	cat := requireCatalog()

	if fset.NArg() == 0 {
		if err := importProject(cat, os.Stdin); err != nil {
			fail(err)
		}
	} else {
		failed := false
		for _, path := range fset.Args() {
			f, err := os.Open(path)
			if err == nil {
				err := importProject(cat, f)
				f.Close()
				if err != nil {
					failed = true
					fmt.Fprintln(os.Stderr, err)
				}
			} else {
				fmt.Fprintln(os.Stderr, err)
				failed = true
			}
		}
		if failed {
			os.Exit(exitFailure)
		}
	}
	return nil
}

func importProject(cat catalog.Catalog, r io.Reader) error {
	var proj catalog.Project
	if err := json.NewDecoder(r).Decode(&proj); err != nil {
		return err
	}
	return cat.PutProject(&proj)
}

const rfc3339example = "2006-01-02T15:04:05-07:00"

func cmdCreate(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	now := time.Now()
	proj := &catalog.Project{
		VCS:         new(catalog.VCSInfo),
		CatalogTime: now,
		CreateTime:  now,
	}
	var hostInfo catalog.HostInfo

	fset := cmd.FlagSet(set)
	fset.StringVar(&proj.ShortName, "shortname", "", "identifier for project (default is lowercased full name)")
	fset.Var((*tagSetFlag)(&proj.Tags), "tags", "comma-separated tags to assign to the new project")
	fset.StringVar(&hostInfo.Path, "path", "", "path of working copy")
	fset.StringVar(&proj.Homepage, "url", "", "project homepage")
	fset.StringVar(&proj.VCS.Type, "vcs", "", "type of VCS for project")
	fset.StringVar(&proj.VCS.URL, "vcsurl", "", "project VCS URL")
	fset.Var((*timeFlag)(&proj.CreateTime), "created", "project creation date, formatted as RFC3339 ("+rfc3339example+")")
	parseFlags(fset, args)
	if fset.NArg() != 1 {
		exitSynopsis(set, cmd)
	}
	cat := requireCatalog()

	name := strings.TrimSpace(fset.Arg(0))
	if len(name) == 0 {
		fail("empty name")
	}
	proj.Name = name
	proj.Tags.Unique()
	id, err := catalog.GenerateID()
	if err != nil {
		fail(err)
	}
	proj.ID = id
	if proj.ShortName == "" {
		proj.ShortName = sanitizeName(name)
	}
	if proj.VCS.Type == "" {
		proj.VCS = nil
	} else if !isValidVCSType(proj.VCS.Type) {
		failf("%q is not a valid -vcs\nvalid choices are: %s\n", proj.VCS.Type, validVCSText)
	}
	if hostInfo.Path != "" {
		absPath, err := filepath.Abs(filepath.Clean(hostInfo.Path))
		if err != nil {
			fail(err)
		}
		hostInfo.Path = absPath
		if host == "" {
			fail("-path given and " + HostEnv + " not set")
		}
		proj.PerHost = map[string]*catalog.HostInfo{host: &hostInfo}
	}
	if err := cat.PutProject(proj); err != nil {
		fail(err)
	}
	return nil
}

func cmdUpdate(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	var (
		name        optStringFlag
		tagsFlag    optTagSetFlag
		addTagsFlag optTagSetFlag
		delTagsFlag optTagSetFlag
		path        optStringFlag
		created     optTimeFlag
		homepage    optStringFlag
		vcsType     optStringFlag
		vcsURL      optStringFlag
	)

	fset := cmd.FlagSet(set)
	fset.Var(&name, "name", "human-readable name of project")
	fset.Var(&tagsFlag, "tags", "set the project's tags, separated by commas. Can't be used with -addtags or -deltags.")
	fset.Var(&addTagsFlag, "addtags", "add tags to the project, separated by commas. Can't be used with -tags.")
	fset.Var(&delTagsFlag, "deltags", "delete tags from the project, separated by commas. Can't be used with -tags.")
	fset.Var(&path, "path", "path of working copy")
	fset.Var(&created, "created", "project creation date, formatted as RFC3339 ("+rfc3339example+")")
	fset.Var(&homepage, "url", "project homepage")
	fset.Var(&vcsType, "vcs", "type of VCS for project")
	fset.Var(&vcsURL, "vcsurl", "project VCS URL")
	parseFlags(fset, args)
	if fset.NArg() != 1 {
		exitSynopsis(set, cmd)
	}
	if tagsFlag.present && (addTagsFlag.present || delTagsFlag.present) {
		// -tags and -addtags/-deltags are mutally exclusive
		fmt.Fprintln(os.Stderr, "cannot use -tags flag with -addtags or -deltags")
		os.Exit(exitUsage)
	}
	cat := requireCatalog()

	shortName := fset.Arg(0)
	proj, err := cat.GetProject(shortName)
	if err != nil {
		fail(err)
	}

	updateString(&proj.Name, &name)
	updateString(&proj.Homepage, &homepage)
	if created.present {
		proj.CreateTime = time.Time(created.t)
	}
	if vcsType.present {
		vt := vcsType.s
		switch {
		case vt == "":
			proj.VCS = nil
		case isValidVCSType(vt):
			if proj.VCS == nil {
				proj.VCS = new(catalog.VCSInfo)
			}
			proj.VCS.Type = vt
		default:
			failf("%q is not a valid -vcs\nvalid choices are: %s\n", vt, validVCSText)
		}
	}
	if vcsURL.present {
		if proj.VCS == nil {
			fail("-vcsurl given, but project has no VCS")
		}
		proj.VCS.URL = vcsURL.s
	}
	if path.present {
		if host == "" {
			fail("-path given, but " + HostEnv + " not set")
		}
		if path.s != "" {
			var err error
			if path.s, err = filepath.Abs(filepath.Clean(path.s)); err != nil {
				fail(err)
			}
		}
		proj.SetPath(host, path.s)
	}

	if tagsFlag.present {
		proj.Tags = catalog.TagSet(tagsFlag.ts)
		proj.Tags.Unique()
	} else {
		if addTagsFlag.present {
			for _, tag := range addTagsFlag.ts {
				proj.Tags.Add(tag)
			}
		}
		if delTagsFlag.present {
			for _, tag := range delTagsFlag.ts {
				proj.Tags.Remove(tag)
			}
		}
	}

	if err := cat.PutProject(proj); err != nil {
		fail(err)
	}
	return nil
}

func updateString(s *string, f *optStringFlag) {
	if f.present {
		*s = f.s
	}
}

func cmdRename(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	fset := cmd.FlagSet(set)
	parseFlags(fset, args)
	if fset.NArg() != 2 {
		exitSynopsis(set, cmd)
	}
	cat := requireCatalog()

	src, dst := fset.Arg(0), fset.Arg(1)
	proj, err := cat.GetProject(src)
	if err != nil {
		fail(err)
	}
	proj.ShortName = dst
	if err := cat.PutProject(proj); err != nil {
		fail(err)
	}
	return nil
}

func cmdDelete(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	fset := cmd.FlagSet(set)
	parseFlags(fset, args)
	if fset.NArg() == 0 {
		exitSynopsis(set, cmd)
	}
	cat := requireCatalog()

	failed := false
	for _, name := range fset.Args() {
		if err := cat.DelProject(name); err != nil {
			failed = true
			fmt.Fprintln(os.Stderr, err)
		}
	}
	if failed {
		os.Exit(exitFailure)
	}
	return nil
}

func cmdCheckout(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	fset := cmd.FlagSet(set)
	setPath := fset.Bool("setpath", true, "update the project's path to the new checkout")
	overwritePath := fset.Bool("overwritepath", false, "change the project's path, even if there already is one")
	parseFlags(fset, args)
	if n := fset.NArg(); n == 0 || n > 2 {
		exitSynopsis(set, cmd)
	}
	shortName := fset.Arg(0)
	path := shortName
	if fset.NArg() == 2 {
		path = fset.Arg(1)
	}
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		fail(err)
	}
	if *setPath && host == "" {
		fail(HostEnv + " not set")
	}
	cat := requireCatalog()

	proj, err := cat.GetProject(shortName)
	if err != nil {
		fail(err)
	}
	if proj.VCS == nil || proj.VCS.URL == "" {
		failf("project %s has no VCS URL", shortName)
	}
	if p := proj.Path(host); *setPath && p != "" && !*overwritePath {
		failf("project already has path: %s\n(use -overwritepath to force)\n", p)
	}

	var vc vcs.VCS
	switch proj.VCS.Type {
	case catalog.Mercurial:
		vc = new(vcs.Mercurial)
	default:
		fail(errUnhandledVCS)
	}
	if _, err := vc.Checkout(proj.VCS.URL, absPath); err != nil {
		fail(err)
	}
	if *setPath {
		proj.SetPath(host, absPath)
		if err := cat.PutProject(proj); err != nil {
			fail(err)
		}
	}
	return nil
}

func requireCatalog() catalog.Catalog {
	if catalogPath == "" {
		fail(CatalogPathEnv + " not set")
	}

	// TODO(light): check for other VCSs
	var v vcs.VCS = new(vcs.Mercurial)
	if ok, err := v.IsWorkingCopy(catalogPath); !ok || err != nil {
		if err != nil {
			fmt.Fprintln(os.Stderr, "catalog VCS warning:", err)
		}
		v = nil
	}

	cat, err := catalog.Open(catalogPath, v)
	if err != nil {
		fail(err)
	}
	return cat
}

var errUnhandledVCS = errors.New("unhandled VCS")
