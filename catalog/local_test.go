package catalog

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"bitbucket.org/zombiezen/blackforest/vcs"
)

var magicTime = time.Date(2013, 2, 7, 10, 51, 13, 0, time.FixedZone("PST", int(-8*time.Hour/time.Second)))

const exampleProjectJSON = `{
	"id": "b11dzGs4SQid",
	"shortname": "blackforest",
	"name": "Black Forest",
	"description": "Giant Library and Distributed Organizing System",
	"catalog_time": "2013-02-07T10:51:13-08:00",
	"create_time": "2013-02-07T10:51:13-08:00",
	"tags": ["go", "http", "os", "tools"]
}` + "\n"

func newTestCatalog() (*localCatalog, *mockFilesystem, *mockWC) {
	const sep = string(filepath.Separator)

	fs := newMockFS()
	fs.Mkdir("foo")
	fs.makeFile("foo"+sep+"version.json", `{"version": 1}`)
	fs.makeFile("foo"+sep+"catalog.json", `{
	"id_to_shortname": {
		"b11dzGs4SQid": "blackforest"
	}
}`)

	fs.Mkdir("foo" + sep + "projects")
	fs.makeFile("foo"+sep+"projects"+sep+"blackforest.json", exampleProjectJSON)

	wc := &mockWC{
		renamed: make(map[string]string),
	}

	return &localCatalog{root: "foo", fs: fs, wc: wc}, fs, wc
}

func TestLocalCreate(t *testing.T) {
	const root = "foo"

	fs := newMockFS()
	cat, err := create(fs, root)
	if err != nil {
		t.Errorf("create(%q) error: %v", root, err)
	}
	if cat.root != root {
		t.Errorf("cat.root = %q; want %q", cat.root, root)
	}
	fileChecks := []struct {
		FileName string
		Content  string
	}{
		{"version.json", `{"version":1}` + "\n"},
		{"catalog.json", `{"id_to_shortname":{}}` + "\n"},
	}
	for _, fc := range fileChecks {
		name := filepath.Join(root, fc.FileName)
		if data, ok := fs.files[name]; ok && string(data) != fc.Content {
			t.Errorf("%v contents = %q; want %q", name, string(data), fc.Content)
		} else if !ok {
			t.Errorf("%q does not exist!", name)
		}
	}
	if _, ok := fs.dirs[filepath.Join(root, "projects")]; !ok {
		t.Error("projects directory does not exist")
	}
}

func TestLocalList(t *testing.T) {
	cat, _, wc := newTestCatalog()
	list, err := cat.List()
	if want := []string{"blackforest"}; !reflect.DeepEqual(list, want) {
		t.Errorf("cat.List() = %q; want %q", list, want)
	}
	if err != nil {
		t.Errorf("cat.List() error = %v", err)
	}

	if len(wc.added) != 0 {
		t.Errorf("vcs added = %v; want []", wc.added)
	}
	if len(wc.removed) != 0 {
		t.Errorf("vcs removed = %v; want []", wc.removed)
	}
	if len(wc.renamed) != 0 {
		t.Errorf("vcs renamed = %v; want {}", wc.renamed)
	}
	if wc.committed {
		t.Error("vcs committed")
	}
}

func TestLocalShortName(t *testing.T) {
	cat, _, wc := newTestCatalog()
	id := ID{0x6f, 0x5d, 0x5d, 0xcc, 0x6b, 0x38, 0x49, 0x08, 0x9d}
	sn, err := cat.ShortName(id)
	if want := "blackforest"; sn != want {
		t.Errorf("cat.ShortName(%v) = %q; want %q", id, sn, want)
	}
	if err != nil {
		t.Errorf("cat.ShortName(%v) error = %v", id, err)
	}

	if len(wc.added) != 0 {
		t.Errorf("vcs added = %v; want []", wc.added)
	}
	if len(wc.removed) != 0 {
		t.Errorf("vcs removed = %v; want []", wc.removed)
	}
	if len(wc.renamed) != 0 {
		t.Errorf("vcs renamed = %v; want {}", wc.renamed)
	}
	if wc.committed {
		t.Error("vcs committed")
	}
}

