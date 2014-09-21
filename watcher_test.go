package notify

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func nonil(err ...error) error {
	for _, err := range err {
		if err != nil {
			return err
		}
	}
	return nil
}

func test(t *testing.T, w Watcher, ei []EventInfo, d time.Duration) {
	done, c, fn := make(chan error), make(chan EventInfo, len(ei)), filepath.WalkFunc(nil)
	walk, exec, cleanup := Tree.Create(t)
	// TODO(rjeczalik): Uncomment it after Recursive.RecursiveWatch is implemented.
	// rw, ok := w.(RecursiveWatcher)
	rw, ok := (RecursiveWatcher)(nil), false
	defer cleanup()
	if ok {
		var once sync.Once
		fn = func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			once.Do(func() {
				err = rw.RecursiveWatch(p, All)
			})
			return nonil(err, filepath.SkipDir)
		}
	} else {
		fn = func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				err = w.Watch(p, All)
			}
			return err
		}
	}
	w.Fanin(c)
	if err := walk(fn); err != nil {
		t.Fatal(err)
	}
	go func() {
		for _, ei := range ei {
			exec(ei)
			if err := equal(<-c, ei); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()
	select {
	case <-time.After(d):
		t.Fatalf("test has timed out after %v", d)
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestRuntimeWatcher(t *testing.T) {
	if runtime == nil || runtime.Watcher == nil {
		t.Skip("no global watcher to test")
	}
	ei := []EventInfo{
		Ev("github.com/rjeczalik/fs/fs_test.go", Create, false),
		Ev("github.com/rjeczalik/fs/binfs", Create, true),
		Ev("github.com/rjeczalik/fs/binfs.go", Create, false),
		Ev("github.com/rjeczalik/fs/binfs_test.go", Create, false),
		Ev("github.com/rjeczalik/fs/binfs", Delete, true),
		Ev("github.com/rjeczalik/fs/binfs", Create, true),
		Ev("github.com/rjeczalik/fs/virfs", Create, false),
		Ev("github.com/rjeczalik/fs/virfs", Delete, false),
		Ev("file", Create, false),
		Ev("dir", Create, true),
	}
	test(t, runtime.Watcher, ei, time.Second)
}
