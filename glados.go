// The glados command provides a front-end for a GLaDOS catalog.
package main

import (
	"bitbucket.org/zombiezen/glados/catalog"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

func main() {
	fset := flag.NewFlagSet("glados", flag.ContinueOnError)
	fset.SetOutput(os.Stdout)
	fset.Usage = func() {
		fmt.Println("glados [options] COMMAND ...")
		fmt.Println()
		fmt.Println("commands:")
		for i := range commands {
			cmd := &commands[i]
			// TODO(light): only one line of description
			fmt.Printf("  %-8s %s\n", cmd.Name, cmd.Description)
		}
		fmt.Println()
		fmt.Println("options:")
		fset.PrintDefaults()
	}
	globalFlags(fset)
	parseFlags(fset, os.Args[1:])
	if fset.NArg() == 0 {
		fset.Usage()
		os.Exit(exitUsage)
	}
	cname := fset.Arg(0)
	for i := range commands {
		cmd := &commands[i]
		if cmd.Matches(cname) {
			cmd.Func(cmd, fset.Args()[1:])
			os.Exit(exitSuccess)
		}
	}
	fmt.Fprintln(os.Stderr, "unrecognized command:", cname)
	os.Exit(exitUsage)
}

var commands = []subcmd{
	{cmdInit, "init", []string{}, "init -catalog=PATH", "create a catalog"},
	{cmdList, "list", []string{"ls"}, "list", "list project short names"},
	{cmdPath, "path", []string{}, "path PROJECT", "print a project's local path"},
	{cmdShow, "show", []string{}, "show PROJECT [...]", "print projects"},
	{cmdCreate, "create", []string{}, "create [options] NAME", "create a project"},
	{cmdUpdate, "update", []string{"up"}, "update [options] PROJECT", "change project fields"},
	{cmdRename, "rename", []string{"mv"}, "rename SRC DST", "change a project's short name"},
	{cmdDelete, "delete", []string{"del", "rm"}, "delete PROJECT [...]", "delete projects"},
}

func cmdInit(cmd *subcmd, args []string) {
	fset := cmd.NewFlagSet()
	parseFlags(fset, args)
	if fset.NArg() != 0 {
		cmd.ExitSynopsis()
	}

	if catalogPath == "" {
		fail(CatalogPathEnv + " not set")
	}
	if _, err := catalog.Create(catalogPath); err != nil {
		fail(err)
	}
}

func cmdList(cmd *subcmd, args []string) {
	fset := cmd.NewFlagSet()
	parseFlags(fset, args)
	cat := requireCatalog()
	if fset.NArg() != 0 {
		cmd.ExitSynopsis()
	}

	list, err := cat.List()
	if err != nil {
		fail(err)
	}
	for _, name := range list {
		fmt.Println(name)
	}
}

func cmdPath(cmd *subcmd, args []string) {
	fset := cmd.NewFlagSet()
	parseFlags(fset, args)
	if fset.NArg() != 1 {
		cmd.ExitSynopsis()
	}
	cat := requireCatalog()

	if host == "" {
		fail(HostEnv + " not set")
	}
	proj, err := cat.GetProject(fset.Arg(0))
	if err != nil {
		fail(err)
	}
	info := proj.PerHost[host]
	if info == nil || info.Path == "" {
		os.Exit(exitFailure)
	}
	fmt.Println(info.Path)
}

func cmdShow(cmd *subcmd, args []string) {
	fset := cmd.NewFlagSet()
	jsonFormat := fset.Bool("json", false, "print project as JSON")
	rfc3339Time := fset.Bool("rfc3339", false, "print dates as RFC3339")
	parseFlags(fset, args)
	if fset.NArg() == 0 {
		cmd.ExitSynopsis()
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
}

func showProject(proj *catalog.Project, fmtTime func(time.Time) string) {
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

const rfc3339example = "2006-01-02T15:04:05-07:00"

func cmdCreate(cmd *subcmd, args []string) {
	now := time.Now()
	proj := &catalog.Project{
		VCS:         new(catalog.VCSInfo),
		CatalogTime: now,
		CreateTime:  now,
	}
	var hostInfo catalog.HostInfo

	fset := cmd.NewFlagSet()
	fset.StringVar(&proj.ShortName, "shortname", "", "identifier for project (default is lowercased full name)")
	fset.Var((*tagSetFlag)(&proj.Tags), "tags", "comma-separated tags to assign to the new project")
	fset.StringVar(&hostInfo.Path, "path", "", "path of working copy")
	fset.StringVar(&proj.Homepage, "url", "", "project homepage")
	fset.StringVar(&proj.VCS.Type, "vcs", "", "type of VCS for project")
	fset.StringVar(&proj.VCS.URL, "vcsurl", "", "project VCS URL")
	fset.Var((*timeFlag)(&proj.CreateTime), "created", "project creation date, formatted as RFC3339 ("+rfc3339example+")")
	parseFlags(fset, args)
	if fset.NArg() != 1 {
		cmd.ExitSynopsis()
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

func cmdUpdate(cmd *subcmd, args []string) {
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

	fset := cmd.NewFlagSet()
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
		cmd.ExitSynopsis()
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
		if proj.PerHost == nil {
			proj.PerHost = make(map[string]*catalog.HostInfo)
		}
		if proj.PerHost[host] == nil {
			proj.PerHost[host] = new(catalog.HostInfo)
		}
		proj.PerHost[host].Path = path.s
	}

	if tagsFlag.present {
		proj.Tags = catalog.TagSet(tagsFlag.ts)
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
}

func updateString(s *string, f *optStringFlag) {
	if f.present {
		*s = f.s
	}
}

func cmdRename(cmd *subcmd, args []string) {
	fset := cmd.NewFlagSet()
	parseFlags(fset, args)
	if fset.NArg() != 2 {
		cmd.ExitSynopsis()
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

func cmdDelete(cmd *subcmd, args []string) {
	fset := cmd.NewFlagSet()
	parseFlags(fset, args)
	if fset.NArg() == 0 {
		cmd.ExitSynopsis()
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