func TestLocalGetProject(t *testing.T) {
	cat, _, wc := newTestCatalog()
	proj, err := cat.GetProject("blackforest")
	want := &Project{
		ID:          ID{0x6f, 0x5d, 0x5d, 0xcc, 0x6b, 0x38, 0x49, 0x08, 0x9d},
		ShortName:   "blackforest",
		Name:        "Black Forest",
		Description: "Giant Library and Distributed Organizing System",
		CatalogTime: magicTime,
		CreateTime:  magicTime,
		Tags:        []string{"go", "http", "os", "tools"},
	}
	if !projectEqual(proj, want) {
		t.Errorf("cat.GetProject(%q) = %v; want %v", want.ShortName, proj, want)
	}
	if err != nil {
		t.Errorf("cat.GetProject(%q) error = %v", want.ShortName, err)
	}

	if len(wc.added) != 0 {
		t.Errorf("vcs added = %v; want []", wc.added)
	}
	if len(wc.removed) != 0 {
		t.Errorf("vcs removed = %v; want []", wc.removed)
	}
	if len(wc.renamed) != 0 {
		t.Errorf("vcs renamed = %v; want {}", wc.renamed)
	}
	if wc.committed {
		t.Error("vcs committed")
	}
}

func TestLocalPutProject_New(t *testing.T) {
	const root = "foo"

	id := ID{0xba, 0x7b, 0xbb, 0x6c, 0x2b, 0x66, 0x61, 0x54, 0xfb}
	cat, fs, wc := newTestCatalog()
	proj := &Project{
		ID:          id,
		ShortName:   "foo",
		Name:        "Teh Foo",
		Description: "A junk project",
		Tags:        []string{"foo", "junk"},
		Homepage:    "http://example.com/",
		CatalogTime: magicTime,
		CreateTime:  magicTime,
	}
	if err := cat.PutProject(proj); err != nil {
		t.Error("put error:", err)
	}

	fileChecks := []struct {
		FileName string
		Content  string
	}{
		{"foo.json", `{"id":"unu7bCtmYVT7","shortname":"foo","name":"Teh Foo","description":"A junk project","tags":["foo","junk"],"homepage":"http://example.com/","catalog_time":"2013-02-07T10:51:13-08:00","create_time":"2013-02-07T10:51:13-08:00"}` + "\n"},
	}
	for _, fc := range fileChecks {
		name := filepath.Join(root, "projects", fc.FileName)
		if data, ok := fs.files[name]; ok && string(data) != fc.Content {
			t.Errorf("%v contents = %q; want %q", name, string(data), fc.Content)
		} else if !ok {
			t.Errorf("%q does not exist!", name)
		}
	}

	sn, err := cat.ShortName(id)
	if want := "foo"; sn != want {
		t.Errorf("cat.ShortName(%v) = %q; want %q", id, sn, want)
	}
	if err != nil {
		t.Errorf("cat.ShortName(%v) error = %v", id, err)
	}

	if want := []string{"projects/foo.json"}; !reflect.DeepEqual(wc.added, want) {
		t.Errorf("vcs added = %v; want %v", wc.added, want)
	}
	if len(wc.removed) != 0 {
		t.Errorf("vcs removed = %v; want []", wc.removed)
	}
	if len(wc.renamed) != 0 {
		t.Errorf("vcs renamed = %v; want {}", wc.renamed)
	}
	if !wc.committed {
		t.Error("vcs not committed")
	}
}

