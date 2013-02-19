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
		fail("GLADOS_PATH not set")
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
		if proj.Description != "" {
			fmt.Println("\n" + proj.Description)
		}
	}
}

func cmdCreate(args []string) {
	const synopsis = "create [options] NAME"

	fset := newFlagSet("create", synopsis)
	shortName := fset.String("shortname", "", "identifier for project (default is lowercased full name)")
	tagsFlag := fset.String("tags", "", "comma-separated tags to assign to the new project")
	parseFlags(fset, args)
	if fset.NArg() != 1 {
		exitSynopsis(synopsis)
	}
	cat := requireCatalog()

	name := strings.TrimSpace(fset.Arg(0))
	if len(name) == 0 {
		fail("empty name")
	}
	if *shortName == "" {
		*shortName = sanitizeName(name)
	}
	tags := splitTags(*tagsFlag)
	id, err := catalog.GenerateID()
	if err != nil {
		fail(err)
	}
	proj := &catalog.Project{
		ID:        id,
		Name:      name,
		ShortName: *shortName,
		Tags:      tags,
	}
	if err := cat.PutProject(proj); err != nil {
		fail(err)
	}
}

func splitTags(val string) []string {
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
	return tags
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

// global flags
var (
	catalogPath string
)

func init() {
	catalogPath = os.Getenv("GLADOS_PATH")
}

func newFlagSet(name string, synopsis string) *flag.FlagSet {
	fset := flag.NewFlagSet(name, flag.ContinueOnError)
	fset.StringVar(&catalogPath, "catalog", catalogPath, "path to catalog directory")
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
		fail("GLADOS_PATH not set")
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
