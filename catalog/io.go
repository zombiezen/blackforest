package catalog

import (
	"encoding/json"
	"io"
	"os"
)

// filesystem is an abstraction for the operating system's filesystem interface.  Useful for mocking.
// All paths should be OS paths (i.e. from path/filepath).
type filesystem interface {
	// Open opens a file for reading.
	Open(path string) (file, error)

	// Create opens a file for writing.
	// If excl is true, then an error will be returned if the file already exists.
	Create(path string, excl bool) (file, error)

	// Remove deletes a file.
	Remove(path string) error

	IsNotExist(e error) bool
	IsExist(e error) bool
}

type file interface {
	io.Reader
	io.Writer
	io.Closer
	Readdir(n int) (fi []os.FileInfo, err error)
	Stat() (os.FileInfo, error)
}

type realFilesystem struct{}

func (realFilesystem) Open(path string) (file, error) { return os.Open(path) }
func (realFilesystem) Remove(path string) error       { return os.Remove(path) }
func (realFilesystem) IsExist(e error) bool           { return os.IsExist(e) }
func (realFilesystem) IsNotExist(e error) bool        { return os.IsNotExist(e) }

func (realFilesystem) Create(path string, excl bool) (file, error) {
	const permMask os.FileMode = 0666
	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	if excl {
		flag |= os.O_EXCL
	}
	return os.OpenFile(path, flag, permMask)
}

func readJSON(fs filesystem, path string, v interface{}) error {
	f, err := fs.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(v)
}

func writeJSON(fs filesystem, path string, v interface{}, excl bool) (retErr error) {
	f, err := fs.Create(path, excl)
	if err != nil {
		return err
	}
	defer func() {
		if err := f.Close(); err != nil && retErr == nil {
			retErr = err
		}
	}()
	if err := json.NewEncoder(f).Encode(v); err != nil {
		return err
	}
	return nil
}
