package catalog

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func indir(dir, path string) bool {
	return filepath.Dir(filepath.Clean(path)) == filepath.Clean(dir)
}

func TestIndir(t *testing.T) {
	tests := []struct {
		Dir    string
		Path   string
		Expect bool
	}{
		{"foo", "foo", false},
		{"foo", "bar/bar.txt", false},
		{"foo", "foo/bar.txt", true},
	}
	for _, test := range tests {
		if in := indir(test.Dir, test.Path); in != test.Expect {
			t.Errorf("indir(%q, %q) = %t; want %t", test.Dir, test.Path, in, test.Expect)
		}
	}
}

type mockFilesystem struct {
	files map[string][]byte
	dirs  map[string]struct{}
}

func newMockFS() *mockFilesystem {
	return &mockFilesystem{
		files: make(map[string][]byte),
		dirs:  make(map[string]struct{}),
	}
}

func (fs *mockFilesystem) find(path string) (isFile, ok bool) {
	if _, ok := fs.files[path]; ok {
		return true, true
	}
	if _, ok := fs.dirs[path]; ok {
		return false, true
	}
	return false, false
}

func (fs *mockFilesystem) Mkdir(path string) error {
	if _, ok := fs.find(path); ok {
		return &os.PathError{
			Path: path,
			Op:   "mkdir",
			Err:  os.ErrExist,
		}
	}
	// TODO: check for parent directory
	fs.dirs[path] = struct{}{}
	return nil
}

func (fs *mockFilesystem) makeFile(path string, content string) {
	fs.files[path] = []byte(content)
}

func (fs *mockFilesystem) Open(path string) (file, error) {
	if data, ok := fs.files[path]; ok {
		return &mockFile{
			fs:   fs,
			name: path,
			data: data,
		}, nil
	}
	if _, ok := fs.dirs[path]; ok {
		names := make([]string, 0)
		for name := range fs.files {
			if indir(path, name) {
				names = append(names, filepath.Base(name))
			}
		}
		for name := range fs.dirs {
			if indir(path, name) {
				names = append(names, filepath.Base(name))
			}
		}
		return &mockFile{
			fs:   fs,
			name: path,
			dir:  names,
		}, nil
	}
	return nil, &os.PathError{
		Path: path,
		Op:   "open",
		Err:  os.ErrNotExist,
	}
}

func (fs *mockFilesystem) Create(path string, excl bool) (file, error) {
	if isFile, ok := fs.find(path); ok && (excl || !isFile) {
		return nil, &os.PathError{
			Path: path,
			Op:   "open",
			Err:  os.ErrExist,
		}
	}
	return &mockFile{
		fs:   fs,
		name: path,
	}, nil
}

func (fs *mockFilesystem) Remove(path string) error {
	if isFile, ok := fs.find(path); !ok {
		return &os.PathError{
			Path: path,
			Op:   "remove",
			Err:  os.ErrNotExist,
		}
	} else if isFile {
		return &os.PathError{
			Path: path,
			Op:   "remove",
			Err:  os.ErrInvalid,
		}
	}
	return nil
}

func (*mockFilesystem) IsExist(e error) bool {
	return os.IsExist(e)
}

func (*mockFilesystem) IsNotExist(e error) bool {
	return os.IsNotExist(e)
}

func (fs *mockFilesystem) stat(path string) (os.FileInfo, error) {
	if data, ok := fs.files[path]; ok {
		return &mockFileInfo{
			name: filepath.Base(path),
			size: int64(len(data)),
			mode: 0666,
		}, nil
	}
	if _, ok := fs.dirs[path]; ok {
		return &mockFileInfo{
			name: filepath.Base(path),
			mode: os.ModeDir | 0777,
		}, nil
	}
	return nil, &os.PathError{
		Path: path,
		Op:   "stat",
		Err:  os.ErrNotExist,
	}
}

type mockFile struct {
	fs   *mockFilesystem
	name string

	pos  int
	data []byte
	dir  []string
}

func (mf *mockFile) Read(p []byte) (n int, err error) {
	copy(p, mf.data[mf.pos:])
	if n = len(mf.data) - mf.pos; n < len(p) {
		mf.pos = len(mf.data)
		return n, io.EOF
	}
	mf.pos += len(p)
	return len(p), nil
}

func (mf *mockFile) Write(p []byte) (n int, err error) {
	if size := mf.pos + len(p); size < len(mf.data) {
		copy(mf.data[mf.pos:], p)
	} else {
		mf.data = append(mf.data[mf.pos:], p...)
	}
	return len(p), nil
}

func (mf *mockFile) Close() error {
	if mf.dir == nil {
		mf.fs.files[mf.name] = mf.data
	}
	return nil
}

