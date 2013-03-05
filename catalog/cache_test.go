package catalog

import (
	"reflect"
	"testing"
)

func TestCacheAccess(t *testing.T) {
	magicID := ID{0x6f, 0x5d, 0x5d, 0xcc, 0x6b, 0x38, 0x49, 0x08, 0x9d}
	mc := mockCatalog{
		"glados": &Project{
			ID:          magicID,
			ShortName:   "glados",
			Name:        "GLaDOS",
			Description: "Giant Library and Distributed Organizing System",
			CatalogTime: magicTime,
			CreateTime:  magicTime,
			Tags:        []string{"go", "http", "os", "tools"},
		},
	}
	c, err := NewCache(mc)
	if err != nil {
		t.Error("NewCache error:", err)
	}

	list, err := c.List()
	if want := []string{"glados"}; !reflect.DeepEqual(list, want) {
		t.Errorf("Cache.List() = %q; want %q", list, want)
	}
	if err != nil {
		t.Error("Cache.List() error:", err)
	}

	sn, err := c.ShortName(magicID)
	if sn != "glados" {
		t.Errorf("Cache.ShortName(%q) = %q; want %q", sn, "glados")
	}
	if err != nil {
		t.Errorf("Cache.ShortName(%v) error: %v", magicID, err)
	}

	p, err := c.GetProject("glados")
	want := &Project{
		ID:          magicID,
		ShortName:   "glados",
		Name:        "GLaDOS",
		Description: "Giant Library and Distributed Organizing System",
		CatalogTime: magicTime,
		CreateTime:  magicTime,
		Tags:        []string{"go", "http", "os", "tools"},
	}
	if !projectEqual(p, want) {
		t.Errorf("Cache.GetProject(%q) = %v; want %v", "glados", p, want)
	}
	if err != nil {
		t.Errorf("Cache.GetProject(%q) error: %v", "glados", err)
	}
}

func TestCachePut(t *testing.T) {
	magicID := ID{0x6f, 0x5d, 0x5d, 0xcc, 0x6b, 0x38, 0x49, 0x08, 0x9d}
	mc := mockCatalog{
		"glados": &Project{
			ID:          magicID,
			ShortName:   "glados",
			Name:        "GLaDOS",
			Description: "Giant Library and Distributed Organizing System",
			CatalogTime: magicTime,
			CreateTime:  magicTime,
			Tags:        []string{"go", "http", "os", "tools"},
		},
	}
	c, err := NewCache(mc)
	if err != nil {
		t.Error("NewCache error:", err)
	}

	proj := &Project{
		ID:          magicID,
		ShortName:   "gladosa",
		Name:        "GLaDOS",
		Description: "Giant Library and Distributed Organizing System",
		CatalogTime: magicTime,
		CreateTime:  magicTime,
		Tags:        []string{"go", "http", "os", "tools"},
	}

	if err := c.PutProject(proj); err != nil {
		t.Errorf("Cache.PutProject(%v) error: %v", proj, err)
	}

	p, err := c.GetProject("gladosa")
	// to ensure the pointer isn't modified
	want := &Project{
		ID:          magicID,
		ShortName:   "gladosa",
		Name:        "GLaDOS",
		Description: "Giant Library and Distributed Organizing System",
		CatalogTime: magicTime,
		CreateTime:  magicTime,
		Tags:        []string{"go", "http", "os", "tools"},
	}
	if !projectEqual(p, want) {
		t.Errorf("Cache.GetProject(%q) = %v; want %v", "gladosa", p, want)
	}
	if err != nil {
		t.Errorf("Cache.GetProject(%q) error: %v", "gladosa", err)
	}
	if mcProj := mc["gladosa"]; !projectEqual(mcProj, want) {
		t.Errorf("Cache.cat[%q] = %v; want %v", "gladosa", mcProj, want)
	}
	if mcProj := mc["glados"]; mcProj != nil {
		t.Errorf("Cache.cat[%q] = %v; want nil", "glados", mcProj)
	}

	p, err = c.GetProject("glados")
	if p != nil {
		t.Errorf("Cache.GetProject(%q) = %v; want nil", "glados", p)
	}
	if err != nil {
		t.Errorf("Cache.GetProject(%q) error: %v", "glados", err)
	}

	sn, err := c.ShortName(magicID)
	if sn != "gladosa" {
		t.Errorf("Cache.ShortName(%v) = %q; want %q", magicID, sn, "gladosa")
	}
	if err != nil {
		t.Errorf("Cache.ShortName(%v) error: %v", magicID, err)
	}
}

type mockCatalog map[string]*Project

func (mc mockCatalog) List() ([]string, error) {
	names := make([]string, 0, len(mc))
	for sn := range mc {
		names = append(names, sn)
	}
	return names, nil
}

func (mc mockCatalog) GetProject(shortName string) (*Project, error) {
	return mc[shortName], nil
}

func (mc mockCatalog) PutProject(project *Project) error {
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

func (mc mockCatalog) ShortName(id ID) (string, error) {
	for _, p := range mc {
		if p.ID == id {
			return p.ShortName, nil
		}
	}
	return "", nil
}
