// +build fsnotify

package notify

import (
	"testing"
	"time"
)

func TestFsnotify(t *testing.T) {
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
	test(t, newFsnotify(), ei, time.Second)
}

func TestIssue16(t *testing.T) {
	t.Skip("TODO(#16)")
	test(t, runtime.Watcher, []EventInfo{Ev("github.com", Delete, true)}, time.Second)
}
