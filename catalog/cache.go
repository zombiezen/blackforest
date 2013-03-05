package catalog

import (
	"sync"
)

// A Cache provides an in-memory cache of another Catalog.
type Cache struct {
	cat Catalog

	m    map[string]Project
	id   map[ID]string
	tags map[string]stringSet
	lock sync.RWMutex
}

// NewCache returns a new Cache given a Catalog.
func NewCache(cat Catalog) (*Cache, error) {
	c := &Cache{cat: cat}
	err := c.RefreshAll()
	return c, err
}

// cache adds a project into the cache indices.  It does not acquire a lock.
func (c *Cache) cache(p *Project) {
	sn := p.ShortName
	c.m[sn] = *p
	c.id[p.ID] = sn
	for _, tag := range p.Tags {
		if set := c.tags[tag]; set == nil {
			c.tags[tag] = stringSet{sn: {}}
		} else {
			set.Add(sn)
		}
	}
}

// uncache removes all occurences of a short name from the indices.  It does not
// acquire a lock.
func (c *Cache) uncache(shortName string) {
	for id, sn := range c.id {
		if sn == shortName {
			delete(c.id, id)
		}
	}
	for _, set := range c.tags {
		set.Remove(shortName)
	}
	delete(c.m, shortName)
}

// RefreshAll purges all keys from the cache and retrieves all the projects from
// the underlying catalog.  Any error encountered in the process will abort the
// refresh.
func (c *Cache) RefreshAll() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	names, err := c.cat.List()
	if err != nil {
		return err
	}
	c.m = make(map[string]Project, len(names))
	c.id = make(map[ID]string, len(names))
	c.tags = make(map[string]stringSet)
	for _, sn := range names {
		p, err := c.cat.GetProject(sn)
		if err != nil {
			return err
		} else if p != nil {
			c.cache(p)
		}
	}
	return nil
}

// Tags returns the list of known tags in the cache.
func (c *Cache) Tags() []string {
	c.lock.RLock()
	defer c.lock.RUnlock()

	tags := make([]string, 0, len(c.tags))
	for tag, set := range c.tags {
		if len(set) > 0 {
			tags = append(tags, tag)
		}
	}
	return tags
}

// FindTag builds a list of all the projects with a tag.
func (c *Cache) FindTag(tag string) []string {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.tags[tag].Slice()
}

// List returns a list of all the project short names in the catalog.
func (c *Cache) List() ([]string, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	names := make([]string, 0, len(c.m))
	for k := range c.m {
		names = append(names, k)
	}
	return names, nil
}

// GetProject fetches the project record with the given short name, or nil if
// the project was not found in the cache.
func (c *Cache) GetProject(shortName string) (*Project, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	p, ok := c.m[shortName]
	if !ok {
		return nil, nil
	}
	return &p, nil
}

// PutProject stores a project record.  If the put fails in the catalog, the
// cache remains unchanged.
func (c *Cache) PutProject(project *Project) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if err := c.cat.PutProject(project); err != nil {
		return err
	}

	if old, ok := c.id[project.ID]; ok {
		c.uncache(old)
	}
	c.cache(project)
	return nil
}

// DelProject removes a project record from the catalog.  If the delete fails in
// the catalog, the cache remains unchanged.
func (c *Cache) DelProject(shortName string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if err := c.cat.DelProject(shortName); err != nil {
		return err
	}
	c.uncache(shortName)
	return nil
}

// ShortName returns the short name for the given ID.  If the ID is not in the
// cache, this method returns an empty string with no error.
func (c *Cache) ShortName(id ID) (string, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	return c.id[id], nil
}

type stringSet map[string]struct{}

func newStringSet(vals []string) stringSet {
	ss := make(stringSet, len(vals))
	for _, v := range vals {
		ss.Add(v)
	}
	return ss
}

func (ss stringSet) Has(s string) bool {
	if ss == nil {
		return false
	}
	_, ok := ss[s]
	return ok
}

func (ss stringSet) Add(s string) {
	ss[s] = struct{}{}
}

func (ss stringSet) Remove(s string) {
	delete(ss, s)
}

func (ss stringSet) Slice() []string {
	if ss == nil {
		return []string{}
	}
	slice := make([]string, 0, len(ss))
	for k := range ss {
		slice = append(slice, k)
	}
	return slice
}
