// +build fsnotify

package notify

import (
	"testing"
	"time"
)

func TestFsnotify(t *testing.T) {
	ei := []EventInfo{
		EI("github.com/rjeczalik/fs/fs_test.go", Create),
		EI("github.com/rjeczalik/fs/binfs/", Create),
		EI("github.com/rjeczalik/fs/binfs.go", Create),
		EI("github.com/rjeczalik/fs/binfs_test.go", Create),
		EI("github.com/rjeczalik/fs/binfs/", Delete),
		EI("github.com/rjeczalik/fs/binfs/", Create),
		// BUG(OS X): Fsnotify claims, the following is Create not Move.
		// Ev("github.com/rjeczalik/fs/binfs", Move, true),
		EI("github.com/rjeczalik/fs/virfs", Create),
		// BUG(OS X): When being watched by fsnotify, writing to the newly-created
		// file fails with "bad file descriptor".
		// Ev("github.com/rjeczalik/fs/virfs", Write, false),
		EI("github.com/rjeczalik/fs/virfs", Delete),
		// BUG(OS X): The same as above, this time "bad file descriptor" on a file
		// that was created previously.
		// Ev("github.com/rjeczalik/fs/LICENSE", Write, false),
		// TODO(rjeczalik): For non-recursive watchers directory delete results
		// in lots of delete events of the subtree. Is it possible to "squash"
		// them into one event? How it should work cross-platform?
		// Ev("github.com", Delete, true),
		EI("file", Create),
		EI("dir/", Create),
	}
	test(t, NewWatcher(), All, ei, time.Second)
}

func TestIssue16(t *testing.T) {
	t.Skip("TODO(#16)")
	test(t, NewWatcher(), All, []EventInfo{EI("github.com/", Delete)}, time.Second)
}
