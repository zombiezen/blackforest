// Package search provides text search in GLaDOS catalogs.
package search

import (
	"log"
	"sort"
	"strings"
	"unicode"

	"bitbucket.org/zombiezen/glados/catalog"
)

// A Searcher implements a textual search.
type Searcher interface {
	Search(query string) ([]Result, error)
}

// Result stores a search result for a project.
type Result struct {
	ShortName string
	Snippet   string
	Relevance float32
}

type textSearch struct {
	c    catalog.Catalog
	i    map[string][]indexEntry
	list []string
}

// NewTextSearch returns a Searcher that performs full text search over the
// short name, name, tags, and description fields of all projects in a catalog.
// The Searcher maintains its own in-memory index of the catalog.  You must
// create a new index if the underlying catalog is modified.
func NewTextSearch(cat catalog.Catalog) (Searcher, error) {
	names, err := cat.List()
	if err != nil {
		return nil, err
	}
	ts := &textSearch{
		c:    cat,
		i:    make(map[string][]indexEntry),
		list: names,
	}
	for _, n := range names {
		if err := ts.build(n); err != nil {
			return nil, err
		}
	}
	return ts, nil
}

func (ts *textSearch) Search(q string) ([]Result, error) {
	query, err := parseQuery(q)
	if err != nil {
		return nil, err
	} else if query == nil {
		return []Result{}, nil
	}
	m := ts.search(query)
	results := make([]Result, 0, len(m))
	for _, r := range m {
		results = append(results, *r)
	}
	sort.Sort(byRelevance(results))
	return results, nil
}

func (ts *textSearch) search(q queryAST) map[string]*Result {
	switch q := q.(type) {
	case queryAnd:
		return ts.searchAnd(q)
	case queryOr:
		return ts.searchOr(q)
	case queryNot:
		return ts.searchNot(q)
	case token:
		return ts.searchToken(q)
	case tagAtom:
		return ts.searchTagAtom(q)
	}
	panic("unknown queryAST type")
}

// A resultMap is a mapping from short name to Result.
type resultMap map[string]*Result

// Get gets or creates a result for a short name.
func (m resultMap) Get(sn string) *Result {
	r := m[sn]
	if r == nil {
		r = &Result{ShortName: sn}
		m[sn] = r
	}
	return r
}

// Put adds r into m.
func (m resultMap) Put(r *Result) {
	m[r.ShortName] = r
}

func (ts *textSearch) searchAnd(q queryAnd) resultMap {
	if len(q) == 0 {
		return resultMap{}
	}

	// Map
	c := make(chan resultMap)
	for _, subq := range q {
		go func(subq queryAST) {
			c <- ts.search(subq)
		}(subq)
	}
	maps := make([]resultMap, 0, len(q))
	minIdx := -1
	for nret := 0; nret < len(q); nret++ {
		ret := <-c
		maps = append(maps, ret)
		if minIdx == -1 || len(ret) < len(maps[minIdx]) {
			minIdx = nret
		}
	}

	// Reduce
	maxResults := len(maps[minIdx])
	if maxResults == 0 {
		return resultMap{}
	}
	results := make(resultMap, maxResults)
	for sn := range maps[minIdx] {
		result := &Result{
			ShortName: sn,
		}
		for _, m := range maps {
			if r := m[sn]; r == nil {
				result.Relevance = 0
				break
			} else {
				result.Relevance += r.Relevance / float32(len(q))
			}
		}
		if result.Relevance > 0 {
			results.Put(result)
		}
	}
	return results
}

func (ts *textSearch) searchOr(q queryOr) resultMap {
	if len(q) == 0 {
		return resultMap{}
	}

	c := make(chan resultMap)
	for _, subq := range q {
		go func(subq queryAST) {
			c <- ts.search(subq)
		}(subq)
	}
	results := make(resultMap)
	for nret := 0; nret < len(q); nret++ {
		ret := <-c
		for _, r := range ret {
			results.Get(r.ShortName).Relevance += r.Relevance
		}
	}
	return results
}

func (ts *textSearch) searchNot(q queryNot) resultMap {
	m := ts.search(q.ast)
	results := make(resultMap, len(ts.list)-len(m))
	for _, sn := range ts.list {
		if m[sn] == nil {
			results.Put(&Result{ShortName: sn, Relevance: 1.0})
		}
	}
	return results
}

func (ts *textSearch) searchToken(q token) resultMap {
	tsi := ts.i[fold(string(q))]
	if len(tsi) == 0 {
		return resultMap{}
	}
	results := make(resultMap)
	for _, ent := range tsi {
		results.Get(ent.shortName).Relevance += ent.kind.Weight()
	}
	// XXX(light): should results be normalized?
	return results
}

func (ts *textSearch) searchTagAtom(q tagAtom) resultMap {
	results := make(resultMap)
	for _, ent := range ts.i[fold(string(q))] {
		if ent.kind == kindTag {
			results.Put(&Result{ShortName: ent.shortName, Relevance: 1.0})
		}
	}
	return results
}

func (ts *textSearch) build(sn string) error {
	p, err := ts.c.GetProject(sn)
	if err != nil {
		return err
	}
	ts.index(sn, kindShortName, sn)
	ts.index(sn, kindName, tokenize(p.Name)...)
	ts.index(sn, kindDescription, tokenize(p.Description)...)
	for _, tag := range p.Tags {
		ts.index(sn, kindTag, tag)
		if parts := strings.Split(tag, "-"); len(parts) > 1 {
			ts.index(sn, kindTagPart, parts...)
		}
	}
	return nil
}

func (ts *textSearch) index(sn string, kind entryKind, words ...string) {
	for _, w := range words {
		if w != "" {
			w = fold(w)
			ts.i[w] = append(ts.i[w], indexEntry{sn, kind})
		}
	}
}

type indexEntry struct {
	shortName string
	kind      entryKind
}

type entryKind int

// Index entry kinds
const (
	kindDescription entryKind = iota
	kindTagPart
	kindTag
	kindName
	kindShortName
)

var kindWeights = [...]float32{
	kindDescription: 0.01,
	kindTagPart:     0.7,
	kindTag:         0.8,
	kindName:        0.9,
	kindShortName:   0.95,
}

func (k entryKind) Weight() float32 {
	return kindWeights[k]
}

func fold(s string) string {
	runes := make([]rune, 0, len(s))
	for _, r := range s {
		rr := unicode.SimpleFold(r)
		for rr > r {
			rr = unicode.SimpleFold(rr)
		}
		runes = append(runes, rr)
	}
	return string(runes)
}

func tokenize(s string) []string {
	t := strings.Fields(s)
	for i := range t {
		t[i] = fold(t[i])
	}
	return t
}

type byRelevance []Result

func (r byRelevance) Len() int      { return len(r) }
func (r byRelevance) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r byRelevance) Less(i, j int) bool {
	ri, rj := r[i].Relevance, r[j].Relevance
	if ri == rj {
		return r[i].ShortName < r[j].ShortName
	}
	return ri > rj
}

// Concurrent dispatches a search to a list of search systems.
type Concurrent []Searcher

func (s Concurrent) Search(q string) ([]Result, error) {
	c := make(chan []Result)
	for _, ss := range s {
		go func(ss Searcher) {
			results, err := ss.Search(q)
			if err != nil {
				log.Println("search error:", err)
			}
			c <- results
		}(ss)
	}
	results := make([]Result, 0)
	for _ = range s {
		r := <-c
		if len(r) > 0 {
			results = append(results, r...)
		}
	}
	sort.Sort(byRelevance(results))
	return results, nil
}
