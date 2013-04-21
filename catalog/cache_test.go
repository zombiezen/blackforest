package catalog

import (
	"errors"
	"reflect"
	"testing"
)

func newMockCatalog() mockCatalog {
	return mockCatalog{
		"blackforest": &Project{
			ID:          ID{0x6f, 0x5d, 0x5d, 0xcc, 0x6b, 0x38, 0x49, 0x08, 0x9d},
			ShortName:   "blackforest",
			Name:        "Black Forest",
			Description: "Giant Library and Distributed Organizing System",
			CatalogTime: magicTime,
			CreateTime:  magicTime,
			Tags:        []string{"go", "http", "os", "tools"},
		},
	}
}

func TestCacheAccess(t *testing.T) {
	magicID := ID{0x6f, 0x5d, 0x5d, 0xcc, 0x6b, 0x38, 0x49, 0x08, 0x9d}
	mc := newMockCatalog()
	c, err := NewCache(mc)
	if err != nil {
		t.Error("NewCache error:", err)
	}

	list, err := c.List()
	if want := []string{"blackforest"}; !reflect.DeepEqual(list, want) {
		t.Errorf("Cache.List() = %q; want %q", list, want)
	}
	if err != nil {
		t.Error("Cache.List() error:", err)
	}

	sn, err := c.ShortName(magicID)
	if sn != "blackforest" {
		t.Errorf("Cache.ShortName(%v) = %q; want %q", magicID, sn, "blackforest")
	}
	if err != nil {
		t.Errorf("Cache.ShortName(%v) error: %v", magicID, err)
	}

	p, err := c.GetProject("blackforest")
	want := &Project{
		ID:          magicID,
		ShortName:   "blackforest",
		Name:        "Black Forest",
		Description: "Giant Library and Distributed Organizing System",
		CatalogTime: magicTime,
		CreateTime:  magicTime,
		Tags:        []string{"go", "http", "os", "tools"},
	}
	if !projectEqual(p, want) {
		t.Errorf("Cache.GetProject(%q) = %v; want %v", "blackforest", p, want)
	}
	if err != nil {
		t.Errorf("Cache.GetProject(%q) error: %v", "blackforest", err)
	}

	if tags, want := newStringSet(c.Tags()), newStringSet([]string{"go", "http", "os", "tools"}); !reflect.DeepEqual(tags, want) {
		t.Errorf("Cache.Tags() = %v; want %v", tags.Slice(), want.Slice())
	}
	if names := c.FindTag("go"); !reflect.DeepEqual(names, []string{"blackforest"}) {
		t.Errorf("Cache.FindTag(%q) = %v; want %v", "go", names, []string{"blackforest"})
	}
}

func TestCachePut(t *testing.T) {
	magicID := ID{0x6f, 0x5d, 0x5d, 0xcc, 0x6b, 0x38, 0x49, 0x08, 0x9d}
	mc := newMockCatalog()
	c, err := NewCache(mc)
	if err != nil {
		t.Error("NewCache error:", err)
	}

	proj := &Project{
		ID:          magicID,
		ShortName:   "blackforesta",
		Name:        "Black Forest",
		Description: "Giant Library and Distributed Organizing System",
		CatalogTime: magicTime,
		CreateTime:  magicTime,
		Tags:        []string{"go", "web", "os", "tools"},
	}

	if err := c.PutProject(proj); err != nil {
		t.Errorf("Cache.PutProject(%v) error: %v", proj, err)
	}

	p, err := c.GetProject("blackforesta")
	// to ensure the pointer isn't modified
	want := &Project{
		ID:          magicID,
		ShortName:   "blackforesta",
		Name:        "Black Forest",
		Description: "Giant Library and Distributed Organizing System",
		CatalogTime: magicTime,
		CreateTime:  magicTime,
		Tags:        []string{"go", "web", "os", "tools"},
	}
	if !projectEqual(p, want) {
		t.Errorf("Cache.GetProject(%q) = %v; want %v", "blackforesta", p, want)
	}
	if err != nil {
		t.Errorf("Cache.GetProject(%q) error: %v", "blackforesta", err)
	}
	if mcProj := mc["blackforesta"]; !projectEqual(mcProj, want) {
		t.Errorf("Cache.cat[%q] = %v; want %v", "blackforesta", mcProj, want)
	}
	if mcProj := mc["blackforest"]; mcProj != nil {
		t.Errorf("Cache.cat[%q] = %v; want nil", "blackforest", mcProj)
	}

	p, err = c.GetProject("blackforest")
	if p != nil {
		t.Errorf("Cache.GetProject(%q) = %v; want nil", "blackforest", p)
	}
	if err != nil {
		t.Errorf("Cache.GetProject(%q) error: %v", "blackforest", err)
	}

	sn, err := c.ShortName(magicID)
	if sn != "blackforesta" {
		t.Errorf("Cache.ShortName(%v) = %q; want %q", magicID, sn, "blackforesta")
	}
	if err != nil {
		t.Errorf("Cache.ShortName(%v) error: %v", magicID, err)
	}

	if tags, want := newStringSet(c.Tags()), newStringSet([]string{"go", "web", "os", "tools"}); !reflect.DeepEqual(tags, want) {
		t.Errorf("Cache.Tags() = %v; want %v", tags.Slice(), want.Slice())
	}
	if names := c.FindTag("web"); !reflect.DeepEqual(names, []string{"blackforesta"}) {
		t.Errorf("Cache.FindTag(%q) = %v; want %v", "web", names, []string{"blackforesta"})
	}
	if names := c.FindTag("http"); len(names) != 0 {
		t.Errorf("Cache.FindTag(%q) = %v; want %v", "http", names, []string{})
	}
}

