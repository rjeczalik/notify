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

// TODO(rjeczalik): merge watch and unwatch in one func

func watch(w Watcher, e Event) filepath.WalkFunc {
	if rw, ok := w.(RecursiveWatcher); ok {
		var once sync.Once
		return func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			once.Do(func() {
				err = rw.RecursiveWatch(p, e)
			})
			return nonil(err, filepath.SkipDir)
		}
	}
	return func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			err = w.Watch(p, e)
		}
		return err
	}
}

func unwatch(w Watcher) filepath.WalkFunc {
	if rw, ok := w.(RecursiveWatcher); ok {
		var once sync.Once
		return func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			once.Do(func() {
				err = rw.Unwatch(p, All)
			})
			return nonil(err, filepath.SkipDir)
		}
	}
	return func(p string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			err = w.Unwatch(p, All)
		}
		return err
	}
}

func test(t *testing.T, w Watcher, e Event, ei []EventInfo, d time.Duration) {
	done, c, stop := make(chan error), make(chan EventInfo, len(ei)), make(chan struct{})
	walk, exec, cleanup := fixture.New(t)
	defer func() {
		if err := walk(unwatch(w)); err != nil {
			t.Fatal(err)
		}
		close(stop)
		cleanup()
	}()
	w.Fanin(c, stop)
	if err := walk(watch(w, e)); err != nil {
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

func TestWatcher(t *testing.T) {
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
	test(t, newWatcher(), All, ei, time.Second)
}
