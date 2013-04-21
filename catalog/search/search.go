// Package search provides text search in Black Forest catalogs.
package search

import (
	"sort"

	"bitbucket.org/zombiezen/blackforest/catalog"
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

// textSearch is a Searcher that can perform a full text search.
type textSearch struct {
	i    map[string][]indexEntry
	tags map[string][]string
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
		i:    make(map[string][]indexEntry),
		tags: make(map[string][]string),
		list: names,
	}
	for _, sn := range names {
		p, err := cat.GetProject(sn)
		if err != nil {
			return nil, err
		}
		ts.build(p)
	}
	return ts, nil
}

// Search parses a query according to the grammar at
// https://bitbucket.org/zombiezen/blackforest/wiki/Search and then finds all
// matching projects, sorted by decreasing relevance.
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

// search executes the query q and stores the matches in results.
func (ts *textSearch) search(q queryAST, results resultMap) {
	switch q := q.(type) {
	case queryAnd:
		ts.searchAnd(q, results)
	case queryOr:
		ts.searchOr(q, results)
	case queryNot:
		ts.searchNot(q, results)
	case term:
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
	// Because AND is an intersection, start with the first subquery and filter
	// the set as we go.  If the number of results is ever zero, we're finished.

	if len(q) == 0 {
		return
	}

	// First subquery
	ts.search(q[0], results)
	if len(results) == 0 {
		return
	}
	for _, r := range results {
		r.Relevance /= float32(len(q))
	}

	// Subsequent subqueries
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

func (ts *textSearch) searchToken(q term, results resultMap) {
	qr := fold([]rune(string(q)))
	for _, ent := range ts.i[string(qr)] {
		results.Get(ent.shortName).Relevance += ent.kind.Weight()
	}
	stripped := stripTokenSep(qr)
	if len(stripped) == len(qr) {
		return
	}
	for _, ent := range ts.i[string(stripped)] {
		results.Get(ent.shortName).Relevance += ent.kind.Weight()
	}
	// XXX(light): should results be normalized?
}

func (ts *textSearch) searchTagAtom(q tagAtom, results resultMap) {
	for _, sn := range ts.tags[foldString(string(q))] {
		results.Put(&Result{ShortName: sn, Relevance: 1.0})
	}
}

// build adds the project to the index.
func (ts *textSearch) build(p *catalog.Project) {
	sn := p.ShortName
	ts.index(sn, kindShortName, [][]rune{fold([]rune(sn))})
	ts.index(sn, kindName, tokenize(fold([]rune(p.Name))))
	ts.index(sn, kindDescription, tokenize(fold([]rune(p.Description))))
	for _, tag := range p.Tags {
		t := fold([]rune(tag))
		ts.indexTag(sn, t)
		ts.index(sn, kindTag, [][]rune{t})
		ts.index(sn, kindTagPart, tokenize(t))
	}
}

// indexTag associates a tag with a short name.
func (ts *textSearch) indexTag(sn string, tag []rune) {
	stag := string(tag)
	indexed := ts.tags[stag]
	for _, indexedShortName := range indexed {
		if indexedShortName == sn {
			return
		}
	}
	indexed = append(indexed, sn)
	ts.tags[stag] = indexed
}

// index associates words with a short name.
func (ts *textSearch) index(sn string, kind entryKind, words [][]rune) {
	for _, w := range words {
		if len(w) > 0 {
			sw := string(w)
			ts.i[sw] = append(ts.i[sw], indexEntry{sn, kind})
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
