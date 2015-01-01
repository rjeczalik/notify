package notify

import "testing"

// NOTE Set DEBUG env var for extra debugging info.

func TestWatcher(t *testing.T) {
	w := newWatcherTest(t, "testdata/gopath.txt")
	defer w.Stop()

	w.Expect(create(w, "src/github.com/rjeczalik/which/.which.go.swp"))
	w.Expect(create(w, "src/github.com/rjeczalik/fs/fs_test.go"))
	w.Expect(create(w, "src/github.com/rjeczalik/fs/binfs/"))
	w.Expect(create(w, "src/github.com/rjeczalik/fs/binfs.go"))
	w.Expect(create(w, "src/github.com/rjeczalik/fs/binfs_test.go"))
	w.Expect(remove(w, "src/github.com/rjeczalik/fs/binfs/")) // #53
	w.Expect(create(w, "src/github.com/rjeczalik/fs/binfs/"))
	w.Expect(create(w, "src/github.com/rjeczalik/fs/virfs"))
	w.Expect(remove(w, "src/github.com/rjeczalik/fs/virfs"))
	w.Expect(create(w, "file"))
	w.Expect(create(w, "dir/"))
}

func TestWatcherWrites(t *testing.T) {
	w := newWatcherTest(t, "testdata/gopath.txt")
	defer w.Stop()

	w.Expect(create(w, "src/github.com/rjeczalik/which/which.go"))
	w.Expect(write(w, "src/github.com/rjeczalik/which/which.go", []byte("XD")))
	w.Expect(write(w, "src/github.com/rjeczalik/which/which.go", []byte("XD")))
	w.Expect(remove(w, "src/github.com/rjeczalik/which/which.go"))
	w.Expect(create(w, "src/github.com/rjeczalik/which/which.go"))
	w.Expect(write(w, "src/github.com/rjeczalik/which/which.go", []byte("XD")))
}