func (mf *mockFile) Readdir(n int) (fi []os.FileInfo, err error) {
	if n > 0 && len(mf.dir) == 0 {
		return []os.FileInfo{}, io.EOF
	}
	if n <= 0 || n > len(mf.dir) {
		n = len(mf.dir)
	}
	fi = make([]os.FileInfo, n)
	for i, name := range mf.dir {
		fi[i], err = mf.fs.stat(filepath.Join(mf.name, name))
		if err != nil {
			return
		}
	}
	mf.dir = mf.dir[n:]
	return
}

func (mf *mockFile) Stat() (os.FileInfo, error) {
	return mf.fs.stat(mf.name)
}

type mockFileInfo struct {
	name string
	size int64
	mode os.FileMode
}

func (info *mockFileInfo) String() string {
	return fmt.Sprint(*info)
}

func (info *mockFileInfo) Name() string {
	return info.name
}

func (info *mockFileInfo) Size() int64 {
	return info.size
}

func (info *mockFileInfo) Mode() os.FileMode {
	return info.mode
}

func (info *mockFileInfo) ModTime() time.Time {
	return time.Unix(0, 0)
}

func (info *mockFileInfo) IsDir() bool {
	return info.mode.IsDir()
}

func (info *mockFileInfo) Sys() interface{} {
	return nil
}

func TestMockFile(t *testing.T) {
	const message = "Hello, World!\n"
	fs := newMockFS()
	{
		f, err := fs.Create("foo.txt", false)
		if err != nil {
			t.Fatal("create error:", err)
		}
		if _, err := f.Write([]byte(message)); err != nil {
			t.Error("write error:", err)
		}
		if err := f.Close(); err != nil {
			t.Error("write close error:", err)
		}
	}
	{
		f, err := fs.Open("foo.txt")
		if err != nil {
			t.Fatal("open error:", err)
		}
		data, err := ioutil.ReadAll(f)
		if err != nil {
			t.Error("read error:", err)
		}
		if string(data) != message {
			t.Errorf("data = %q; want %q", string(data), message)
		}
		if err := f.Close(); err != nil {
			t.Error("read close error:", err)
		}
	}
}

func TestMockFS_Dir(t *testing.T) {
	const message = "Hello, World!\n"

	fs := newMockFS()
	if err := fs.Mkdir("foo"); err != nil {
		t.Fatal("mkdir error:", err)
	}

	{
		tests := []struct {
			N   int
			Err error
		}{
			{0, nil},
			{-1, nil},
			{1, io.EOF},
		}
		for _, test := range tests {
			dir, err := fs.Open("foo")
			if err != nil {
				t.Error("open empty dir:", err)
			}
			fi, err := dir.Readdir(test.N)
			if want := []os.FileInfo{}; !reflect.DeepEqual(fi, want) {
				t.Errorf("empty dir read(%d) = %v; want %v", test.N, fi, want)
			}
			if err != test.Err {
				t.Errorf("empty dir read(%d) error = %v; want %v", test.N, err, test.Err)
			}
			if err := dir.Close(); err != nil {
				t.Errorf("close empty dir: %v", err)
			}
		}
	}

	const filename = "foo" + string(filepath.Separator) + "bar.txt"
	{
		f, err := fs.Create(filename, false)
		if err != nil {
			t.Error("create error:", err)
		}
		if _, err := f.Write([]byte(message)); err != nil {
			t.Error("write error:", err)
		}
		if err := f.Close(); err != nil {
			t.Error("write close error:", err)
		}
	}

	{
		f, err := fs.Open(filename)
		if err != nil {
			t.Error("open error:", err)
		}
		data, err := ioutil.ReadAll(f)
		if err != nil {
			t.Error("read error:", err)
		}
		if string(data) != message {
			t.Errorf("data = %q; want %q", string(data), message)
		}
		if err := f.Close(); err != nil {
			t.Error("read close error:", err)
		}
	}

	{
		fi := &mockFileInfo{name: "bar.txt", size: int64(len(message)), mode: 0666}
		tests := [][]struct {
			N    int
			Info []os.FileInfo
			Err  error
		}{
			{{0, []os.FileInfo{fi}, nil}},
			{{-1, []os.FileInfo{fi}, nil}},
			{{1, []os.FileInfo{fi}, nil}, {1, []os.FileInfo{}, io.EOF}},
			{{2, []os.FileInfo{fi}, nil}, {2, []os.FileInfo{}, io.EOF}},
			{{2, []os.FileInfo{fi}, nil}, {2, []os.FileInfo{}, io.EOF}},
		}
		for _, test := range tests {
			dir, err := fs.Open("foo")
			if err != nil {
				t.Error("open 1-file dir:", err)
			}
			for i, call := range test {
				fi, err := dir.Readdir(call.N)
				if !reflect.DeepEqual(fi, call.Info) {
					t.Errorf("1-file read(%d)[%d] = %v; want %v", call.N, i, fi, call.Info)
				}
				if err != call.Err {
					t.Errorf("1-file read(%d)[%d] error = %v; want %v", call.N, i, err, call.Err)
				}
				if err := dir.Close(); err != nil {
					t.Errorf("1-file close dir: %v", err)
				}
			}
		}
	}
}
