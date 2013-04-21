package main

import (
	"encoding/json"
	"html"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"bitbucket.org/zombiezen/blackforest/catalog"
	"bitbucket.org/zombiezen/blackforest/catalog/search"
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
		"prettyurl":     prettyurl,
		"ellipsis":      ellipsis,
		"stringeq":      func(a, b string) bool { return a == b },
		"inteq":         func(a, b int) bool { return a == b },
		"rfc3339":       rfc3339,
		"milliseconds":  milliseconds,
		"prevPage":      prevPage,
		"nextPage":      nextPage,
		"prevPageList":  prevPageList,
		"nextPageList":  nextPageList,
		"searchSnippet": searchSnippet,
	})
	if _, err := env.tmpl.ParseGlob(filepath.Join(*templateDir, "*.html")); err != nil {
		return err
	}

	go func() {
		for t := range time.Tick(*refresh) {
			if err := refreshEnv(env); err == nil {
				log.Println("refresh took", time.Since(t))
			} else {
				log.Println("refresh failed:", err)
			}
		}
	}()

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
	const perPage = 10

	type projectResult struct {
		*catalog.Project
		search.Result
	}
	var v struct {
		Query     string
		Results   []projectResult
		NResults  int
		TimeTaken time.Duration

		Page      int
		PageCount int
	}
	var params struct {
		Query string `schema:"q"`
		Page  int    `schema:"page"`
	}
	params.Page = 1
	if err := req.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}
	if err := decoder.Decode(&params, req.Form); err != nil {
		log.Println("search form decode:", err)
	}

	v.Query = params.Query
	v.Page = params.Page
	if v.Query != "" {
		start := time.Now()
		results, err := env.searcher.Search(v.Query)
		v.TimeTaken = time.Since(start)
		if err != nil {
			return err
		}
		v.NResults = len(results)
		v.PageCount = (len(results) + perPage - 1) / perPage
		if v.Page < 1 || (v.Page > v.PageCount && v.PageCount != 0) {
			return webapp.NotFound
		}

		v.Results = make([]projectResult, 0, perPage)
		for i := (v.Page - 1) * perPage; i < v.Page*perPage && i < len(results); i++ {
			r := results[i]
			p, err := env.cat.GetProject(r.ShortName)
			if err != nil {
				return err
			}
			if p != nil {
				v.Results = append(v.Results, projectResult{p, r})
			}
		}
	}

	return env.tmpl.ExecuteTemplate(w, "search.html", v)
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

func refreshEnv(env *webEnv) error {
	env.RLock()
	cat := env.realCat
	env.RUnlock()

	// Refresh cache and searcher
	// Even though the cache can be recreated using RefreshAll, we don't
	// want to block requests for that long.  Creating a new cache uses a
	// bit more memory during refresh, but does not affect QPS.
	cache, err := catalog.NewCache(cat)
	if err != nil {
		return err
	}
	searcher, err := search.NewTextSearch(cache)
	if err != nil {
		return err
	}

	// Update environment (temporarily blocks requests)
	env.Lock()
	env.cat = cache
	env.searcher = searcher
	env.Unlock()

	return nil
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

func milliseconds(d time.Duration) string {
	const (
		suffix     = "ms"
		unit       = time.Millisecond
		fracPrec   = unit / 1000
		fracDigits = 3
	)

	ms := strconv.FormatInt(int64(d/unit), 10)
	frac := strconv.FormatInt(int64((d%unit)/fracPrec), 10)
	for len(frac) < fracDigits {
		frac = "0" + frac
	}
	return ms + "." + frac + suffix
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

func prevPage(curr, n int) int {
	if curr <= 1 {
		return 0
	}
	return curr - 1
}

func nextPage(curr, n int) int {
	if curr >= n {
		return 0
	}
	return curr + 1
}

func prevPageList(curr, n int, size int) []int {
	var start, end int
	end = curr - 1
	switch {
	case curr < 1+size/2 || n < size:
		start = 1
	case curr > n-size/2:
		start = n - size + 1
	default:
		start = curr - size/2
	}
	if start > end {
		return []int{}
	}
	list := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		list = append(list, i)
	}
	return list
}

func nextPageList(curr, n int, size int) []int {
	var start, end int
	start = curr + 1
	switch {
	case curr > n-size/2 || n < size:
		end = n
	case curr < 1+size/2:
		end = size
	default:
		end = curr + size/2
	}
	if start > end {
		return []int{}
	}
	list := make([]int, 0, end-start+1)
	for i := start; i <= end; i++ {
		list = append(list, i)
	}
	return list
}

func searchSnippet(query string, description string) template.HTML {
	pairs := search.FindTerms(query, description)
	parts := make([]string, 0, len(pairs)+1)
	last := 0
	for _, pos := range pairs {
		parts = append(parts, description[last:pos])
		last = pos
	}
	parts = append(parts, description[last:])

	for i := range parts {
		parts[i] = html.EscapeString(parts[i])
		if i%2 == 1 {
			parts[i] = "<b>" + parts[i] + "</b>"
		}
	}
	return template.HTML(strings.Join(parts, ""))
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