func TestCache_RefreshProject(t *testing.T) {
	const (
		projShortName = "blackforest"
		projInitName  = "Black Forest"
		projNewName   = "FOO"
	)

	mc := newMockCatalog()
	c, err := NewCache(mc)
	if err != nil {
		t.Error("NewCache error:", err)
	}

	steps := []struct {
		Desc       string
		Func       func()
		Check      func() (*Project, error)
		ExpectName string
	}{
		{
			`initial GetProject("` + projShortName + `")`,
			nil,
			func() (*Project, error) {
				return c.GetProject(projShortName)
			},
			projInitName,
		},
		{
			`before refresh GetProject("` + projShortName + `")`,
			func() {
				mc[projShortName].Name = projNewName
			},
			func() (*Project, error) {
				return c.GetProject(projShortName)
			},
			projInitName,
		},
		{
			`RefreshProject("` + projShortName + `")`,
			nil,
			func() (*Project, error) {
				return c.RefreshProject(projShortName)
			},
			projNewName,
		},
		{
			`after refresh GetProject("` + projShortName + `")`,
			nil,
			func() (*Project, error) {
				return c.GetProject(projShortName)
			},
			projNewName,
		},
	}

	for _, step := range steps {
		if step.Func != nil {
			step.Func()
		}
		proj, err := step.Check()
		if err != nil {
			t.Errorf("%s error: %v", step.Desc, err)
		}
		if proj == nil {
			t.Errorf("%s = nil", step.Desc)
		} else if proj.Name != step.ExpectName {
			t.Errorf("%s.Name = %q; want %q", step.Desc, proj.Name, step.ExpectName)
		}
	}
}

func TestCache_RefreshProject_Delete(t *testing.T) {
	const (
		projShortName = "blackforest"
		projName      = "Black Forest"
	)

	mc := newMockCatalog()
	c, err := NewCache(mc)
	if err != nil {
		t.Error("NewCache error:", err)
	}

	steps := []struct {
		Desc       string
		Func       func()
		Check      func() (*Project, error)
		ExpectNil  bool
		ExpectName string
	}{
		{
			Desc: `initial GetProject("` + projShortName + `")`,
			Check: func() (*Project, error) {
				return c.GetProject(projShortName)
			},
			ExpectName: projName,
		},
		{
			Func: func() {
				delete(mc, projShortName)
			},
			Desc: `before refresh GetProject("` + projShortName + `")`,
			Check: func() (*Project, error) {
				return c.GetProject(projShortName)
			},
			ExpectName: projName,
		},
		{
			Desc: `RefreshProject("` + projShortName + `")`,
			Check: func() (*Project, error) {
				return c.RefreshProject(projShortName)
			},
			ExpectNil: true,
		},
		{
			Desc: `after refresh GetProject("` + projShortName + `")`,
			Check: func() (*Project, error) {
				return c.GetProject(projShortName)
			},
			ExpectNil: true,
		},
	}

	for _, step := range steps {
		if step.Func != nil {
			step.Func()
		}
		proj, err := step.Check()
		if err != nil {
			t.Errorf("%s error: %v", step.Desc, err)
		}
		switch {
		case step.ExpectNil && proj != nil:
			t.Errorf("%s = %#v; want nil", step.Desc, proj)
		case !step.ExpectNil && proj == nil:
			t.Errorf("%s = nil", step.Desc)
		case !step.ExpectNil && proj.Name != step.ExpectName:
			t.Errorf("%s.Name = %q; want %q", step.Desc, proj.Name, step.ExpectName)
		}
	}
}

