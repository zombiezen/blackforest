package main

import (
	"encoding/json"
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

func init() {
	for i := range commandSet.Commands {
		c := &commandSet.Commands[i]
		c.Func = catchCmdPanics(c.Func)
	}
}

func catchCmdPanics(f subcmd.Func) subcmd.Func {
	return func(set *subcmd.Set, cmd *subcmd.Command, args []string) (err error) {
		defer func() {
			r := recover()
			if e, ok := r.(error); ok {
				err = e
			} else if r != nil {
				panic(r)
			}
		}()
		err = f(set, cmd, args)
		return
	}
}

func cmdInit(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	fset := cmd.FlagSet(set)
	parseFlags(fset, args)
	if fset.NArg() != 0 {
		cmd.PrintSynopsis(set)
		return exitError(exitUsage)
	}

	if catalogPath == "" {
		return errCatalogPathNotSet
	}
	if _, err := catalog.Create(catalogPath); err != nil {
		return err
	}
	return nil
}

func cmdList(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	fset := cmd.FlagSet(set)
	parseFlags(fset, args)
	cat := requireCatalog()
	if fset.NArg() != 0 {
		cmd.PrintSynopsis(set)
		return exitError(exitUsage)
	}

	list, err := cat.List()
	if err != nil {
		return err
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
		cmd.PrintSynopsis(set)
		return exitError(exitUsage)
	}
	cat := requireCatalog()

	if host == "" {
		return errHostNotSet
	}
	proj, err := cat.GetProject(fset.Arg(0))
	if err != nil {
		return err
	}
	p := proj.Path(host)
	if p == "" {
		return errFailed
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
		cmd.PrintSynopsis(set)
		return exitError(exitUsage)
	}
	cat := requireCatalog()

	if *jsonFormat {
		projects := make([]*catalog.Project, 0, fset.NArg())
		for _, shortName := range fset.Args() {
			proj, err := cat.GetProject(shortName)
			if err != nil {
				return err
			}
			projects = append(projects, proj)
		}
		if err := json.NewEncoder(os.Stdout).Encode(projects); err != nil {
			return err
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
			return errFailed
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
			return err
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
			return errFailed
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
		cmd.PrintSynopsis(set)
		return exitError(exitUsage)
	}
	cat := requireCatalog()

	name := strings.TrimSpace(fset.Arg(0))
	if len(name) == 0 {
		return errEmptyName
	}
	proj.Name = name
	proj.Tags.Unique()
	id, err := catalog.GenerateID()
	if err != nil {
		return err
	}
	proj.ID = id
	if proj.ShortName == "" {
		proj.ShortName = sanitizeName(name)
	}
	if proj.VCS.Type == "" {
		proj.VCS = nil
	} else if !isValidVCSType(proj.VCS.Type) {
		return badVCSError(proj.VCS.Type)
	}
	if hostInfo.Path != "" {
		absPath, err := filepath.Abs(filepath.Clean(hostInfo.Path))
		if err != nil {
			return err
		}
		hostInfo.Path = absPath
		if host == "" {
			return errHostNotSetPathGiven
		}
		proj.PerHost = map[string]*catalog.HostInfo{host: &hostInfo}
	}
	if err := cat.PutProject(proj); err != nil {
		return err
	}
	return nil
}

func cmdRename(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	fset := cmd.FlagSet(set)
	parseFlags(fset, args)
	if fset.NArg() != 2 {
		cmd.PrintSynopsis(set)
		return exitError(exitUsage)
	}
	cat := requireCatalog()

	src, dst := fset.Arg(0), fset.Arg(1)
	proj, err := cat.GetProject(src)
	if err != nil {
		return err
	}
	proj.ShortName = dst
	if err := cat.PutProject(proj); err != nil {
		return err
	}
	return nil
}

func cmdDelete(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	fset := cmd.FlagSet(set)
	parseFlags(fset, args)
	if fset.NArg() == 0 {
		cmd.PrintSynopsis(set)
		return exitError(exitUsage)
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
		return errFailed
	}
	return nil
}

func cmdCheckout(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	fset := cmd.FlagSet(set)
	setPath := fset.Bool("setpath", true, "update the project's path to the new checkout")
	overwritePath := fset.Bool("overwritepath", false, "change the project's path, even if there already is one")
	parseFlags(fset, args)
	if n := fset.NArg(); n == 0 || n > 2 {
		cmd.PrintSynopsis(set)
		return exitError(exitUsage)
	}
	shortName := fset.Arg(0)
	path := shortName
	if fset.NArg() == 2 {
		path = fset.Arg(1)
	}
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return err
	}
	if *setPath && host == "" {
		return errHostNotSet
	}
	cat := requireCatalog()

	proj, err := cat.GetProject(shortName)
	if err != nil {
		return err
	}
	if proj.VCS == nil || proj.VCS.URL == "" {
		return noVCSURLError(shortName)
	}
	if p := proj.Path(host); *setPath && p != "" && !*overwritePath {
		return &projectHasPathError{ShortName: proj.ShortName, Path: p}
	}

	var vc vcs.VCS
	switch vt := proj.VCS.Type; vt {
	case catalog.Mercurial:
		vc = new(vcs.Mercurial)
	default:
		return badVCSError(vt)
	}
	if _, err := vc.Checkout(proj.VCS.URL, absPath); err != nil {
		return err
	}
	if *setPath {
		proj.SetPath(host, absPath)
		if err := cat.PutProject(proj); err != nil {
			return err
		}
	}
	return nil
}