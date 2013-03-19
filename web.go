package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"bitbucket.org/zombiezen/glados/catalog"
	"bitbucket.org/zombiezen/glados/catalog/search"
	"bitbucket.org/zombiezen/subcmd"
	"bitbucket.org/zombiezen/webapp"
	"github.com/gorilla/mux"
)

var (
	router   *mux.Router
	tmpl     *template.Template
	searcher search.Searcher
)

func cmdWeb(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	fset := cmd.FlagSet(set)
	addr := fset.String("listen", "localhost:10710", "address to listen for HTTP")
	templateDir := fset.String("templatedir", "templates", "template directory")
	staticDir := fset.String("staticdir", "static", "static directory")
	parseFlags(fset, args)
	if fset.NArg() != 0 {
		cmd.PrintSynopsis(set)
		return exitError(exitUsage)
	}
	cat := requireCatalog()
	var err error
	if cat, err = catalog.NewCache(cat); err != nil {
		return err
	}
	if searcher, err = search.NewTextSearch(cat); err != nil {
		return err
	}

	router = mux.NewRouter()
	router.Handle("/", &handler{cat, handleIndex}).Name("index")
	router.Handle("/search", &handler{cat, handleSearch}).Name("search")
	router.Handle("/project/", &handler{cat, handlePostProject}).Methods("POST").Name("postproject")
	router.Handle("/project/{project}", &handler{cat, handleProject}).Methods("GET", "HEAD").Name("project")
	router.Handle("/project/{project}", &handler{cat, handlePutProject}).Methods("PUT").Name("putproject")
	router.Handle("/tag/", &handler{cat, handleTagIndex}).Name("tagindex")
	router.Handle("/tag/{tag}", &handler{cat, handleTag}).Name("tag")
	staticDirRoute(router, "/css/", filepath.Join(*staticDir, "css")).Name("css")
	staticDirRoute(router, "/img/", filepath.Join(*staticDir, "img")).Name("img")
	staticDirRoute(router, "/js/", filepath.Join(*staticDir, "js")).Name("js")

	tmpl = template.New("")
	webapp.AddFuncs(tmpl, router)
	tmpl.Funcs(template.FuncMap{
		"prettyurl": prettyurl,
		"ellipsis":  ellipsis,
		"stringeq":  func(a, b string) bool { return a == b },
	})
	if _, err := tmpl.ParseGlob(filepath.Join(*templateDir, "*.html")); err != nil {
		return err
	}

	return http.ListenAndServe(*addr, router)
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

func handleSearch(cat catalog.Catalog, w http.ResponseWriter, req *http.Request) error {
	query := req.FormValue("q")
	var results []search.Result
	if query != "" {
		var err error
		results, err = searcher.Search(query)
		if err != nil {
			return err
		}
	}

	return tmpl.ExecuteTemplate(w, "search.html", struct {
		Query   string
		Results []search.Result
	}{
		query, results,
	})
}

func handleProject(cat catalog.Catalog, w http.ResponseWriter, req *http.Request) error {
	sn := mux.Vars(req)["project"]
	proj, err := cat.GetProject(sn)
	if err != nil {
		return err
	} else if proj == nil {
		return webapp.NotFound
	}
	return tmpl.ExecuteTemplate(w, "project.html", proj)
}

func handlePostProject(cat catalog.Catalog, w http.ResponseWriter, req *http.Request) error {
	if err := req.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	delete(req.Form, projectFormAddTagsKey)
	delete(req.Form, projectFormDelTagsKey)
	delete(req.Form, projectFormPathKey)
	proj, err := createProjectForm(req.Form, "")
	if err != nil {
		// TODO(light): handle form errors
		return err
	}
	if err := cat.PutProject(proj); err != nil {
		return err
	}

	projPath := routerPath("project", "project", proj.ShortName)
	w.Header().Set(webapp.HeaderLocation, projPath)
	w.Header().Set(webapp.HeaderContentType, webapp.JSONType)
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(proj)
}

func handlePutProject(cat catalog.Catalog, w http.ResponseWriter, req *http.Request) error {
	sn := mux.Vars(req)["project"]
	if err := req.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	proj, err := cat.GetProject(sn)
	if err != nil {
		return err
	} else if proj == nil {
		return webapp.NotFound
	}

	delete(req.Form, projectFormAddTagsKey)
	delete(req.Form, projectFormDelTagsKey)
	delete(req.Form, projectFormPathKey)
	if err := updateProjectForm(proj, req.Form, ""); err != nil {
		// TODO(light): handle form errors
		return err
	}

	if err := cat.PutProject(proj); err != nil {
		return err
	}
	return webapp.JSONResponse(w, proj)
}

type tagSidebar struct {
	Groups []tagGroup
	Active string
}

func handleTagIndex(cat catalog.Catalog, w http.ResponseWriter, req *http.Request) error {
	cache := cat.(*catalog.Cache)
	return tmpl.ExecuteTemplate(w, "tag-index.html", tagSidebar{Groups: organizeTags(cache)})
}

func handleTag(cat catalog.Catalog, w http.ResponseWriter, req *http.Request) error {
	tag := mux.Vars(req)["tag"]

	cache := cat.(*catalog.Cache)
	tags := organizeTags(cache)

	names := cache.FindTag(tag)
	if len(names) == 0 {
		return webapp.NotFound
	}
	sort.Strings(names)
	projects := make([]*catalog.Project, 0, len(names))
	for _, sn := range names {
		p, err := cat.GetProject(sn)
		if err == nil {
			projects = append(projects, p)
		} else {
			log.Printf("error fetching %s from list: %v", sn, err)
		}
	}

	return tmpl.ExecuteTemplate(w, "tag.html", struct {
		Tag      string
		Sidebar  tagSidebar
		Projects []*catalog.Project
	}{
		tag, tagSidebar{tags, tag}, projects,
	})
}

type handler struct {
	Catalog catalog.Catalog
	Func    func(cat catalog.Catalog, w http.ResponseWriter, req *http.Request) error
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	method, path := req.Method, req.URL.Path
	rb := new(webapp.ResponseBuffer)
	err := h.Func(h.Catalog, rb, req)
	if err == nil {
		if rb.HeaderSent().Get(webapp.HeaderContentLength) == "" {
			webapp.ContentLength(w.Header(), rb.Size())
		}
		if method == "HEAD" {
			h := w.Header()
			for k, v := range rb.HeaderSent() {
				h[k] = v
			}
			w.WriteHeader(rb.StatusCode())
		} else {
			if err := rb.Copy(w); err != nil {
				log.Printf("%s send error: %v", path, err)
			}
		}
	} else if webapp.IsNotFound(err) {
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

type tagInfo struct {
	Tag   string
	Count int
}

type tagGroup struct {
	Label string
	Tags  []tagInfo
}

type tagFinder interface {
	Tags() []string
	FindTag(tag string) []string
}

// organizeTags splits the list of tags from a cache into groups and retrieves
// the number of projects each tag has.
func organizeTags(finder tagFinder) []tagGroup {
	tags := finder.Tags()
	sort.Strings(tags)

	misc := make([]tagInfo, 0, len(tags))
	groups := []tagGroup{}
	for i := 0; i < len(tags); {
		t := tags[i]
		info := tagInfo{Tag: t, Count: len(finder.FindTag(t))}
		if dash := strings.IndexRune(t, '-'); dash != -1 {
			label, prefix := t[:dash], t[:dash+1]
			j := i + 1
			for ; j < len(tags) && strings.HasPrefix(tags[j], prefix); j++ {
			}
			if j-i > 1 {
				infos := make([]tagInfo, j-i)
				for i, t := range tags[i:j] {
					infos[i] = tagInfo{Tag: t, Count: len(finder.FindTag(t))}
				}
				groups = append(groups, tagGroup{Label: label, Tags: infos})
				i = j
				continue
			}
		}

		misc = append(misc, info)
		i++
	}
	sort.Sort(Stable(byTagCount(misc)))
	groups = append(groups, tagGroup{Tags: misc})
	return groups
}

type byTagCount []tagInfo

func (t byTagCount) Len() int           { return len(t) }
func (t byTagCount) Less(i, j int) bool { return t[i].Count > t[j].Count }
func (t byTagCount) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }

func routerPath(name string, pairs ...string) string {
	u, err := router.Get(name).URLPath(pairs...)
	if err != nil {
		panic(err)
	}
	return u.Path
}

func prettyurl(u string) string {
	if uu, err := url.Parse(u); err == nil {
		if uu.Scheme == "http" || uu.Scheme == "https" {
			u = uu.Host
		} else {
			u = uu.Scheme + "://" + uu.Host
		}
		if uu.Path != "/" {
			u += uu.Path
		}
	}
	return u
}

func ellipsis(n int, s string) string {
	const (
		width    = 3
		ellipsis = "…"
	)

	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-width]) + ellipsis
}

type stable struct {
	x    sort.Interface
	perm []int
}

func (s *stable) Len() int { return len(s.perm) }

func (s *stable) Less(i, j int) bool {
	return s.x.Less(i, j) || !s.x.Less(j, i) && s.perm[i] < s.perm[j]
}

func (s *stable) Swap(i, j int) {
	s.x.Swap(i, j)
	s.perm[i], s.perm[j] = s.perm[j], s.perm[i]
}

func Stable(x sort.Interface) sort.Interface {
	s := &stable{
		x:    x,
		perm: make([]int, x.Len()),
	}
	for i := range s.perm {
		s.perm[i] = i
	}
	return s
}