func TestCache_RefreshProject_Rename(t *testing.T) {
	const (
		projShortName    = "blackforest"
		projNewShortName = "blackforest2"

		projInitName = "Black Forest"
		projNewName  = "Black Forest 2"
	)

	mc := newMockCatalog()
	c, err := NewCache(mc)
	if err != nil {
		t.Error("NewCache error:", err)
	}

	steps := []struct {
		Desc       string
		Func       func()
		Check      func() (*Project, error)
		ExpectNil  bool
		ExpectName string
	}{
		{
			Desc: `initial GetProject("` + projShortName + `")`,
			Check: func() (*Project, error) {
				return c.GetProject(projShortName)
			},
			ExpectName: projInitName,
		},
		{
			Desc: `initial GetProject("` + projNewShortName + `")`,
			Check: func() (*Project, error) {
				return c.GetProject(projNewShortName)
			},
			ExpectNil: true,
		},
		{
			Func: func() {
				p := mc[projShortName]
				delete(mc, projShortName)
				mc[projNewShortName] = p
				p.Name = projNewName
				p.ShortName = projNewShortName
			},
			Desc: `before refresh GetProject("` + projShortName + `")`,
			Check: func() (*Project, error) {
				return c.GetProject(projShortName)
			},
			ExpectName: projInitName,
		},
		{
			Desc: `before refresh GetProject("` + projNewShortName + `")`,
			Check: func() (*Project, error) {
				return c.GetProject(projNewShortName)
			},
			ExpectNil: true,
		},
		{
			Desc: `RefreshProject("` + projNewShortName + `")`,
			Check: func() (*Project, error) {
				return c.RefreshProject(projNewShortName)
			},
			ExpectName: projNewName,
		},
		{
			Desc: `after refresh GetProject("` + projShortName + `")`,
			Check: func() (*Project, error) {
				return c.GetProject(projShortName)
			},
			ExpectNil: true,
		},
		{
			Desc: `after refresh GetProject("` + projNewShortName + `")`,
			Check: func() (*Project, error) {
				return c.GetProject(projNewShortName)
			},
			ExpectName: projNewName,
		},
	}

	for _, step := range steps {
		if step.Func != nil {
			step.Func()
		}
		proj, err := step.Check()
		if err != nil {
			t.Errorf("%s error: %v", step.Desc, err)
		}
		switch {
		case step.ExpectNil && proj != nil:
			t.Errorf("%s = %#v; want nil", step.Desc, proj)
		case !step.ExpectNil && proj == nil:
			t.Errorf("%s = nil", step.Desc)
		case !step.ExpectNil && proj.Name != step.ExpectName:
			t.Errorf("%s.Name = %q; want %q", step.Desc, proj.Name, step.ExpectName)
		}
	}
}

func TestCache_RefreshProject_Fail(t *testing.T) {
	const (
		projShortName = "blackforest"
		projName      = "Black Forest"
	)

	mc := mockFailCatalog{mockCatalog: newMockCatalog()}
	c, err := NewCache(&mc)
	if err != nil {
		t.Error("NewCache error:", err)
	}

	steps := []struct {
		Desc        string
		Func        func()
		Check       func() (*Project, error)
		ExpectError bool
		ExpectName  string
	}{
		{
			Desc: `initial GetProject("` + projShortName + `")`,
			Check: func() (*Project, error) {
				return c.GetProject(projShortName)
			},
			ExpectName: projName,
		},
		{
			Func: func() {
				mc.Fail = true
			},
			Desc: `before refresh GetProject("` + projShortName + `")`,
			Check: func() (*Project, error) {
				return c.GetProject(projShortName)
			},
			ExpectName: projName,
		},
		{
			Desc: `RefreshProject("` + projShortName + `")`,
			Check: func() (*Project, error) {
				return c.RefreshProject(projShortName)
			},
			ExpectError: true,
			ExpectName:  projName,
		},
		{
			Desc: `after refresh GetProject("` + projShortName + `")`,
			Check: func() (*Project, error) {
				return c.GetProject(projShortName)
			},
			ExpectName: projName,
		},
	}

	for _, step := range steps {
		if step.Func != nil {
			step.Func()
		}
		proj, err := step.Check()
		if !step.ExpectError && err != nil {
			t.Errorf("%s error: %v", step.Desc, err)
		} else if step.ExpectError && err == nil {
			t.Errorf("%s expected error", step.Desc)
		}
		switch {
		case proj == nil:
			t.Errorf("%s = nil", step.Desc)
		case proj.Name != step.ExpectName:
			t.Errorf("%s.Name = %q; want %q", step.Desc, proj.Name, step.ExpectName)
		}
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

var errMockCatalogFail = errors.New("catalog: time to fail")

type mockFailCatalog struct {
	mockCatalog
	Fail bool
}

func (mc *mockFailCatalog) List() ([]string, error) {
	if mc.Fail {
		return nil, errMockCatalogFail
	}
	return mc.mockCatalog.List()
}

func (mc *mockFailCatalog) GetProject(shortName string) (*Project, error) {
	if mc.Fail {
		return nil, errMockCatalogFail
	}
	return mc.mockCatalog.GetProject(shortName)
}

func (mc *mockFailCatalog) PutProject(project *Project) error {
	if mc.Fail {
		return errMockCatalogFail
	}
	return mc.mockCatalog.PutProject(project)
}

func (mc *mockFailCatalog) DelProject(shortName string) error {
	if mc.Fail {
		return errMockCatalogFail
	}
	return mc.mockCatalog.DelProject(shortName)
}

func (mc *mockFailCatalog) ShortName(id ID) (string, error) {
	if mc.Fail {
		return "", errMockCatalogFail
	}
	return mc.mockCatalog.ShortName(id)
}
