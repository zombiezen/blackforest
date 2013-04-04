// Package search provides text search in GLaDOS catalogs.
package search

import (
	"log"
	"sort"
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
	tsi := ts.i[sanitizeTerm(string(q))]
	if len(tsi) == 0 {
		return
	}
	for _, ent := range tsi {
		results.Get(ent.shortName).Relevance += ent.kind.Weight()
	}
	// XXX(light): should results be normalized?
}

func (ts *textSearch) searchTagAtom(q tagAtom, results resultMap) {
	for _, ent := range ts.i[sanitizeTerm(string(q))] {
		if ent.kind == kindTag {
			results.Put(&Result{ShortName: ent.shortName, Relevance: 1.0})
		}
	}
}

func sanitizeTerm(s string) string {
	r := []rune(s)
	fold(r)
	for i := len(r) - 1; i >= 0; i-- {
		if !isTokenizeRune(r[i]) {
			copy(r[i:], r[i+1:])
			r = r[:len(r)-1]
		}
	}
	return string(r)
}

func (ts *textSearch) build(sn string) error {
	p, err := ts.c.GetProject(sn)
	if err != nil {
		return err
	}
	ts.index(sn, kindShortName, [][]rune{fold([]rune(sn))})
	ts.index(sn, kindName, tokenize(fold([]rune(p.Name))))
	ts.index(sn, kindDescription, tokenize(fold([]rune(p.Description))))
	for _, tag := range p.Tags {
		t := fold([]rune(tag))
		ts.index(sn, kindTag, [][]rune{t})
		if parts := tokenize(t); len(parts) > 1 {
			ts.index(sn, kindTagPart, parts)
		}
	}
	return nil
}

func (ts *textSearch) index(sn string, kind entryKind, words [][]rune) {
	for _, w := range words {
		if len(w) > 0 {
			sw := string(w)
			ts.i[sw] = append(ts.i[sw], indexEntry{sn, kind})
		}
	}
}

// fold changes every rune in s to its least equivalent folded case, according
// to unicode.SimpleFold and returns s.
func fold(s []rune) []rune {
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			// the only characters in ASCII that need folding are lowercase
			s[i] = r - 'a' + 'A'
		case r < 128:
			// do nothing
		default:
			rr := unicode.SimpleFold(r)
			for rr > r {
				rr = unicode.SimpleFold(rr)
			}
			s[i] = rr
		}
	}
	return s
}

// tokenize splits a slice of runes s around each instance of one or more
// consecutive non-alphanumeric characters, returning an array of folded
// substrings or an empty list if s contains only non-alphanumerics.
func tokenize(s []rune) [][]rune {
	// borrowed from strings.FieldsFunc in standard library

	n := 0
	inField := false
	for _, rune := range s {
		wasInField := inField
		inField = isTokenizeRune(rune)
		if inField && !wasInField {
			n++
		}
	}

	a := make([][]rune, n)
	na := 0
	fieldStart := -1
	for i, rune := range s {
		if !isTokenizeRune(rune) {
			if fieldStart >= 0 {
				a[na] = s[fieldStart:i]
				na++
				fieldStart = -1
			}
		} else if fieldStart == -1 {
			fieldStart = i
		}
	}
	if fieldStart >= 0 {
		a[na] = s[fieldStart:]
	}

	return a
}

var tokenizeRanges = []*unicode.RangeTable{unicode.Letter, unicode.Number}

func isTokenizeRune(r rune) bool {
	return unicode.IsOneOf(tokenizeRanges, r)
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
