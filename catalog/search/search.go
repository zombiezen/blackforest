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
	c catalog.Catalog
	i map[string][]string
}

// NewTextSearch returns a Searcher that performs full text search over the
// short name, name, and description fields of all projects in a catalog.
// The Searcher maintains its own in-memory index of the catalog.  You must
// create a new index if the underlying catalog is modified.
func NewTextSearch(cat catalog.Catalog) (Searcher, error) {
	ts := &textSearch{
		c: cat,
		i: make(map[string][]string),
	}
	names, err := cat.List()
	if err != nil {
		return nil, err
	}
	for _, n := range names {
		if err := ts.build(n); err != nil {
			return nil, err
		}
	}
	return ts, nil
}

func (ts *textSearch) Search(q string) ([]Result, error) {
	tokens := tokenize(q)
	if len(tokens) == 0 {
		return nil, nil
	}

	results := make(map[string]*Result)
	tokenScale := 1.0 / float32(len(tokens))
	for _, tok := range tokens {
		tsi := ts.i[tok]
		quantum := 1.0 / float32(len(tsi))
		for _, sn := range tsi {
			r := results[sn]
			if r == nil {
				r = &Result{ShortName: sn}
				results[sn] = r
			}
			r.Relevance += quantum * tokenScale
		}
	}

	resultSlice := make([]Result, 0, len(results))
	for _, r := range results {
		resultSlice = append(resultSlice, *r)
	}
	sort.Sort(byRelevance(resultSlice))
	return resultSlice, nil
}

func (ts *textSearch) build(sn string) error {
	p, err := ts.c.GetProject(sn)
	if err != nil {
		return err
	}
	words := append(tokenize(p.Name), tokenize(p.Description)...)
	for _, w := range words {
		ts.i[w] = append(ts.i[w], sn)
	}
	return nil
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

func (r byRelevance) Len() int           { return len(r) }
func (r byRelevance) Less(i, j int) bool { return r[i].Relevance > r[j].Relevance }
func (r byRelevance) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

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
