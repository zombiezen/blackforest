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
	i map[string][]indexEntry
}

// NewTextSearch returns a Searcher that performs full text search over the
// short name, name, tags, and description fields of all projects in a catalog.
// The Searcher maintains its own in-memory index of the catalog.  You must
// create a new index if the underlying catalog is modified.
func NewTextSearch(cat catalog.Catalog) (Searcher, error) {
	ts := &textSearch{
		c: cat,
		i: make(map[string][]indexEntry),
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

	// Get results for each individual search term
	type im struct {
		i int
		m map[string]*Result
	}
	c := make(chan im)
	for i, tok := range tokens {
		go func(i int, tok string) {
			results := make(map[string]*Result)
			tsi := ts.i[tok]
			if len(tsi) == 0 {
				c <- im{i, results}
				return
			}

			for _, ent := range tsi {
				sn := ent.shortName
				r := results[sn]
				if r == nil {
					r = &Result{ShortName: sn}
					results[sn] = r
				}
				r.Relevance += ent.kind.Weight()
			}

			var maxScore float32
			for _, r := range results {
				if r.Relevance > maxScore {
					maxScore = r.Relevance
				}
			}
			for _, r := range results {
				r.Relevance /= maxScore
			}

			c <- im{i, results}
		}(i, tok)
	}

	// Collect results for each term
	master := make([]map[string]*Result, len(tokens))
	minIdx := -1
	for nret := 0; nret < len(tokens); nret++ {
		ret := <-c
		master[ret.i] = ret.m
		if minIdx == -1 || len(ret.m) < len(master[minIdx]) {
			minIdx = ret.i
		}
	}
	maxResults := len(master[minIdx])
	if maxResults == 0 {
		return []Result{}, nil
	}

	// Filter results and re-normalize relevance
	resultSlice := make([]Result, 0, maxResults)
	for sn := range master[minIdx] {
		result := Result{
			ShortName: sn,
		}
		for _, m := range master {
			if r := m[sn]; r == nil {
				result.Relevance = 0
				break
			} else {
				result.Relevance += r.Relevance / float32(len(tokens))
			}
		}
		if result.Relevance > 0 {
			resultSlice = append(resultSlice, result)
		}
	}
	sort.Sort(byRelevance(resultSlice))
	return resultSlice, nil
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
