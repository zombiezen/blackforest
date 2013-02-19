package catalog

import (
	"path/filepath"
	"reflect"
	"testing"
)

func newTestCatalog() (*localCatalog, *mockFilesystem) {
	const sep = string(filepath.Separator)

	fs := newMockFS()
	fs.Mkdir("foo")
	fs.makeFile("foo"+sep+"version.json", `{"version": 1}`)
	fs.makeFile("foo"+sep+"catalog.json", `{
	"id_to_shortname": {
		"b11dzGs4SQid": "glados"
	}
}`)

	fs.Mkdir("foo" + sep + "projects")
	fs.makeFile(
		"foo"+sep+"projects"+sep+"glados.json",
		`{
	"id": "b11dzGs4SQid",
	"shortname": "glados",
	"name": "GLaDOS",
	"description": "Giant Library and Distributed Organizing System",
	"tags": ["go", "http", "os", "tools"]
}`)

	return &localCatalog{root: "foo", fs: fs}, fs
}

func TestLocalList(t *testing.T) {
	cat, _ := newTestCatalog()
	list, err := cat.List()
	if want := []string{"glados"}; !reflect.DeepEqual(list, want) {
		t.Errorf("cat.List() = %q; want %q", list, want)
	}
	if err != nil {
		t.Errorf("cat.List() error = %v", err)
	}
}

func TestLocalShortName(t *testing.T) {
	cat, _ := newTestCatalog()
	sn, err := cat.ShortName(ID{0x6f, 0x5d, 0x5d, 0xcc, 0x6b, 0x38, 0x49, 0x08, 0x9d})
	if want := "glados"; sn != want {
		t.Errorf("cat.ShortName() = %q; want %q", sn, want)
	}
	if err != nil {
		t.Errorf("cat.ShortName() error = %v", err)
	}
}
