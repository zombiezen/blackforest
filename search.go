package main

import (
	"log"
	"sort"
	"strings"
	"unicode"

	"bitbucket.org/zombiezen/glados/catalog"
)

type Searcher interface {
	Search(query string) ([]SearchResult, error)
}

type SearchResult struct {
	ShortName string
	Snippet   string
	Relevance float32
}

type textSearch struct {
	c catalog.Catalog
	i map[string][]string
}

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

func (ts *textSearch) Search(q string) ([]SearchResult, error) {
	tokens := tokenize(q)
	if len(tokens) == 0 {
		return nil, nil
	}

	results := make(map[string]*SearchResult)
	for _, tok := range tokens {
		for _, sn := range ts.i[tok] {
			r := results[sn]
			if r == nil {
				r = &SearchResult{ShortName: sn}
				results[sn] = r
			}
			// TODO(light): calculate relevance
		}
	}

	resultSlice := make([]SearchResult, 0, len(results))
	for _, r := range results {
		resultSlice = append(resultSlice, *r)
	}
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

type byRelevance []SearchResult

func (r byRelevance) Len() int           { return len(r) }
func (r byRelevance) Less(i, j int) bool { return r[i].Relevance > r[j].Relevance }
func (r byRelevance) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

type aggregateSearch []Searcher

func (s aggregateSearch) Search(q string) ([]SearchResult, error) {
	c := make(chan []SearchResult)
	for _, ss := range s {
		go func(ss Searcher) {
			results, err := ss.Search(q)
			if err != nil {
				log.Println("search error:", err)
			}
			c <- results
		}(ss)
	}
	results := make([]SearchResult, 0)
	for _ = range s {
		r := <-c
		if len(r) > 0 {
			results = append(results, r...)
		}
	}
	sort.Sort(byRelevance(results))
	return results, nil
}
