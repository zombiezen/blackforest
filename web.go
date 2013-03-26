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
	"sync"
	"time"

	"bitbucket.org/zombiezen/glados/catalog"
	"bitbucket.org/zombiezen/glados/catalog/search"
	"bitbucket.org/zombiezen/subcmd"
	"bitbucket.org/zombiezen/webapp"
	"github.com/gorilla/mux"
)

type webEnv struct {
	cat      *catalog.Cache
	realCat  catalog.Catalog
	router   *mux.Router
	tmpl     *template.Template
	searcher search.Searcher

	sync.RWMutex
}

func (env *webEnv) routerPath(name string, pairs ...string) string {
	u, err := env.router.Get(name).URLPath(pairs...)
	if err != nil {
		panic(err)
	}
	return u.Path
}

func cmdWeb(set *subcmd.Set, cmd *subcmd.Command, args []string) error {
	fset := cmd.FlagSet(set)
	addr := fset.String("listen", "localhost:10710", "address to listen for HTTP")
	templateDir := fset.String("templatedir", "templates", "template directory")
	staticDir := fset.String("staticdir", "static", "static directory")
	refresh := fset.Duration("refresh", 1*time.Minute, "interval between catalog cache refreshes")
	parseFlags(fset, args)
	if fset.NArg() != 0 {
		cmd.PrintSynopsis(set)
		return exitError(exitUsage)
	}

	env := new(webEnv)
	env.realCat = requireCatalog()
	var err error
	if env.cat, err = catalog.NewCache(env.realCat); err != nil {
		return err
	}
	if env.searcher, err = search.NewTextSearch(env.cat); err != nil {
		return err
	}

	r := mux.NewRouter()
	r.Handle("/", &handler{env, handleIndex}).Name("index")
	r.Handle("/search", &handler{env, handleSearch}).Name("search")
	r.Handle("/project/", &handler{env, handlePostProject}).Methods("POST").Name("postproject")
	r.Handle("/project/{project}", &handler{env, handleProject}).Methods("GET", "HEAD").Name("project")
	r.Handle("/project/{project}", &handler{env, handlePutProject}).Methods("PUT").Name("putproject")
	r.Handle("/tag/", &handler{env, handleTagIndex}).Name("tagindex")
	r.Handle("/tag/{tag}", &handler{env, handleTag}).Name("tag")
	staticDirRoute(r, "/css/", filepath.Join(*staticDir, "css")).Name("css")
	staticDirRoute(r, "/img/", filepath.Join(*staticDir, "img")).Name("img")
	staticDirRoute(r, "/js/", filepath.Join(*staticDir, "js")).Name("js")
	r.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(*staticDir, "img", "favicon.ico"))
	}).Name("favicon")
	env.router = r

	env.tmpl = template.New("")
	webapp.AddFuncs(env.tmpl, env.router)
	env.tmpl.Funcs(template.FuncMap{
		"prettyurl": prettyurl,
		"ellipsis":  ellipsis,
		"stringeq":  func(a, b string) bool { return a == b },
		"rfc3339":   rfc3339,
	})
	if _, err := env.tmpl.ParseGlob(filepath.Join(*templateDir, "*.html")); err != nil {
		return err
	}

	go refreshEnvJob(env, *refresh)

	return http.ListenAndServe(*addr, env.router)
}

func handleIndex(env *webEnv, w http.ResponseWriter, req *http.Request) error {
	now := time.Now()
	list, err := env.cat.List()
	if err != nil {
		return err
	}
	sort.Strings(list)
	projects := make([]*catalog.Project, 0, len(list))
	for _, sn := range list {
		p, err := env.cat.GetProject(sn)
		if err == nil {
			projects = append(projects, p)
		} else {
			log.Printf("error fetching %s from list: %v", sn, err)
		}
	}
	return env.tmpl.ExecuteTemplate(w, "index.html", struct {
		Projects []*catalog.Project
		Now      time.Time
	}{
		projects, now,
	})
}

func handleSearch(env *webEnv, w http.ResponseWriter, req *http.Request) error {
	query := req.FormValue("q")
	var results []search.Result
	if query != "" {
		var err error
		results, err = env.searcher.Search(query)
		if err != nil {
			return err
		}
	}

	return env.tmpl.ExecuteTemplate(w, "search.html", struct {
		Query   string
		Results []search.Result
	}{
		query, results,
	})
}

