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
	"init":   cmdInit,
	"list":   cmdList,
	"ls":     cmdList,
	"show":   cmdShow,
	"create": cmdCreate,
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
		// TODO(light): make this a dynamic list
		failf("%q is not a valid -vcs\nvalid choices are: cvs, svn, hg, git, bzr, darcs\n", proj.VCS.Type)
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

func isValidVCSType(t string) bool {
	return t == catalog.CVS || t == catalog.Subversion || t == catalog.Mercurial || t == catalog.Git || t == catalog.Bazaar || t == catalog.Darcs
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

const (
	CatalogPathEnv = "GLADOS_PATH"
	HostEnv        = "GLADOS_HOST"
)

// global flags
var (
	catalogPath string
	host        string
)

func init() {
	catalogPath = os.Getenv(CatalogPathEnv)
	host = os.Getenv(HostEnv)
}

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

// exit codes
const (
	exitSuccess = 0
	exitFailure = 1
	exitUsage   = 2
)
