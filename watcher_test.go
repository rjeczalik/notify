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

func test(t *testing.T, w Watcher, mask Event, ei []EventInfo, d time.Duration) {
	done, c, fn := make(chan error), make(chan EventInfo, len(ei)), filepath.WalkFunc(nil)
	walk, exec, cleanup := fixture.New(t)
	// TODO(rjeczalik): Uncomment it after Recursive.RecursiveWatch is implemented.
	// rw, ok := w.(RecursiveWatcher)
	rw, ok, paths := (RecursiveWatcher)(nil), false, []string{}
	defer func() {
		for _, p := range paths {
			w.Unwatch(p)
		}
		cleanup()
	}()
	if ok {
		var once sync.Once
		fn = func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			once.Do(func() {
				err = rw.RecursiveWatch(p, mask)
				paths = append(paths, p)
			})
			return nonil(err, filepath.SkipDir)
		}
	} else {
		fn = func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				err = w.Watch(p, mask)
				paths = append(paths, p)
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
	if notifier == nil || notifier.Watcher == nil {
		t.Skip("no global watcher to test")
	}
	ei := []EventInfo{
		EI("github.com/rjeczalik/fs/fs_test.go", Create),
		EI("github.com/rjeczalik/fs/binfs/", Create),
		EI("github.com/rjeczalik/fs/binfs.go", Create),
		EI("github.com/rjeczalik/fs/binfs_test.go", Create),
		EI("github.com/rjeczalik/fs/binfs/", Delete),
		EI("github.com/rjeczalik/fs/binfs/", Create),
		EI("github.com/rjeczalik/fs/virfs", Create),
		EI("github.com/rjeczalik/fs/virfs", Delete),
		EI("file", Create),
		EI("dir/", Create),
	}
	test(t, notifier.Watcher, All, ei, time.Second)
}
