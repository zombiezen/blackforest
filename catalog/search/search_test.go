package search

import (
	"strconv"
	"testing"

	"bitbucket.org/zombiezen/glados/catalog"
)

func TestTextSearch(t *testing.T) {
	tests := []struct {
		Query   string
		Catalog catalog.Catalog
		Results []string
	}{
		{
			"",
			mockCatalog{},
			[]string{},
		},
		{
			"go",
			mockCatalog{},
			[]string{},
		},
		{
			"go",
			mockCatalog{
				"go": &catalog.Project{
					ShortName:   "go",
					Name:        "Go",
					Tags:        catalog.TagSet{"compiler", "external", "lang-c", "lang-go", "language"},
					Description: "Go is an open source programming environment that makes it easy to build simple, reliable, and efficient software.",
				},
			},
			[]string{"go"},
		},
		{
			"programming",
			mockCatalog{
				"go": &catalog.Project{
					ShortName:   "go",
					Name:        "Go",
					Tags:        catalog.TagSet{"compiler", "external", "lang-c", "lang-go", "language"},
					Description: "Go is an open source programming environment that makes it easy to build simple, reliable, and efficient software.",
				},
			},
			[]string{"go"},
		},
		{
			"bacon",
			mockCatalog{
				"go": &catalog.Project{
					ShortName:   "go",
					Name:        "Go",
					Tags:        catalog.TagSet{"compiler", "external", "lang-c", "lang-go", "language"},
					Description: "Go is an open source programming environment that makes it easy to build simple, reliable, and efficient software.",
				},
			},
			[]string{},
		},
		{
			"go",
			mockCatalog{
				"go": &catalog.Project{
					ShortName:   "go",
					Name:        "Go",
					Tags:        catalog.TagSet{"compiler", "external", "lang-c", "lang-go", "language"},
					Description: "Go is an open source programming environment that makes it easy to build simple, reliable, and efficient software.",
				},
				"python": &catalog.Project{
					ShortName:   "python",
					Name:        "Python",
					Tags:        catalog.TagSet{"interpreter", "external", "lang-c", "lang-python", "language"},
					Description: "Python is a general-purpose, high-level programming language whose design philosophy emphasizes code readability.",
				},
			},
			[]string{"go"},
		},
		{
			"language python",
			mockCatalog{
				"go": &catalog.Project{
					ShortName:   "go",
					Name:        "Go",
					Tags:        catalog.TagSet{"compiler", "external", "lang-c", "lang-go", "language"},
					Description: "Go is an open source programming environment that makes it easy to build simple, reliable, and efficient software.",
				},
				"python": &catalog.Project{
					ShortName:   "python",
					Name:        "Python",
					Tags:        catalog.TagSet{"interpreter", "external", "lang-c", "lang-python", "language"},
					Description: "Python is a general-purpose, high-level programming language whose design philosophy emphasizes code readability.",
				},
			},
			[]string{"python"},
		},
		{
			"GO OR PYTHON",
			mockCatalog{
				"go": &catalog.Project{
					ShortName:   "go",
					Name:        "Go",
					Tags:        catalog.TagSet{"compiler", "external", "lang-c", "lang-go", "language"},
					Description: "Go is an open source programming environment that makes it easy to build simple, reliable, and efficient software.",
				},
				"python": &catalog.Project{
					ShortName:   "python",
					Name:        "Python",
					Tags:        catalog.TagSet{"interpreter", "external", "lang-c", "lang-python", "language"},
					Description: "Python is a general-purpose, high-level programming language whose design philosophy emphasizes code readability.",
				},
			},
			[]string{"go", "python"},
		},
		{
			"PYTHON OR GO",
			mockCatalog{
				"go": &catalog.Project{
					ShortName:   "go",
					Name:        "Go",
					Tags:        catalog.TagSet{"compiler", "external", "lang-c", "lang-go", "language"},
					Description: "Go is an open source programming environment that makes it easy to build simple, reliable, and efficient software.",
				},
				"python": &catalog.Project{
					ShortName:   "python",
					Name:        "Python",
					Tags:        catalog.TagSet{"interpreter", "external", "lang-c", "lang-python", "language"},
					Description: "Python is a general-purpose, high-level programming language whose design philosophy emphasizes code readability.",
				},
			},
			[]string{"go", "python"},
		},
		{
			"PYTHON OR GO",
			mockCatalog{
				"go": &catalog.Project{
					ShortName:   "go",
					Name:        "Go",
					Tags:        catalog.TagSet{"compiler", "external", "lang-c", "lang-go", "language"},
					Description: "Go is an open source programming environment that makes it easy to build simple, reliable, and efficient software.",
				},
				"python": &catalog.Project{
					ShortName:   "python",
					Name:        "Python",
					Tags:        catalog.TagSet{"interpreter", "external", "lang-c", "lang-python", "language"},
					Description: "Python is a general-purpose, high-level programming language whose design philosophy emphasizes code readability.",
				},
				"bacon": &catalog.Project{
					ShortName:   "bacon",
					Name:        "Bacon",
					Tags:        catalog.TagSet{"tasty", "breakfast"},
					Description: "Bacon is a meat product that is quite delicious.",
				},
			},
			[]string{"go", "python"},
		},
		{
			"go",
			mockCatalog{
				"go": &catalog.Project{
					ShortName:   "go",
					Name:        "Go",
					Tags:        catalog.TagSet{"compiler", "external", "lang-c", "lang-go", "language"},
					Description: "Go is an open source programming environment that makes it easy to build simple, reliable, and efficient software.",
				},
				"aaaa": &catalog.Project{
					ShortName:   "aaaa",
					Name:        "SCons Go Tools",
					Tags:        catalog.TagSet{"build", "lang-python", "scons"},
					Description: "SCons Go Tools is a collection of builders that makes it easy to compile Go projects in SCons.",
				},
			},
			[]string{"go", "aaaa"},
		},
	}
	for _, test := range tests {
		ts, err := NewTextSearch(test.Catalog)
		if err != nil {
			t.Errorf("NewTextSearch(%v) failed: %v", test.Catalog, err)
			continue
		}
		r, err := ts.Search(test.Query)
		if err != nil {
			t.Errorf("ts.Query(%q) error: %v", test.Query, err)
		}
		results := make([]string, len(r))
		for i := range r {
			results[i] = r[i].ShortName
		}
		if len(results) != len(test.Results) {
			t.Errorf("ts.Query(%q) = %v; want %v", test.Query, results, test.Results)
		} else {
			for i := range results {
				if results[i] != test.Results[i] {
					t.Errorf("ts.Query(%q) = %v; want %v", test.Query, results, test.Results)
					break
				}
			}
		}
	}
}

