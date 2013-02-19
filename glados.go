// The glados command provides a front-end for a GLaDOS catalog.
package main

import (
	"bitbucket.org/zombiezen/glados/catalog"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
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
	"list": cmdList,
	"ls":   cmdList,
	"show": cmdShow,
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
