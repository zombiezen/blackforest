// The glados command provides a front-end for a GLaDOS catalog.
package main

import (
	"bitbucket.org/zombiezen/glados/catalog"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {
	fset := newFlagSet("glados", "glados [options] COMMAND ...")
	parseFlags(fset, os.Args[1:])
	if fset.NArg() == 0 {
		fset.Usage()
		os.Exit(exitUsage)
	}
	cname := fset.Arg(0)
	if c := commands[cname]; c != nil {
		c(fset.Args()[1:])
		os.Exit(exitSuccess)
	} else {
		fmt.Fprintln(os.Stderr, "unrecognized command:", cname)
		os.Exit(exitUsage)
	}
}

var commands = map[string]func([]string){
	"create": cmdCreate,
	"del":    cmdDelete,
	"delete": cmdDelete,
	"init":   cmdInit,
	"list":   cmdList,
	"ls":     cmdList,
	"mv":     cmdRename,
	"rename": cmdRename,
	"rm":     cmdDelete,
	"show":   cmdShow,
	"up":     cmdUpdate,
	"update": cmdUpdate,
}

func cmdInit(args []string) {
	const synopsis = "init -catalog=PATH"

	fset := newFlagSet("init", synopsis)
	parseFlags(fset, args)
	if fset.NArg() != 0 {
		exitSynopsis(synopsis)
	}

	if catalogPath == "" {
		fail(CatalogPathEnv + " not set")
	}
	if _, err := catalog.Create(catalogPath); err != nil {
		fail(err)
	}
}

func cmdList(args []string) {
	const synopsis = "list"

	fset := newFlagSet("list", synopsis)
	parseFlags(fset, args)
	cat := requireCatalog()
	if fset.NArg() != 0 {
		exitSynopsis(synopsis)
	}

	list, err := cat.List()
	if err != nil {
		fail(err)
	}
	for _, name := range list {
		fmt.Println(name)
	}
}

func cmdShow(args []string) {
	const synopsis = "show PROJECT"

	fset := newFlagSet("show", synopsis)
	jsonFormat := fset.Bool("json", false, "print project as JSON")
	rfc3339Time := fset.Bool("rfc3339", false, "print dates as RFC3339")
	parseFlags(fset, args)
	if fset.NArg() != 1 {
		exitSynopsis(synopsis)
	}
	cat := requireCatalog()

	proj, err := cat.GetProject(fset.Arg(0))
	if err != nil {
		fail(err)
	}
	if *jsonFormat {
		if err := json.NewEncoder(os.Stdout).Encode(proj); err != nil {
			fail(err)
		}
	} else {
		fmtTime := fmtSimpleTime
		if *rfc3339Time {
			fmtTime = fmtRFC3339Time
		}

		fmt.Println(proj.Name)
		showField("ID", proj.ID)
		if info := proj.PerHost[host]; host != "" && info != nil {
			showField("Path", info.Path)
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
}

func showField(label string, args ...interface{}) {
	fmt.Printf("%-9s %s", label+":", fmt.Sprintln(args...))
}

func fmtSimpleTime(t time.Time) string {
	return t.Local().Format(time.Stamp)
}

func fmtRFC3339Time(t time.Time) string {
	return t.Format(time.RFC3339)
}

const rfc3339example = "2006-01-02T15:04:05-07:00"

func cmdCreate(args []string) {
	const synopsis = "create [options] NAME"

	now := time.Now()
	proj := &catalog.Project{
		VCS:         new(catalog.VCSInfo),
		CatalogTime: now,
		CreateTime:  now,
	}
	var hostInfo catalog.HostInfo

	fset := newFlagSet("create", synopsis)
	fset.StringVar(&proj.ShortName, "shortname", "", "identifier for project (default is lowercased full name)")
	fset.Var((*tagsList)(&proj.Tags), "tags", "comma-separated tags to assign to the new project")
	fset.StringVar(&hostInfo.Path, "path", "", "path of working copy")
	fset.StringVar(&proj.Homepage, "url", "", "project homepage")
	fset.StringVar(&proj.VCS.Type, "vcs", "", "type of VCS for project")
	fset.StringVar(&proj.VCS.URL, "vcsurl", "", "project VCS URL")
	fset.Var((*timeFlag)(&proj.CreateTime), "created", "project creation date, formatted as RFC3339 ("+rfc3339example+")")
	parseFlags(fset, args)
	if fset.NArg() != 1 {
		exitSynopsis(synopsis)
	}
	cat := requireCatalog()

	name := strings.TrimSpace(fset.Arg(0))
	if len(name) == 0 {
		fail("empty name")
	}
	proj.Name = name
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
		if host == "" {
			fail("-path given and " + HostEnv + " not set")
		}
		proj.PerHost = map[string]*catalog.HostInfo{host: &hostInfo}
	}
	if err := cat.PutProject(proj); err != nil {
		fail(err)
	}
}

func cmdUpdate(args []string) {
	const synopsis = "update [options] PROJECT"

	var (
		name     optStringFlag
		tagsFlag optStringFlag
		path     optStringFlag
		created  optTimeFlag
		homepage optStringFlag
		vcsType  optStringFlag
		vcsURL   optStringFlag
	)

	fset := newFlagSet("update", synopsis)
	fset.Var(&name, "name", "human-readable name of project")
	fset.Var(&tagsFlag, "tags", "comma-separated tags to assign to the new project")
	fset.Var(&path, "path", "path of working copy")
	fset.Var(&created, "created", "project creation date, formatted as RFC3339 ("+rfc3339example+")")
	fset.Var(&homepage, "url", "project homepage")
	fset.Var(&vcsType, "vcs", "type of VCS for project")
	fset.Var(&vcsURL, "vcsurl", "project VCS URL")
	parseFlags(fset, args)
	if fset.NArg() != 1 {
		exitSynopsis(synopsis)
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
		if proj.PerHost == nil {
			proj.PerHost = make(map[string]*catalog.HostInfo)
		}
		if proj.PerHost[host] == nil {
			proj.PerHost[host] = new(catalog.HostInfo)
		}
		proj.PerHost[host].Path = path.s
	}

	if err := cat.PutProject(proj); err != nil {
		fail(err)
	}
}

func updateString(s *string, f *optStringFlag) {
	if f.present {
		*s = f.s
	}
}

func cmdRename(args []string) {
	const synopsis = "rename SRC DST"

	fset := newFlagSet("rename", synopsis)
	parseFlags(fset, args)
	if fset.NArg() != 2 {
		exitSynopsis(synopsis)
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
}

func cmdDelete(args []string) {
	const synopsis = "delete PROJECT [...]"

	fset := newFlagSet("delete", synopsis)
	parseFlags(fset, args)
	if fset.NArg() == 0 {
		exitSynopsis(synopsis)
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
}

func requireCatalog() catalog.Catalog {
	if catalogPath == "" {
		fail(CatalogPathEnv + " not set")
	}
	cat, err := catalog.Open(catalogPath)
	if err != nil {
		fail(err)
	}
	return cat
}