func newTestCatalog() catalog.Catalog {
	cat := mockCatalog{
		"go": &catalog.Project{
			ShortName:   "go",
			Name:        "Go",
			Tags:        catalog.TagSet{"compiler", "external", "lang-c", "lang-go", "language"},
			Description: "Go is an open source programming environment that makes it easy to build simple, reliable, and efficient software.",
		},
		"python": &catalog.Project{
			ShortName:   "python",
			Name:        "Python",
			Tags:        catalog.TagSet{"interpreter", "external", "lang-c", "lang-python", "language"},
			Description: "Python is a general-purpose, high-level programming language whose design philosophy emphasizes code readability.",
		},
	}
	for i := 0; i < 1000; i++ {
		sn := "PROJECT_" + strconv.Itoa(i)
		cat[sn] = &catalog.Project{
			ShortName: sn,
			Name: sn,
			Tags: catalog.TagSet{"junk"},
			Description: "Lorem ipsum",
		}
	}
	return cat
}

func BenchmarkTextSearchIndex(b *testing.B) {
	b.StopTimer()
	cat := newTestCatalog()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		NewTextSearch(cat)
	}
}

func BenchmarkTextSearchAnd(b *testing.B) {
	b.StopTimer()
	cat := newTestCatalog()
	searcher, _ := NewTextSearch(cat)
	ts := searcher.(*textSearch)
	q, _ := parseQuery("go go go go go go go")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ts.search(q)
	}
}

func BenchmarkTextSearchOr(b *testing.B) {
	b.StopTimer()
	cat := newTestCatalog()
	searcher, _ := NewTextSearch(cat)
	ts := searcher.(*textSearch)
	q, _ := parseQuery("PYTHON OR GO OR PYTHON OR GO OR PYTHON")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ts.search(q)
	}
}

func TestFold(t *testing.T) {
	tests := []struct {
		S      string
		Folded string
	}{
		{"", ""},
		{"A", "A"},
		{"a", "A"},
		{"Hello, World!", "HELLO, WORLD!"},
		{"hello, world!", "HELLO, WORLD!"},
	}
	for _, test := range tests {
		f := fold(test.S)
		if f != test.Folded {
			t.Errorf("fold(%q) = %q; want %q", test.S, f, test.Folded)
		}
	}
}

func BenchmarkFold(b *testing.B) {
	const n = 65536
	b.StopTimer()
	r := make([]rune, 0, n)
	for i := 0; i < n; i++ {
		r = append(r, rune(i))
	}
	s := string(r)
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		fold(s)
	}
	b.SetBytes(n)
}

type mockCatalog map[string]*catalog.Project

func (mc mockCatalog) List() ([]string, error) {
	names := make([]string, 0, len(mc))
	for sn := range mc {
		names = append(names, sn)
	}
	return names, nil
}

func (mc mockCatalog) GetProject(shortName string) (*catalog.Project, error) {
	return mc[shortName], nil
}

func (mc mockCatalog) PutProject(project *catalog.Project) error {
	if sn, _ := mc.ShortName(project.ID); sn != "" {
		delete(mc, sn)
	}
	mc[project.ShortName] = project
	return nil
}

func (mc mockCatalog) DelProject(shortName string) error {
	delete(mc, shortName)
	return nil
}

func (mc mockCatalog) ShortName(id catalog.ID) (string, error) {
	for _, p := range mc {
		if p.ID == id {
			return p.ShortName, nil
		}
	}
	return "", nil
}