func TestLocalPutProject_Update(t *testing.T) {
	const root = "foo"

	id := ID{0x6f, 0x5d, 0x5d, 0xcc, 0x6b, 0x38, 0x49, 0x08, 0x9d}
	cat, fs, wc := newTestCatalog()
	proj := &Project{
		ID:          id,
		ShortName:   "blackforest",
		Name:        "Teh Foo",
		Description: "A junk project",
		Tags:        []string{"foo", "junk"},
		Homepage:    "http://example.com/",
		CatalogTime: magicTime,
		CreateTime:  magicTime,
	}
	if err := cat.PutProject(proj); err != nil {
		t.Error("put error:", err)
	}

	fileChecks := []struct {
		FileName string
		Content  string
	}{
		{"blackforest.json", `{"id":"b11dzGs4SQid","shortname":"blackforest","name":"Teh Foo","description":"A junk project","tags":["foo","junk"],"homepage":"http://example.com/","catalog_time":"2013-02-07T10:51:13-08:00","create_time":"2013-02-07T10:51:13-08:00"}` + "\n"},
	}
	for _, fc := range fileChecks {
		name := filepath.Join(root, "projects", fc.FileName)
		if data, ok := fs.files[name]; ok && string(data) != fc.Content {
			t.Errorf("%v contents = %q; want %q", name, string(data), fc.Content)
		} else if !ok {
			t.Errorf("%q does not exist!", name)
		}
	}

	sn, err := cat.ShortName(id)
	if want := "blackforest"; sn != want {
		t.Errorf("cat.ShortName(%v) = %q; want %q", id, sn, want)
	}
	if err != nil {
		t.Errorf("cat.ShortName(%v) error = %v", id, err)
	}

	if len(wc.added) != 0 {
		t.Errorf("vcs added = %v; want []", wc.added)
	}
	if len(wc.removed) != 0 {
		t.Errorf("vcs removed = %v; want []", wc.removed)
	}
	if len(wc.renamed) != 0 {
		t.Errorf("vcs renamed = %v; want {}", wc.renamed)
	}
	if !wc.committed {
		t.Error("vcs not committed")
	}
}

func TestLocalPutProject_Rename(t *testing.T) {
	const root = "foo"

	id := ID{0x6f, 0x5d, 0x5d, 0xcc, 0x6b, 0x38, 0x49, 0x08, 0x9d}
	cat, fs, wc := newTestCatalog()
	proj := &Project{
		ID:          id,
		ShortName:   "foo",
		Name:        "Teh Foo",
		Description: "A junk project",
		Tags:        []string{"foo", "junk"},
		Homepage:    "http://example.com/",
		CatalogTime: magicTime,
		CreateTime:  magicTime,
	}
	if err := cat.PutProject(proj); err != nil {
		t.Error("put error:", err)
	}

	fileChecks := []struct {
		FileName string
		Content  string
	}{
		{"foo.json", `{"id":"b11dzGs4SQid","shortname":"foo","name":"Teh Foo","description":"A junk project","tags":["foo","junk"],"homepage":"http://example.com/","catalog_time":"2013-02-07T10:51:13-08:00","create_time":"2013-02-07T10:51:13-08:00"}` + "\n"},
	}
	for _, fc := range fileChecks {
		name := filepath.Join(root, "projects", fc.FileName)
		if data, ok := fs.files[name]; ok && string(data) != fc.Content {
			t.Errorf("%v contents = %q; want %q", name, string(data), fc.Content)
		} else if !ok {
			t.Errorf("%q does not exist!", name)
		}
	}

	if _, ok := fs.files[filepath.Join(root, "projects", "blackforest.json")]; ok {
		t.Error("blackforest.json still exists")
	}

	sn, err := cat.ShortName(id)
	if want := "foo"; sn != want {
		t.Errorf("cat.ShortName(%v) = %q; want %q", id, sn, want)
	}
	if err != nil {
		t.Errorf("cat.ShortName(%v) error = %v", id, err)
	}

	if want := []string{"projects/foo.json"}; !reflect.DeepEqual(wc.added, want) {
		t.Errorf("vcs added = %v; want %v", wc.added, want)
	}
	if want := []string{"projects/blackforest.json"}; !reflect.DeepEqual(wc.removed, want) {
		t.Errorf("vcs removed = %v; want %v", wc.removed, want)
	}
	if want := map[string]string{"projects/blackforest.json": "projects/foo.json"}; !reflect.DeepEqual(wc.renamed, want) {
		t.Errorf("vcs renamed = %v; want %v", wc.renamed, want)
	}
	if !wc.committed {
		t.Error("vcs not committed")
	}
}

