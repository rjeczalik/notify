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
	return root, nil
}

func TestDebug(t *testing.T) {
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
		t.Fatalf(`want tmptree(%q, "testdata/gopath.txt")=nil; got %v`, root, err)
	}
	fmt.Println(root)
}

type eventinfo struct {
	sys   interface{}
	path  string
	event Event
}

func (ei eventinfo) Sys() interface{} { return ei.sys }
func (ei eventinfo) Path() string     { return strings.TrimRight(ei.path, `/\`) }
func (ei eventinfo) Event() Event     { return ei.event }
func (ei eventinfo) String() string   { return fmt.Sprintf("%s on %q", ei.event, ei.path) }

// WCase TODO
type WCase struct {
	Action func()
	Events []EventInfo
}

// W TODO
type W struct {
	// Watcher TODO
	Watcher Watcher

	// C TODO
	C <-chan EventInfo

	// Timeout TODO
	Timeout time.Duration

	t     *testing.T
	root  string
	debug []string // NOTE for debugging only
}

// NewWatcherTest TODO
func NewWatcherTest(t *testing.T, tree string, events ...Event) *W {
	t.Parallel()
	root, err := tmptree("", filepath.FromSlash(tree))
	if err != nil {
		t.Fatalf(`tmptree("", %q)=%v`, tree, err)
	}
	Sync()
	w := &W{
		t:    t,
		root: root,
	}
	if len(events) == 0 {
		events = []Event{Create, Delete, Write, Move}
	}
	if rw, ok := w.watcher().(RecursiveWatcher); ok {
		if err := rw.RecursiveWatch(root, joinevents(events)); err != nil {
			t.Fatalf("RecursiveWatch(%q, All)=%v", root, err)
		}
	} else {
		fn := func(path string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				if err := w.watcher().Watch(path, joinevents(events)); err != nil {
					return err
				}
			}
			return nil
		}
		if err := filepath.Walk(root, fn); err != nil {
			t.Fatalf("Walk(%q, fn)=%v", root, err)
		}
	}
	return w
}

func (w *W) printdebug() {
	fmt.Println("[D] [WATCHER_TEST] to debugduce manually:")
	fmt.Printf("go test -run TestDebug -- %s\n", w.root)
	fmt.Println("cd", w.root)
	for _, debug := range w.debug {
		fmt.Println(debug)
	}
	fmt.Println()
}

// Fatal TODO
func (w *W) Fatal(v interface{}) {
	if dbg {
		w.printdebug()
	}
	w.t.Fatal(v)
}

// Fatalf TODO
func (w *W) Fatalf(format string, v ...interface{}) {
	if dbg {
		w.printdebug()
	}
	w.t.Fatalf(format, v...)
}

// Debug TODO
func (w *W) Debug(command string) {
	w.debug = append(w.debug, command)
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
		w.Fatalf("w.Watcher.Close()=%v", err)
	}
}

// create TODO
func create(w *W, path string) WCase {
	return WCase{
		Action: func() {
			isdir, err := tmpcreate(w.root, filepath.FromSlash(path))
			if err != nil {
				w.Fatalf("tmpcreate(%q, %q)=%v", w.root, path, err)
			}
			if isdir {
				w.Debug(fmt.Sprintf("mkdir %s", path))
			} else {
				w.Debug(fmt.Sprintf("touch %s", path))
			}
		},
		Events: []EventInfo{
			eventinfo{path: path, event: Create},
		},
	}
}

// remove TODO
func remove(w *W, path string) WCase {
	return WCase{
		Action: func() {
			if err := os.RemoveAll(filepath.Join(w.root, filepath.FromSlash(path))); err != nil {
				w.Fatal(err)
			}
			w.Debug(fmt.Sprintf("rm -rf %s", path))
		},
		Events: []EventInfo{
			eventinfo{path: path, event: Delete},
		},
	}
}

// rename TODO
func rename(w *W, oldpath, newpath string) WCase {
	return WCase{
		Action: func() {
			err := os.Rename(filepath.Join(w.root, filepath.FromSlash(oldpath)),
				filepath.Join(w.root, filepath.FromSlash(newpath)))
			if err != nil {
				w.Fatal(err)
			}
			w.Debug(fmt.Sprintf("mv %s %s", oldpath, newpath))
		},
		Events: []EventInfo{
			eventinfo{path: newpath, event: Move},
		},
	}
}

// write TODO
func write(w *W, path string, p []byte) WCase {
	return WCase{
		Action: func() {
			f, err := os.OpenFile(filepath.Join(w.root, filepath.FromSlash(path)),
				os.O_WRONLY, 0644)
			if err != nil {
				w.Fatalf("OpenFile(%q)=%v", path, err)
			}
			if _, err := f.Write(p); err != nil {
				w.Fatalf("Write(%q)=%v", path, err)
			}
			if err := nonil(f.Sync(), f.Close()); err != nil {
				w.Fatalf("Sync(%q)/Close(%q)=%v", path, path, err)
			}
			w.Debug(fmt.Sprintf("echo %q > %s", p, path))
		},
		Events: []EventInfo{
			eventinfo{path: path, event: Write},
		},
	}
}

// ExpectAny TODO
func (w *W) ExpectAny(cases []WCase) {
Test:
	for i, cas := range cases {
		cas.Action()
		Sync()
		want := cas.Events[0]
		select {
		case ei := <-w.C:
			dbg.Printf("received: path=%q, event=%v, sys=%v (i=%d)", ei.Path(), ei.Event(), ei.Sys(), i)
			for j, want := range cas.Events {
				if ei.Event() != want.Event() {
					dbg.Printf("want event=%v; got %v (path=%s, i=%d, j=%d)",
						want.Event(), ei.Event(), ei.Path(), i, j)
					continue
				}
				if !strings.HasSuffix(ei.Path(), want.Path()) {
					dbg.Printf("want path=%s; got %s (i=%d)", want.Path(), ei.Path(), i)
					continue
				}
				continue Test
			}
		case <-time.After(w.timeout()):
			w.Fatalf("timed out after %v waiting for %v (i=%d)", w.timeout(), want, i)
		}
	}
}
