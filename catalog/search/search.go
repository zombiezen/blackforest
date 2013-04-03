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
	m := make(resultMap)
	ts.search(query, m)
	results := make([]Result, 0, len(m))
	for _, r := range m {
		results = append(results, *r)
	}
	sort.Sort(byRelevance(results))
	return results, nil
}

func (ts *textSearch) search(q queryAST, results resultMap) {
	switch q := q.(type) {
	case queryAnd:
		ts.searchAnd(q, results)
	case queryOr:
		ts.searchOr(q, results)
	case queryNot:
		ts.searchNot(q, results)
	case token:
		ts.searchToken(q, results)
	case tagAtom:
		ts.searchTagAtom(q, results)
	default:
		panic("unknown queryAST type")
	}
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

// Clear deletes all keys from m.
func (m resultMap) Clear() {
	for k := range m {
		delete(m, k)
	}
}

func (ts *textSearch) searchAnd(q queryAnd, results resultMap) {
	// Because AND is an intersection, start with the first term and filter
	// the set as we go.  If the number of results is ever zero, we're finished.

	if len(q) == 0 {
		return
	}

	// First term
	ts.search(q[0], results)
	if len(results) == 0 {
		return
	}
	for _, r := range results {
		r.Relevance /= float32(len(q))
	}

	// Subsequent terms
	m := make(resultMap)
	for _, subq := range q[1:] {
		ts.search(subq, m)
		for sn, r0 := range results {
			if r, ok := m[sn]; ok {
				r0.Relevance += r.Relevance / float32(len(q))
			} else {
				delete(results, sn)
				if len(results) == 0 {
					return
				}
			}
		}
		m.Clear()
	}
}

func (ts *textSearch) searchOr(q queryOr, results resultMap) {
	if len(q) == 0 {
		return
	}

	m := make(resultMap)
	for _, subq := range q {
		ts.search(subq, m)
		for _, r := range m {
			results.Get(r.ShortName).Relevance += r.Relevance
		}
		m.Clear()
	}
}

func (ts *textSearch) searchNot(q queryNot, results resultMap) {
	m := make(resultMap)
	ts.search(q.ast, m)
	for _, sn := range ts.list {
		if m[sn] == nil {
			results.Put(&Result{ShortName: sn, Relevance: 1.0})
		}
	}
}

func (ts *textSearch) searchToken(q token, results resultMap) {
	tsi := ts.i[fold(string(q))]
	if len(tsi) == 0 {
		return
	}
	for _, ent := range tsi {
		results.Get(ent.shortName).Relevance += ent.kind.Weight()
	}
	// XXX(light): should results be normalized?
}

func (ts *textSearch) searchTagAtom(q tagAtom, results resultMap) {
	for _, ent := range ts.i[fold(string(q))] {
		if ent.kind == kindTag {
			results.Put(&Result{ShortName: ent.shortName, Relevance: 1.0})
		}
	}
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
	runes := []rune(s)
	for i, r := range runes {
		switch {
		case r >= 'a' && r <= 'z':
			// the only characters in ASCII that need folding are lowercase
			runes[i] = r - 'a' + 'A'
		case r < 128:
			// do nothing
		default:
			rr := unicode.SimpleFold(r)
			for rr > r {
				rr = unicode.SimpleFold(rr)
			}
			runes[i] = rr
		}
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
