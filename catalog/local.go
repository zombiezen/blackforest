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
}

// Open opens the catalog in a directory.
func Open(root string) (Catalog, error) {
	var v struct {
		Version int `json:"version"`
	}
	if err := readJSON(filepath.Join(root, versionFile), &v); err != nil {
		return nil, err
	}
	if v.Version != 1 {
		return nil, VersionError(v.Version)
	}
	return &localCatalog{root: root}, nil
}

func (cat *localCatalog) List() ([]string, error) {
	dir, err := os.Open(filepath.Join(cat.root, projectsDir))
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
	if err := readJSON(path, proj); err != nil {
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
		// Write project file
		if err := writeJSON(cat.projectPath(sn), project, true); err != nil {
			return &projectError{ShortName: sn, Op: op, Err: err}
		}

		// Rewrite catalog
		var old string
		err := cat.rewriteCatalog(func(c *catalogMeta) error {
			old, c.ShortNameMap[idString] = c.ShortNameMap[idString], sn
			return nil
		})
		if err != nil {
			return &projectError{ShortName: sn, Op: op, Err: err}
		}

		// Delete old file (if necessary)
		if old != "" {
			if err := os.Remove(cat.projectPath(old)); err != nil {
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
		if err := os.Remove(cat.projectPath(shortName)); err != nil {
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
	if err := readJSON(filepath.Join(cat.root, catalogFile), &c); err != nil {
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
	if err := readJSON(p, &c); err != nil {
		return err
	}
	if err := f(&c); err != nil {
		return err
	}
	if err := writeJSON(p, &c, false); err != nil {
		return err
	}
	return nil
}

func (cat *localCatalog) lockPath() string {
	return filepath.Join(cat.root, lockFile)
}

func (cat *localCatalog) lock() error {
	p := cat.lockPath()
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_TRUNC, 0666)
	if err != nil {
		if os.IsExist(err) {
			return ErrLocked
		}
		return err
	}
	err = f.Close()
	if err != nil {
		// Error ignored.
		// If the remove fails, the lock is already in a weird state, so there's not too much we can do.
		os.Remove(p)
	}
	return err
}

func (cat *localCatalog) unlock() error {
	return os.Remove(cat.lockPath())
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