func handleProject(env *webEnv, w http.ResponseWriter, req *http.Request) error {
	sn := mux.Vars(req)["project"]
	htmlAccept, jsonAccept := 1.0, 0.0
	if h := req.Header.Get(webapp.HeaderAccept); h != "" {
		accept, err := webapp.ParseAcceptHeader(h)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return nil
		}
		htmlAccept = accept.Quality("text/html", map[string][]string{"charset": {"utf-8"}})
		jsonAccept = accept.Quality("application/json", map[string][]string{"charset": {"utf-8"}})
		if htmlAccept == 0 && jsonAccept == 0 {
			http.Error(w, "projects can either be text/html or application/json", http.StatusNotAcceptable)
			return nil
		}
	}

	proj, err := env.cat.RefreshProject(sn)
	if err != nil {
		return err
	} else if proj == nil {
		return webapp.NotFound
	}
	if jsonAccept > htmlAccept {
		return webapp.JSONResponse(w, proj)
	}
	return env.tmpl.ExecuteTemplate(w, "project.html", proj)
}

func handlePostProject(env *webEnv, w http.ResponseWriter, req *http.Request) error {
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
	if err := env.cat.PutProject(proj); err != nil {
		return err
	}

	projPath := env.routerPath("project", "project", proj.ShortName)
	w.Header().Set(webapp.HeaderLocation, projPath)
	w.Header().Set(webapp.HeaderContentType, webapp.JSONType)
	w.WriteHeader(http.StatusCreated)
	return json.NewEncoder(w).Encode(proj)
}

func handlePutProject(env *webEnv, w http.ResponseWriter, req *http.Request) error {
	sn := mux.Vars(req)["project"]
	if err := req.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	proj, err := env.cat.RefreshProject(sn)
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

	if err := env.cat.PutProject(proj); err != nil {
		return err
	}
	return webapp.JSONResponse(w, proj)
}

type tagSidebar struct {
	Groups []tagGroup
	Active string
}

func handleTagIndex(env *webEnv, w http.ResponseWriter, req *http.Request) error {
	return env.tmpl.ExecuteTemplate(w, "tag-index.html", tagSidebar{Groups: organizeTags(env.cat)})
}

func handleTag(env *webEnv, w http.ResponseWriter, req *http.Request) error {
	tag := mux.Vars(req)["tag"]

	tags := organizeTags(env.cat)

	names := env.cat.FindTag(tag)
	if len(names) == 0 {
		return webapp.NotFound
	}
	sort.Strings(names)
	projects := make([]*catalog.Project, 0, len(names))
	for _, sn := range names {
		p, err := env.cat.GetProject(sn)
		if err == nil {
			projects = append(projects, p)
		} else {
			log.Printf("error fetching %s from list: %v", sn, err)
		}
	}

	return env.tmpl.ExecuteTemplate(w, "tag.html", struct {
		Tag      string
		Sidebar  tagSidebar
		Projects []*catalog.Project
	}{
		tag, tagSidebar{tags, tag}, projects,
	})
}

type handler struct {
	Env  *webEnv
	Func func(env *webEnv, w http.ResponseWriter, req *http.Request) error
}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	method, path := req.Method, req.URL.Path
	rb := new(webapp.ResponseBuffer)

	h.Env.RLock()
	err := h.Func(h.Env, rb, req)
	h.Env.RUnlock()

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

func refreshEnvJob(env *webEnv, d time.Duration) {
	for t := range time.Tick(d) {
		env.RLock()
		cat := env.realCat
		env.RUnlock()

		// Refresh cache and searcher
		// Even though the cache can be recreated using RefreshAll, we don't
		// want to block requests for that long.  Creating a new cache uses a
		// bit more memory during refresh, but does not affect QPS.
		cache, err := catalog.NewCache(cat)
		if err != nil {
			log.Println("refresh cache:", err)
			continue
		}
		searcher, err := search.NewTextSearch(cache)
		if err != nil {
			log.Println("refresh search:", err)
			continue
		}

		// Update environment (temporarily blocks requests)
		env.Lock()
		env.cat = cache
		env.searcher = searcher
		env.Unlock()

		log.Println("refresh took", time.Since(t))
	}
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

func rfc3339(t time.Time) string {
	return t.Format(time.RFC3339)
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
