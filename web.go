package main

import (
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sort"

	"bitbucket.org/zombiezen/glados/catalog"
	"bitbucket.org/zombiezen/subcmd"
	"bitbucket.org/zombiezen/webapp"
	"github.com/gorilla/mux"
)

var tmpl *template.Template

func cmdWeb(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	fset := cmd.FlagSet(set)
	addr := fset.String("listen", ":8080", "address to listen for HTTP")
	templateDir := fset.String("templatedir", "templates", "template directory")
	staticDir := fset.String("staticdir", "static", "static directory")
	parseFlags(fset, args)
	if fset.NArg() != 0 {
		cmd.PrintSynopsis(set)
		return exitError(exitUsage)
	}
	cat := requireCatalog()

	r := mux.NewRouter()
	r.Handle("/", &handler{cat, handleIndex}).Name("index")
	r.Handle("/project/{project}", &handler{cat, handleProject}).Name("project")
	staticDirRoute(r, "/css/", filepath.Join(*staticDir, "css")).Name("css")
	staticDirRoute(r, "/img/", filepath.Join(*staticDir, "img")).Name("img")
	staticDirRoute(r, "/js/", filepath.Join(*staticDir, "js")).Name("js")

	tmpl = template.New("")
	webapp.AddFuncs(tmpl, r)
	if _, err := tmpl.ParseGlob(filepath.Join(*templateDir, "*.html")); err != nil {
		return err
	}

	return http.ListenAndServe(*addr, r)
}

func handleIndex(cat catalog.Catalog, w http.ResponseWriter, req *http.Request) error {
	list, err := cat.List()
	if err != nil {
		return err
	}
	sort.Strings(list)
	projects := make([]*catalog.Project, 0, len(list))
	for _, sn := range list {
		p, err := cat.GetProject(sn)
		if err == nil {
			projects = append(projects, p)
		} else {
			log.Printf("error fetching %s from list: %v", sn, err)
		}
	}
	return tmpl.ExecuteTemplate(w, "index.html", projects)
}

func handleProject(cat catalog.Catalog, w http.ResponseWriter, req *http.Request) error {
	sn := mux.Vars(req)["project"]
	proj, err := cat.GetProject(sn)
	if err != nil {
		return err
	} else if proj == nil {
		return &webapp.NotFound{req.URL}
	}
	return tmpl.ExecuteTemplate(w, "project.html", proj)
}

type handler struct {
	Catalog catalog.Catalog
	Func    func(cat catalog.Catalog, w http.ResponseWriter, req *http.Request) error
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	rb := new(webapp.ResponseBuffer)
	err := h.Func(h.Catalog, rb, req)
	if err == nil {
		if rb.HeaderSent().Get(webapp.HeaderContentLength) == "" {
			webapp.ContentLength(w.Header(), rb.Size())
		}
		if err := rb.Copy(w); err != nil {
			log.Printf("%s send error: %v", path, err)
		}
	} else if _, ok := err.(*webapp.NotFound); ok {
		http.NotFound(w, req)
	} else {
		log.Printf("%s error: %v", path, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func staticDirRoute(r *mux.Router, prefix, path string) *mux.Route {
	route := prefix + "{path:.*}"
	fs := http.FileServer(http.Dir(path))
	return r.HandleFunc(route, func(w http.ResponseWriter, req *http.Request) {
		req.URL.Path = mux.Vars(req)["path"]
		fs.ServeHTTP(w, req)
	})
}
