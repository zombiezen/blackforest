// The glados command provides a front-end for a GLaDOS catalog.
package main

import (
	"bitbucket.org/zombiezen/glados/catalog"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
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
		fmt.Println(proj.Name)
		fmt.Println("ID:  ", proj.ID)
		if info := proj.PerHost[host]; host != "" && info != nil {
			fmt.Println("Path:", info.Path)
		}
		if len(proj.Tags) != 0 {
			fmt.Print("Tags: ")
			sort.Strings(proj.Tags)
			for i, tag := range proj.Tags {
				if i != 0 {
					fmt.Print(", ")
				}
				fmt.Print(tag)
			}
			fmt.Println()
		}
		if proj.Homepage != "" {
			fmt.Println("URL: ", proj.Homepage)
		}
		if vcsInfo := proj.VCS; vcsInfo != nil {
			if vcsInfo.URL != "" {
				fmt.Println("VCS: ", vcsInfo.Type, vcsInfo.URL)
			} else {
				fmt.Println("VCS: ", vcsInfo.Type)
			}
		}
		if proj.Description != "" {
			fmt.Println("\n" + proj.Description)
		}
	}
}

func cmdCreate(args []string) {
	const synopsis = "create [options] NAME"

	proj := &catalog.Project{
		VCS: new(catalog.VCSInfo),
	}
	var hostInfo catalog.HostInfo

	fset := newFlagSet("create", synopsis)
	fset.StringVar(&proj.ShortName, "shortname", "", "identifier for project (default is lowercased full name)")
	fset.Var((*tagsList)(&proj.Tags), "tags", "comma-separated tags to assign to the new project")
	fset.StringVar(&hostInfo.Path, "path", "", "path of working copy")
	fset.StringVar(&proj.Homepage, "url", "", "project homepage")
	fset.StringVar(&proj.VCS.Type, "vcs", "", "type of VCS for project")
	fset.StringVar(&proj.VCS.URL, "vcsurl", "", "project VCS URL")
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
		name        optStringFlag
		tagsFlag    optStringFlag
		addTagsFlag optStringFlag
		delTagsFlag optStringFlag
		path        optStringFlag
		homepage    optStringFlag
		vcsType     optStringFlag
		vcsURL      optStringFlag
	)

	fset := newFlagSet("update", synopsis)
	fset.Var(&name, "name", "human-readable name of project")
	fset.Var(&tagsFlag, "tags", "comma-separated tags to assign to the project. This removes any previous flags. Cannot be used with addtags or deltags")
	fset.Var(&addTagsFlag, "addtags", "comma-separated tags to add to the project. This retains any previous flags")
	fset.Var(&delTagsFlag, "deltags", "comma-separated tags to remove from the project. This retains any unmentioned previous flags")
	fset.Var(&path, "path", "path of working copy")
	fset.Var(&homepage, "url", "project homepage")
	fset.Var(&vcsType, "vcs", "type of VCS for project")
	fset.Var(&vcsURL, "vcsurl", "project VCS URL")
	parseFlags(fset, args)

	if fset.NArg() != 1 {
		exitSynopsis(synopsis)
	}

	// for tag operations, make sure that if setting, then not adding or deleting
	if tagsFlag.present && (addTagsFlag.present || delTagsFlag.present) {
		fail("Cannot use -tags flag with -addtags or -deltags")
	}

	cat := requireCatalog()

	shortName := fset.Arg(0)
	proj, err := cat.GetProject(shortName)
	if err != nil {
		fail(err)
	}

	updateString(&proj.Name, &name)
	updateString(&proj.Homepage, &homepage)
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
		// set the tags to what the flag had
		proj.Tags = strings.Split(tagsFlag.String(), ",")
	}

	if addTagsFlag.present {
		// add mentioned tags
		for _, t := range strings.Split(addTagsFlag.String(), ",") {
			alreadyHas := false

			for _, str := range proj.Tags {
				if str == t {
					alreadyHas = true
					break
				}
			}

			if !alreadyHas {
				proj.Tags = append(proj.Tags, t)
			}
		}
	}

	if delTagsFlag.present {
		// remove mentioned tags
		for _, t := range strings.Split(delTagsFlag.String(), ",") {
			proj.Tags = removeTag(proj.Tags, t)
		}
	}

	if err := cat.PutProject(proj); err != nil {
		fail(err)
	}
}

func removeTag(tags []string, tag string) []string {
	for i, str := range tags {
		if str == tag {
			return append(tags[:i], tags[i+1:]...)
		}
	}
	return tags
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
