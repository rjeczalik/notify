package notify

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func isDir(path string) bool {
	r := path[len(path)-1]
	return r == '\\' || r == '/'
}

func tmpcreateall(tmp string, path string) error {
	isdir := isDir(path)
	path = filepath.Join(tmp, filepath.FromSlash(path))
	if isdir {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	} else {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		if err := nonil(f.Sync(), f.Close()); err != nil {
			return err
		}
	}
	return nil
}

func tmpcreate(root, path string) (bool, error) {
	isdir := isDir(path)
	path = filepath.Join(root, filepath.FromSlash(path))
	if isdir {
		if err := os.Mkdir(path, 0755); err != nil {
			return false, err
		}
	} else {
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
func tmptree(root, list string) (string, error) {
	f, err := os.Open(list)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if root == "" {
		pwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		if root, err = ioutil.TempDir(pwd, "tmptree"); err != nil {
			return "", err
		}
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err := tmpcreateall(root, scanner.Text()); err != nil {
			return "", err
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	Sync()
	return root, nil
}

// NOTE for debugging only
func TestDump(t *testing.T) {
	var root string
	var args bool
	for _, arg := range os.Args {
		switch {
		case root != "":
			t.Skip("too many arguments")
		case args:
			root = arg
		case arg == "--":
			args = true
		}
	}
	if root == "" {
		t.Skip()
	}
	if _, err := tmptree(root, filepath.Join("testdata", "gopath.txt")); err != nil {
		t.Fatal(err)
	}
	fmt.Println(root)
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
	i    int

	repro []string // NOTE for debugging only
}

// newWatcherTest TODO
func newWatcherTest(t *testing.T, tree string) *W {
	root, err := tmptree("", filepath.FromSlash(tree))
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

// NOTE for debugging only
func (w *W) printrepro() {
	fmt.Println("[D] [WATCHER_TEST] to reproduce manually:")
	fmt.Printf("go test -run TestDump -- %s\n", w.root)
	fmt.Println("cd", w.root)
	for _, repro := range w.repro {
		fmt.Println(repro)
	}
	fmt.Println()
}

func (w *W) Fatal(v interface{}) {
	if dbg {
		w.printrepro()
	}
	w.t.Fatal(v)
}

func (w *W) Fatalf(format string, v ...interface{}) {
	if dbg {
		w.printrepro()
	}
	w.t.Fatalf(format+" (i=%d)", append(v, w.i)...)
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
	if err := w.watcher().(io.Closer).Close(); err != nil {
		w.Fatal(err)
	}
}

// create TODO
func create(w *W, path string) (func(), EventInfo) {
	ei := eventinfo{path: strings.TrimRight(path, `/\`), event: Create}
	fn := func() {
		isdir, err := tmpcreate(w.root, filepath.FromSlash(path))
		if err != nil {
			w.Fatal(err)
		}
		if ei.isdir = isdir; isdir {
			w.repro = append(w.repro, fmt.Sprintf("mkdir %s", path))
		} else {
			w.repro = append(w.repro, fmt.Sprintf("touch %s", path))
		}
	}
	return fn, ei
}

// remove TODO
func remove(w *W, path string) (func(), EventInfo) {
	ei := eventinfo{path: strings.TrimRight(path, `/\`), event: Delete}
	fn := func() {
		if err := os.RemoveAll(filepath.Join(w.root, filepath.FromSlash(path))); err != nil {
			w.Fatal(err)
		}
		w.repro = append(w.repro, fmt.Sprintf("rm -rf %s", path))
	}
	return fn, ei
}

// rename TODO
func rename(w *W, oldpath, newpath string) (func(), EventInfo) {
	ei := eventinfo{path: strings.TrimRight(newpath, `/\`), event: Move}
	fn := func() {
		err := os.Rename(filepath.Join(w.root, filepath.FromSlash(oldpath)),
			filepath.Join(w.root, filepath.FromSlash(newpath)))
		if err != nil {
			w.Fatal(err)
		}
		w.repro = append(w.repro, fmt.Sprintf("mv %s %s", oldpath, newpath))
	}
	return fn, ei
}

// write TODO
func write(w *W, path string, p []byte) (func(), EventInfo) {
	ei := eventinfo{path: strings.TrimRight(path, `/\`), event: Write}
	fn := func() {
		f, err := os.OpenFile(filepath.Join(w.root, filepath.FromSlash(path)),
			os.O_WRONLY, 0644)
		if err != nil {
			w.Fatal(err)
		}
		if _, err := f.Write(p); err != nil {
			w.Fatal(err)
		}
		if err := nonil(f.Sync(), f.Close()); err != nil {
			w.Fatal(err)
		}
		w.repro = append(w.repro, fmt.Sprintf("echo %q > %s", p, path))
	}
	return fn, ei
}

// Expect TODO
//
// TODO(rjeczalik): refactor, allow for deferred testing
func (w *W) Expect(fn func(), expected EventInfo) {
	fn()
	Sync()
	w.i++ // NOTE for debugging only
	select {
	case ei := <-w.C:
		dbg.Printf("[WATCHER_TEST] received event: path=%q, event=%v, isdir=%v, sys=%v",
			ei.Path(), ei.Event(), ei.IsDir(), ei.Sys())
		if ei.Event() != expected.Event() {
			w.Fatalf("want event=%v; got %v (path=%s)", expected.Event(), ei.Event(),
				ei.Path())
		}
		if !strings.HasSuffix(ei.Path(), expected.Path()) {
			w.Fatalf("want path=%s; got %s", expected.Path(), ei.Path())
		}
	case <-time.After(w.timeout()):
		w.Fatalf("timed out after %v waiting for %v", w.timeout(), expected)
	}
}
