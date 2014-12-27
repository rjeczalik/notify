package notify

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TODO(rjeczalik): add debug printing

func isDir(path string) bool {
	r := path[len(path)-1]
	return r == '\\' || r == '/'
}

func tmpcreate(tmp string, path string) (bool, error) {
	isdir := isDir(path)
	path = filepath.Join(tmp, filepath.FromSlash(path))
	if isdir {
		// Line is a directory.
		if err := os.MkdirAll(path, 0755); err != nil {
			return false, err
		}
	} else {
		// Line is a file.
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return false, err
		}
		f, err := os.Create(path)
		if err != nil {
			return false, err
		}
		if err := nonil(f.Sync(), f.Close()); err != nil {
			return false, err
		}
	}
	return isdir, nil
}

// tmptree TODO
func tmptree(list string) (string, error) {
	f, err := os.Open(list)
	if err != nil {
		return "", err
	}
	defer f.Close()
	tmp, err := ioutil.TempDir("", "tmptree")
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if _, err := tmpcreate(tmp, scanner.Text()); err != nil {
			return "", err
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	Sync()
	return tmp, nil
}

type eventinfo struct {
	sys   interface{}
	path  string
	event Event
	isdir bool
}

func (ei eventinfo) Sys() interface{} { return ei.sys }
func (ei eventinfo) Path() string     { return ei.path }
func (ei eventinfo) Event() Event     { return ei.event }
func (ei eventinfo) IsDir() bool      { return ei.isdir }
func (ei eventinfo) String() string   { return fmt.Sprintf("%s on %q", ei.event, ei.path) }

// W TODO
type W struct {
	// Watcher TODO
	Watcher Watcher

	// C TODO
	C <-chan EventInfo

	// Timeout TODO
	Timeout time.Duration

	t    *testing.T
	root string
}

// newWatcherTest TODO
func newWatcherTest(t *testing.T, tree string) *W {
	root, err := tmptree(filepath.FromSlash(tree))
	if err != nil {
		t.Fatal(err)
	}
	w := &W{
		t:    t,
		root: root,
	}
	if rw, ok := w.watcher().(RecursiveWatcher); ok {
		if err := rw.RecursiveWatch(root, Create|Delete|Move|Write); err != nil {
			t.Fatal(err)
		}
	} else {
		fn := func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				if err := w.watcher().Watch(path, Create|Delete|Move|Write); err != nil {
					return err
				}
			}
			return nil
		}
		if err := filepath.Walk(root, fn); err != nil {
			t.Fatal(err)
		}
	}
	return w
}

// watcher TODO
func (w *W) watcher() Watcher {
	if w.Watcher == nil {
		c := make(chan EventInfo, 128)
		w.Watcher = NewWatcher(c)
		w.C = c
	}
	return w.Watcher
}

// timeout TODO
func (w *W) timeout() time.Duration {
	if w.Timeout != 0 {
		return w.Timeout
	}
	return 2 * time.Second
}

// Stop TODO
func (w *W) Stop() {
	defer os.RemoveAll(w.root)
	// TODO(rjeczalik): make Close part of Watcher interface
	err := w.watcher().(interface {
		Close() error
	}).Close()
	if err != nil {
		w.t.Fatal(err)
	}
}

// create TODO
func create(w *W, path string) (func(), EventInfo) {
	ei := eventinfo{path: path, event: Create}
	fn := func() {
		var err error
		if ei.isdir, err = tmpcreate(w.root, filepath.FromSlash(path)); err != nil {
			w.t.Fatal(err)
		}
		Sync()
	}
	return fn, ei
}

// remove TODO
func remove(w *W, path string) (func(), EventInfo) {
	ei := eventinfo{path: path, event: Delete}
	fn := func() {
		if err := os.RemoveAll(filepath.Join(w.root, filepath.FromSlash(path))); err != nil {
			w.t.Fatal(err)
		}
		Sync()
	}
	return fn, ei
}

// rename TODO
func rename(w *W, oldpath, newpath string) (func(), EventInfo) {
	ei := eventinfo{path: newpath, event: Move}
	fn := func() {
		err := os.Rename(filepath.Join(w.root, filepath.FromSlash(oldpath)),
			filepath.Join(w.root, filepath.FromSlash(newpath)))
		if err != nil {
			w.t.Fatal(err)
		}
		Sync()
	}
	return fn, ei
}

// write TODO
func write(w *W, path string, p []byte) (func(), EventInfo) {
	ei := eventinfo{path: path, event: Write}
	fn := func() {
		perm := os.FileMode(0644)
		if isDir(path) {
			perm = 0755
		}
		err := ioutil.WriteFile(filepath.Join(w.root, filepath.FromSlash(path)), p, perm)
		if err != nil {
			w.t.Fatal(err)
		}
		Sync()
	}
	return fn, ei
}

// Expect TODO
//
// TODO(rjeczalik): refactor, allow for deferred testing
func (w *W) Expect(fn func(), expected EventInfo) {
	fn()
	select {
	case ei := <-w.C:
		if ei.Event() != expected.Event() {
			w.t.Fatalf("want event=%v; got %v", expected.Event(), ei.Event())
		}
		if !strings.HasSuffix(ei.Path(), expected.Path()) {
			w.t.Fatalf("want path=%q; got %q", expected.Path(), ei.Path())
		}
	case <-time.After(w.timeout()):
		w.t.Fatalf("timed out after %v waiting for %v", w.timeout(), expected)
	}
}
