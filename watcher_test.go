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

func strip(e Event) Event {
	return e & ^Recursive
}

func test(t *testing.T, w Watcher, ev []EventInfo, d time.Duration) {
	done, c, fn := make(chan error), make(chan EventInfo, len(ev)), filepath.WalkFunc(nil)
	walk, exec, cleanup := Tree.Create(t)
	defer cleanup()
	if w.IsRecursive() {
		var once sync.Once
		fn = func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			once.Do(func() {
				err = w.Watch(p, All)
			})
			return nonil(err, filepath.SkipDir)
		}
	} else {
		fn = func(p string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if fi.IsDir() {
				err = w.Watch(p, strip(All))
			}
			return err
		}
	}
	w.Fanin(c)
	if err := walk(fn); err != nil {
		t.Fatal(err)
	}
	go func() {
		exec(ev...)
	}()
	go func() {
		var i int
		for ei := range c {
			if err := equal(ei, ev[i]); err != nil {
				done <- err
				return
			}
			i++
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
	if global.Watcher == nil {
		t.Skip("no watcher to test")
	}
	ev := [...]EventInfo{
		0: Ev("github.com/rjeczalik/fs/fs_test.go", Create, false),
		1: Ev("github.com/rjeczalik/fs/binfs", Create, true),
		2: Ev("github.com/rjeczalik/fs/binfs/binfs.go", Create, false),
		3: Ev("github.com/rjeczalik/fs/binfs/binfs_test.go", Create, false),
		4: Ev("github.com/rjeczalik/fs/binfs", Delete, true),
	}
	test(t, global.Watcher, ev[:], 5*time.Second)
}
