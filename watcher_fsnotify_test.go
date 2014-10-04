// +build fsnotify

package notify_test

import (
	"testing"

	"github.com/rjeczalik/notify"
	"github.com/rjeczalik/notify/test"
)

func TestFsnotify(t *testing.T) {
	t.Skip("Duplicate of TestRuntimeWatcher, it's here for doc purposes")
	ei := []notify.EventInfo{
		test.EI("github.com/rjeczalik/fs/fs_test.go", notify.Create),
		test.EI("github.com/rjeczalik/fs/binfs/", notify.Create),
		test.EI("github.com/rjeczalik/fs/binfs.go", notify.Create),
		test.EI("github.com/rjeczalik/fs/binfs_test.go", notify.Create),
		test.EI("github.com/rjeczalik/fs/binfs/", notify.Delete),
		test.EI("github.com/rjeczalik/fs/binfs/", notify.Create),
		// BUG(OS X): Fsnotify claims, the following is Create not Move.
		// Ev("github.com/rjeczalik/fs/binfs", Move, true),
		test.EI("github.com/rjeczalik/fs/virfs", notify.Create),
		// BUG(OS X): When being watched by fsnotify, writing to the newly-created
		// file fails with "bad file descriptor".
		// Ev("github.com/rjeczalik/fs/virfs", Write, false),
		test.EI("github.com/rjeczalik/fs/virfs", notify.Delete),
		// BUG(OS X): The same as above, this time "bad file descriptor" on a file
		// that was created previously.
		// Ev("github.com/rjeczalik/fs/LICENSE", Write, false),
		// TODO(rjeczalik): For non-recursive watchers directory delete results
		// in lots of delete events of the subtree. Is it possible to "squash"
		// them into one event? How it should work cross-platform?
		// Ev("github.com", Delete, true),
		test.EI("file", notify.Create),
		test.EI("dir/", notify.Create),
	}
	test.ExpectEvent(t, notify.NewWatcher(), notify.All, ei)
}

func TestIssue16(t *testing.T) {
	t.Skip("TODO(#16)")
	ei := []notify.EventInfo{test.EI("github.com/", notify.Delete)}
	test.ExpectEvent(t, notify.NewWatcher(), notify.All, ei)
}