func TestLocalPutProject_NameConflict(t *testing.T) {
	const root = "foo"

	id := ID{0x6f, 0x5d, 0x5d, 0xdd, 0x6b, 0x38, 0x49, 0x08, 0x9d}
	cat, fs, wc := newTestCatalog()
	proj := &Project{
		ID:          id,
		ShortName:   "blackforest",
		Name:        "Teh Foo",
		Description: "A junk project",
		Tags:        []string{"foo", "junk"},
		Homepage:    "http://example.com/",
	}
	if err := cat.PutProject(proj); err == nil {
		t.Error("expected put error, got nil")
	}

	fileChecks := []struct {
		FileName string
		Content  string
	}{
		{"blackforest.json", exampleProjectJSON},
	}
	for _, fc := range fileChecks {
		name := filepath.Join(root, "projects", fc.FileName)
		if data, ok := fs.files[name]; ok && string(data) != fc.Content {
			t.Errorf("%v contents = %q; want %q", name, string(data), fc.Content)
		} else if !ok {
			t.Errorf("%q does not exist!", name)
		}
	}

	sn, err := cat.ShortName(ID{0x6f, 0x5d, 0x5d, 0xcc, 0x6b, 0x38, 0x49, 0x08, 0x9d})
	if want := "blackforest"; sn != want {
		t.Errorf("cat.ShortName(%v) = %q; want %q", id, sn, want)
	}
	if err != nil {
		t.Errorf("cat.ShortName(%v) error = %v", id, err)
	}

	if len(wc.added) != 0 {
		t.Errorf("vcs added = %v; want []", wc.added)
	}
	if len(wc.removed) != 0 {
		t.Errorf("vcs removed = %v; want []", wc.removed)
	}
	if len(wc.renamed) != 0 {
		t.Errorf("vcs renamed = %v; want {}", wc.renamed)
	}
	if wc.committed {
		t.Error("vcs committed")
	}
}

func projectEqual(a, b *Project) bool {
	return a.ID == b.ID &&
		a.ShortName == b.ShortName &&
		a.Name == b.Name &&
		a.Description == b.Description &&
		reflect.DeepEqual(a.Tags, b.Tags) &&
		a.Homepage == b.Homepage &&
		a.CatalogTime.Equal(b.CatalogTime) &&
		a.CreateTime.Equal(b.CreateTime) &&
		reflect.DeepEqual(a.VCS, b.VCS) &&
		reflect.DeepEqual(a.PerHost, b.PerHost)
}

type mockWC struct {
	added     []string
	removed   []string
	renamed   map[string]string
	committed bool
}

func (wc *mockWC) VCS() vcs.VCS {
	return nil
}

func (wc *mockWC) Path() string {
	return "foo"
}

func (wc *mockWC) Current() (vcs.Rev, error) {
	return nil, errors.New("mocked")
}

func (wc *mockWC) Update(rev vcs.Rev) error {
	return errors.New("mocked")
}

func (wc *mockWC) ParseRev(s string) (vcs.Rev, error) {
	return nil, errors.New("mocked")
}

func (wc *mockWC) Add(paths []string) error {
	wc.added = append(wc.added, paths...)
	return nil
}

func (wc *mockWC) Remove(paths []string) error {
	wc.removed = append(wc.removed, paths...)
	return nil
}

func (wc *mockWC) Rename(src, dst string) error {
	wc.renamed[src] = dst
	return nil
}

func (wc *mockWC) Commit(message string, files []string) error {
	wc.committed = true
	return nil
}
