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
	rw, ok := w.(RecursiveWatcher)
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

func TestGlobalWatcher(t *testing.T) {
	if global.Watcher == nil {
		t.Skip("no global watcher to test")
	}
	ei := []EventInfo{
		Ev("github.com/rjeczalik/fs/fs_test.go", Create, false),
		Ev("github.com/rjeczalik/fs/binfs", Create, true),
		Ev("github.com/rjeczalik/fs/binfs.go", Create, false),
		Ev("github.com/rjeczalik/fs/binfs_test.go", Create, false),
		Ev("github.com/rjeczalik/fs/binfs", Delete, true),
		Ev("github.com/rjeczalik/fs/binfs", Create, true),
		// BUG(OS X): Fsnotify claims, the following is Create not Move.
		// Ev("github.com/rjeczalik/fs/binfs", Move, true),
		Ev("github.com/rjeczalik/fs/virfs", Create, false),
		// BUG(OS X): When being watched by fsnotify, writing to the newly-created
		// file fails with "bad file descriptor".
		// Ev("github.com/rjeczalik/fs/virfs", Write, false),
		Ev("github.com/rjeczalik/fs/virfs", Delete, false),
		// BUG(OS X): The same as above, this time "bad file descriptor" on a file
		// that was created previously.
		// Ev("github.com/rjeczalik/fs/LICENSE", Write, false),
		// TODO(rjeczalik): For non-recursive watchers directory delete results
		// in lots of delete events of the subtree. Is it possible to "squash"
		// them into one event? How it should work cross-platform?
		// Ev("github.com", Delete, true),
		Ev("file", Create, false),
		Ev("dir", Create, true),
	}
	test(t, global.Watcher, ei, time.Second)
}

func TestIssue16(t *testing.T) {
	t.Skip("#16")
	test(t, global.Watcher, []EventInfo{Ev("github.com", Delete, true)}, time.Second)
}
