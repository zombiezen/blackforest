package catalog

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

// A localCatalog is a catalog that uses the filesystem as storage.
type localCatalog struct {
	root string
	fs   filesystem
}

// Create creates a new catalog at the given directory.
func Create(root string) (Catalog, error) {
	return create(realFilesystem{}, root)
}

func create(fs filesystem, root string) (*localCatalog, error) {
	// Create root directory (must not exist)
	if err := fs.Mkdir(root); err != nil {
		return nil, err
	}

	// Lock catalog
	cat := &localCatalog{root: root, fs: fs}
	err := cat.doChange(func() error {
		// Create projects directory
		if err := fs.Mkdir(filepath.Join(root, projectsDir)); err != nil {
			return err
		}

		// Create catalog
		meta := &catalogMeta{
			ShortNameMap: map[string]string{},
		}
		if err := writeJSON(fs, filepath.Join(root, catalogFile), meta, true); err != nil {
			return err
		}

		// Write out version file
		var v struct {
			Version int `json:"version"`
		}
		v.Version = 1
		if err := writeJSON(fs, filepath.Join(root, versionFile), &v, true); err != nil {
			return err
		}

		return nil
	})
	return cat, err
}

// Open opens the catalog in a directory.
func Open(root string) (Catalog, error) {
	fs := realFilesystem{}
	var v struct {
		Version int `json:"version"`
	}
	if err := readJSON(fs, filepath.Join(root, versionFile), &v); err != nil {
		return nil, err
	}
	if v.Version != 1 {
		return nil, VersionError(v.Version)
	}
	return &localCatalog{root: root, fs: fs}, nil
}

func (cat *localCatalog) List() ([]string, error) {
	dir, err := cat.fs.Open(filepath.Join(cat.root, projectsDir))
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	const dirBatchSize = 100
	var names []string
	for {
		entries, err := dir.Readdir(dirBatchSize)
		for _, ent := range entries {
			name, mt := ent.Name(), ent.Mode()&os.ModeType
			if strings.HasSuffix(name, jsonExt) && (mt == 0 || mt == os.ModeSymlink) {
				names = append(names, name[:len(name)-len(jsonExt)])
			}
		}
		if err == io.EOF {
			break
		} else if err != nil {
			return names, err
		}
	}
	return names, nil
}

func (cat *localCatalog) GetProject(shortName string) (*Project, error) {
	const op = "get"

	if !isValidShortName(shortName) {
		return nil, shortNameError(shortName)
	}
	proj := new(Project)
	path := filepath.Join(cat.root, projectsDir, shortName+jsonExt)
	if err := readJSON(cat.fs, path, proj); err != nil {
		return proj, &projectError{ShortName: shortName, Op: op, Err: err}
	}
	return proj, nil
}

func (cat *localCatalog) PutProject(project *Project) (retErr error) {
	const op = "put"

	id, sn := project.ID, project.ShortName
	idString := id.String()
	if !isValidShortName(sn) {
		return shortNameError(sn)
	}
	return cat.doChange(func() error {
		var old string
		err := cat.rewriteCatalog(func(c *catalogMeta) error {
			old, c.ShortNameMap[idString] = c.ShortNameMap[idString], sn
			if err := writeJSON(cat.fs, cat.projectPath(sn), project, old != project.ShortName); err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			return &projectError{ShortName: sn, Op: op, Err: err}
		}

		// Delete old file (if necessary)
		if old != "" && old != project.ShortName {
			if err := cat.fs.Remove(cat.projectPath(old)); err != nil {
				return &projectError{ShortName: sn, Op: op, Err: err}
			}
		}

		return nil
	})
}

func (cat *localCatalog) DelProject(shortName string) error {
	const op = "del"

	if !isValidShortName(shortName) {
		return shortNameError(shortName)
	}
	return cat.doChange(func() error {
		// Rewrite catalog
		err := cat.rewriteCatalog(func(c *catalogMeta) error {
			m := c.ShortNameMap
			for id, name := range m {
				if name == shortName {
					delete(m, id)
				}
			}
			return nil
		})
		if err != nil {
			return &projectError{ShortName: shortName, Op: op, Err: err}
		}

		// Delete file
		if err := cat.fs.Remove(cat.projectPath(shortName)); err != nil {
			return &projectError{ShortName: shortName, Op: op, Err: err}
		}

		return nil
	})
}

func (cat *localCatalog) projectPath(shortName string) string {
	return filepath.Join(cat.root, projectsDir, shortName+jsonExt)
}

func (cat *localCatalog) ShortName(id ID) (string, error) {
	var c struct {
		Map map[string]string `json:"id_to_shortname"`
	}
	if err := readJSON(cat.fs, filepath.Join(cat.root, catalogFile), &c); err != nil {
		return "", err
	}
	return c.Map[id.String()], nil
}

// doChange locks the catalog, calls f, and then unlocks the catalog.  Any error returned by f is passed through.  It is the responsibility of the function called to roll back any change on failure, if desired.
func (cat *localCatalog) doChange(f func() error) error {
	if err := cat.lock(); err != nil {
		return err
	}
	ferr := f()
	if err := cat.unlock(); err != nil && ferr == nil {
		return err
	}
	return ferr
}

// rewriteCatalog calls f with the unmarshalled contents of catalog.json and writes any changes.
// This method does not lock the catalog; it should be used inside of a doChange.  If f returns an error, catalog.json will not be rewritten.
func (cat *localCatalog) rewriteCatalog(f func(*catalogMeta) error) error {
	var c catalogMeta
	p := filepath.Join(cat.root, catalogFile)
	if err := readJSON(cat.fs, p, &c); err != nil {
		return err
	}
	if err := f(&c); err != nil {
		return err
	}
	if err := writeJSON(cat.fs, p, &c, false); err != nil {
		return err
	}
	return nil
}

func (cat *localCatalog) lockPath() string {
	return filepath.Join(cat.root, lockFile)
}

func (cat *localCatalog) lock() error {
	p := cat.lockPath()
	f, err := cat.fs.Create(p, true)
	if err != nil {
		if cat.fs.IsExist(err) {
			return ErrLocked
		}
		return err
	}
	err = f.Close()
	if err != nil {
		// Error ignored.
		// If the remove fails, the lock is already in a weird state, so there's not too much we can do.
		cat.fs.Remove(p)
	}
	return err
}

func (cat *localCatalog) unlock() error {
	return cat.fs.Remove(cat.lockPath())
}

// A catalogMeta holds the schema for a catalog.json file.
type catalogMeta struct {
	ShortNameMap map[string]string `json:"id_to_shortname"`
}

// Catalog paths
const (
	versionFile = "version.json"
	catalogFile = "catalog.json"
	lockFile    = "catalog.lock"

	projectsDir = "projects"

	jsonExt = ".json"
)
